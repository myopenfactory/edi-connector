package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/twitchtv/twirp"

	pb "github.com/myopenfactory/client/api"

	"github.com/myopenfactory/client/pkg/errors"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/transport"
	"github.com/myopenfactory/client/pkg/transport/file"
	"github.com/myopenfactory/client/pkg/version"
)

// Config configures variables for the client
type Client struct {
	logger         *log.Logger
	username       string
	password       string
	url            string
	id             string
	runWaitTime    time.Duration
	healthWaitTime time.Duration
	client         *http.Client
	service        pb.ClientService
	requestContext context.Context

	certificate *tls.Certificate
	certPool    *x509.CertPool

	// plugins
	inbounds  map[string]transport.InboundPlugin
	outbounds map[string]transport.OutboundPlugin
}

type Option func(*Client)

// New creates client with given options
func New(options ...Option) (*Client, error) {
	const op errors.Op = "client.New"
	c := &Client{
		logger:         log.New(),
		id:             fmt.Sprintf("Core_%s", version.Version),
		requestContext: context.Background(),
		runWaitTime:    time.Minute,
		healthWaitTime: 15 * time.Minute,
		url:            "https://myopenfactory.net",
		client:         http.DefaultClient,
	}
	for _, option := range options {
		option(c)
	}

	if c.username != "" || c.password != "" {
		auth := []byte(c.username + ":" + c.password)
		header := make(http.Header)
		header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(auth))

		var err error
		c.requestContext, err = twirp.WithHTTPRequestHeaders(context.Background(), header)
		if err != nil {
			return nil, errors.E(op, fmt.Errorf("failed to set authorization context: %w", err))
		}
	}

	if c.certificate != nil {
		config := c.getTLSConfig()
		config.Certificates = []tls.Certificate{*c.certificate}
	}

	if c.certPool != nil {
		config := c.getTLSConfig()
		config.RootCAs = c.certPool
		config.BuildNameToCertificate()
	}

	c.service = pb.NewClientServiceProtobufClient(c.url, c.client, twirp.WithClientPathPrefix("/v1"))

	ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
	defer cancel()

	configs, err := c.service.ListConfigs(ctx, &pb.Empty{})
	if err != nil {
		return nil, errors.E(op, fmt.Errorf("failed to retrieve configs: %w", err))
	}

	c.inbounds = make(map[string]transport.InboundPlugin)
	c.outbounds = make(map[string]transport.OutboundPlugin)
	for _, pc := range configs.Inbound {
		switch pc.Type {
		case "FILE":
			c.inbounds[pc.ProcessId], err = file.NewInboundPlugin(c.logger, pc.Parameter)
			if err != nil {
				return nil, errors.E(op, fmt.Errorf("failed to load plugin: processid: %v: %w", pc.ProcessId, err))
			}
		}
	}
	for _, pc := range configs.Outbound {
		switch pc.Type {
		case "FILE":
			c.outbounds[pc.ProcessId], err = file.NewOutboundPlugin(c.logger, pc.ProcessId, c.service.AddMessage, c.service.AddAttachment, pc.Parameter)
			if err != nil {
				return nil, errors.E(op, fmt.Errorf("failed to load plugin: processid: %v: %w", pc.ProcessId, err))
			}
		}
	}

	return c, nil
}

func WithLogger(logger *log.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

func WithUsername(username string) Option {
	return func(c *Client) {
		c.username = username
	}
}

func WithPassword(password string) Option {
	return func(c *Client) {
		c.password = password
	}
}

func WithURL(url string) Option {
	return func(c *Client) {
		c.url = url
	}
}

func WithRunWaitTime(duration time.Duration) Option {
	return func(c *Client) {
		c.runWaitTime = duration
	}
}

func WithHealthWaitTime(duration time.Duration) Option {
	return func(c *Client) {
		c.healthWaitTime = duration
	}
}

func WithMTLS(cert tls.Certificate) Option {
	return func(c *Client) {
		c.certificate = &cert
	}
}

func WithCertPool(pool *x509.CertPool) Option {
	return func(c *Client) {
		c.certPool = pool
	}
}

func WithProxy(proxy string) Option {
	return func(c *Client) {
		os.Setenv("HTTP_PROXY", proxy)
	}
}

func (c *Client) Health(ctx context.Context) error {
	const op errors.Op = "healthClient.Run"

	var notAfter time.Time
	for _, certbytes := range c.certificate.Certificate {
		x509Cert, err := x509.ParseCertificate(certbytes)
		if err != nil {
			return fmt.Errorf("failed to extract notAfter: %w", err)
		}
		if x509Cert.IsCA {
			continue
		}
		notAfter = x509Cert.NotAfter
	}

	c.logger.Infof("using runwaittime=%s and healthwaittime=%s", c.runWaitTime, c.healthWaitTime)

	ticker := time.NewTicker(c.healthWaitTime)
	start := time.Now()
	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
			_, err := c.service.AddHealth(ctx, &pb.HealthInfo{
				Cpu:     float64(runtime.NumCPU()),
				Ram:     float64(m.Alloc) / 1048576.0, //Megabyte
				Status:  fmt.Sprintf("Version: %s | CertValidUntil: %s", c.id, notAfter.Format("2006-01-02")),
				Threads: uint32(runtime.NumGoroutine()),
				Uptime:  uint64(time.Since(start).Nanoseconds()),
				Os:      fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
			})
			cancel()
			if err != nil {
				err = fmt.Errorf("error sending health information: %w", err)
				c.logger.Error(errors.E(op, err))
				continue
			}
			c.logger.Infof("sent health information to remote endpoint")
		case <-ctx.Done():
			return nil
		}
	}
}

// Runs client until context is closed
func (c *Client) Run(ctx context.Context) error {
	const op errors.Op = "client.Run"

	ticker := time.NewTicker(c.runWaitTime)
	for {
		select {
		case <-ticker.C:
			for _, plugin := range c.outbounds {
				ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
				attachments, err := plugin.ListAttachments(ctx)
				if err != nil {
					c.logger.Errorf("error while reading attachment: %v", err)
				}
				cancel()

				for _, atc := range attachments {
					ctx, cancel := context.WithTimeout(c.requestContext, 5*time.Second)
					if _, err := plugin.ProcessAttachment(ctx, atc); err != nil {
						c.logger.Errorf("error while processing attachment %v: %v", atc.Filename, err)
					}
					cancel()
				}

				ctx, cancel = context.WithTimeout(c.requestContext, 15*time.Second)
				messages, err := plugin.ListMessages(ctx)
				if err != nil {
					c.logger.Errorf("error while reading messages: %v", err)
				}
				cancel()

				for _, msg := range messages {
					ctx, cancel = context.WithTimeout(c.requestContext, 15*time.Second)
					if _, err := plugin.ProcessMessage(ctx, msg); err != nil {
						c.logger.Errorf("error while processing message %v: %v", msg.Id, err)
					}
					cancel()
				}
			}

			ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
			msgs, err := c.service.ListMessages(ctx, &pb.Empty{})
			cancel()
			if err != nil {
				c.logger.Infof("failed listing remote messages: %v", err)
				continue
			}
			for _, msg := range msgs.Messages {
				p, ok := c.inbounds[msg.ProcessId]
				if !ok {
					confirm, err := transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "no process configured: %v", msg.ProcessId)
					if err != nil {
						c.logger.Errorf("error while creating confirm: %v", err)
						continue
					}
					ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
					_, err = c.service.ConfirmMessage(ctx, confirm)
					if err != nil {
						c.logger.Errorf("error while sending confirm: %v", err)
					}
					cancel()
					continue
				}
				for _, attachment := range msg.Attachments {
					data, filename, err := downloadAttachment(attachment)
					if err != nil {
						confirm, err := transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "error while processing inbound attachment: %v", err)
						if err != nil {
							c.logger.Errorf("error while creating confirm: %v", err)
							continue
						}
						ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
						_, err = c.service.ConfirmMessage(ctx, confirm)
						if err != nil {
							c.logger.Errorf("error while sending confirm: %v", err)
						}
						cancel()
						continue
					}

					attachment.Content = &pb.Attachment_Data{
						Data: data,
					}
					attachment.Filename = strings.Replace(attachment.Filename, "%(REALFILENAME)", filename, -1)
					ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
					p.ProcessAttachment(ctx, attachment)
					cancel()
				}

				ctx, cancel := context.WithTimeout(c.requestContext, 15*time.Second)
				confirm, err := p.ProcessMessage(ctx, msg)
				if err != nil {
					confirm, err = transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "error while processing inbound msg: %v", err)
					if err != nil {
						c.logger.Errorf("error while creating confirm: %v", err)
						continue
					}
				}
				cancel()

				ctx, cancel = context.WithTimeout(c.requestContext, 15*time.Second)
				_, err = c.service.ConfirmMessage(ctx, confirm)
				if err != nil {
					c.logger.Errorf("error while sending confirm for inbound msg %s: %v", msg.Id, err)
				}
				cancel()
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func downloadAttachment(attachment *pb.Attachment) ([]byte, string, error) {
	const op errors.Op = "client.downloadAttachment"
	url := attachment.GetUrl()
	if url == "" {
		return nil, "", errors.E(op, fmt.Errorf("attachment url couldn't be empty"))
	}
	if attachment.Filename == "" {
		return nil, "", errors.E(op, fmt.Errorf("error no filename found for attachment %s", attachment.Filename))
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", errors.E(op, fmt.Errorf("error while loading attachment with url %q: %w", attachment.GetUrl(), err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.E(op, fmt.Errorf("error bad response for attachment %q: %w", attachment.GetUrl(), err))
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return nil, "", errors.E(op, fmt.Errorf("error while parsing header for attachment %q: %w", attachment.GetUrl(), err))
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.E(op, fmt.Errorf("error while writing response data for attachment %q: %w", attachment.GetUrl(), err))
	}
	return data, params["filename"], nil
}

func processMessage() {
}

func (c *Client) getTLSConfig() *tls.Config {
	if c.client.Transport == nil {
		c.client.Transport = &http.Transport{}
	}
	config := c.client.Transport.(*http.Transport).TLSClientConfig
	if config == nil {
		config = &tls.Config{}
		c.client.Transport.(*http.Transport).TLSClientConfig = config
	}
	return config
}
