package client

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/twitchtv/twirp"

	pb "github.com/myopenfactory/client/api"

	"github.com/myopenfactory/client/pkg/errors"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/transport"
	"github.com/myopenfactory/client/pkg/transport/file"
)

// Config configures variables for the client
type Client struct {
	logger              *log.Logger
	Username            string // Username for the plattform
	Password            string // Password for teh plattform
	URL                 string // URL of the plattform https://myopenfactory.net/pb/ for example
	ctx                 context.Context
	cancel              context.CancelFunc
	ID                  string
	RunWaitTime         time.Duration
	HealthWaitTime      time.Duration
	CertificateNotAfter time.Time
	done                chan struct{}
	mu                  sync.Mutex // guards done
	client              pb.HTTPClient
	ticker              *time.Ticker
}

type Option func(*Client)

// New creates client with given options
func New(logger *log.Logger, identifier string, options ...Option) (*Client, error) {
	const op errors.Op = "client.New"
	c := &Client{
		logger: logger,
	}
	for _, option := range options {
		option(c)
	}
	c.ID = identifier
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if c.client == nil {
		return nil, errors.E(op, "no http client set", errors.KindUnexpected)
	}
	return c, nil
}

func WithUsername(username string) Option {
	return func(c *Client) {
		c.Username = username
	}
}

func WithPassword(password string) Option {
	return func(c *Client) {
		c.Password = password
	}
}

func WithURL(url string) Option {
	return func(c *Client) {
		c.URL = url
	}
}

func WithRunWaitTime(duration time.Duration) Option {
	return func(c *Client) {
		c.RunWaitTime = duration
	}
}

func WithHealthWaitTime(duration time.Duration) Option {
	return func(c *Client) {
		c.HealthWaitTime = duration
	}
}

func WithClient(client pb.HTTPClient) Option {
	return func(c *Client) {
		c.client = client
	}
}

func WithProxy(proxy string) Option {
	return func(c *Client) {
		os.Setenv("HTTP_PROXY", proxy)
	}
}

// Runs client until context is closed
func (c *Client) Run() error {
	const op errors.Op = "client.Run"
	start := time.Now()
	if err := checkParams(c); err != nil {
		return errors.E(op, err)
	}

	clientpb := pb.NewClientServiceProtobufClient(c.URL, c.client)

	header := make(http.Header)
	auth := []byte(c.Username + ":" + c.Password)
	header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(auth))
	reqCxt, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		return errors.E(op, fmt.Errorf("failed to set authorization header: %w", err))
	}

	configs, err := clientpb.ListConfigs(reqCxt, &pb.Empty{})
	if err != nil {
		return errors.E(op, fmt.Errorf("failed to retrieve configs: %w", err))
	}

	inPP := make(map[string]transport.InboundPlugin)
	outPP := make(map[string]transport.OutboundPlugin)
	for _, pc := range configs.Inbound {
		switch pc.Type {
		case "FILE":
			inPP[pc.ProcessId], err = file.NewInboundPlugin(c.logger, pc.Parameter)
			if err != nil {
				return errors.E(op, fmt.Errorf("failed to load plugin: processid: %v: %w", pc.ProcessId, err))
			}
		}
	}
	for _, pc := range configs.Outbound {
		switch pc.Type {
		case "FILE":
			outPP[pc.ProcessId], err = file.NewOutboundPlugin(c.logger, pc.ProcessId, clientpb.AddMessage, clientpb.AddAttachment, pc.Parameter)
			if err != nil {
				return errors.E(op, fmt.Errorf("failed to load plugin: processid: %v: %w", pc.ProcessId, err))
			}
		}
	}

	c.logger.Infof("using runwaittime=%s and healthwaittime=%s", c.RunWaitTime, c.HealthWaitTime)

	healthTicker := time.NewTicker(c.HealthWaitTime)
	go func() {
		certs := c.client.(*http.Client).Transport.(*http.Transport).TLSClientConfig.Certificates
		if len(certs) == 0 {
			c.logger.Errorf("loading client cert notAfter din't work: %v", err)
			os.Exit(1)
		}

		var notAfter time.Time
		for _, certbytes := range certs[0].Certificate {
			x509Cert, err := x509.ParseCertificate(certbytes)
			if err != nil {
				c.logger.Errorf("faild to load certificate: %v", err)
				os.Exit(1)
			}
			if x509Cert.IsCA {
				continue
			}
			notAfter = x509Cert.NotAfter
		}

		for range healthTicker.C {
			sendHealthInformation(c.logger, reqCxt, clientpb, c.ID, start, notAfter)
		}
	}()
	defer healthTicker.Stop()

	c.ticker = time.NewTicker(c.RunWaitTime)
	for {
		select {
		case <-c.ticker.C:
			for _, plugin := range outPP {
				attachments, err := plugin.ListAttachments(reqCxt)
				if err != nil {
					c.logger.Errorf("error while reading attachment: %v", err)
				}

				for _, atc := range attachments {
					if _, err := plugin.ProcessAttachment(reqCxt, atc); err != nil {
						c.logger.Errorf("error while processing attachment %v: %v", atc.Filename, err)
					}
				}

				messages, err := plugin.ListMessages(reqCxt)
				if err != nil {
					c.logger.Errorf("error while reading messages: %v", err)
				}

				for _, msg := range messages {
					if _, err := plugin.ProcessMessage(reqCxt, msg); err != nil {
						c.logger.Errorf("error while processing message %v: %v", msg.Id, err)
					}
				}
			}

			msgs, err := clientpb.ListMessages(reqCxt, &pb.Empty{})
			if err != nil {
				c.logger.Infof("failed listing remote messages: %v", err)
				continue
			}
			for _, msg := range msgs.Messages {
				p, ok := inPP[msg.ProcessId]
				if !ok {
					confirm, err := transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "no process configured: %v", msg.ProcessId)
					if err != nil {
						c.logger.Errorf("error while creating confirm: %v", err)
						continue
					}
					_, err = clientpb.ConfirmMessage(reqCxt, confirm)
					if err != nil {
						c.logger.Errorf("error while sending confirm: %v", err)
					}
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
						_, err = clientpb.ConfirmMessage(reqCxt, confirm)
						if err != nil {
							c.logger.Errorf("error while sending confirm: %v", err)
						}
						continue
					}

					attachment.Content = &pb.Attachment_Data{
						Data: data,
					}
					attachment.Filename = strings.Replace(attachment.Filename, "%(REALFILENAME)", filename, -1)
					p.ProcessAttachment(reqCxt, attachment)
				}

				confirm, err := p.ProcessMessage(reqCxt, msg)
				if err != nil {
					confirm, err = transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "error while processing inbound msg: %v", err)
					if err != nil {
						c.logger.Errorf("error while creating confirm: %v", err)
						continue
					}
				}
				_, err = clientpb.ConfirmMessage(reqCxt, confirm)
				if err != nil {
					c.logger.Errorf("error while sending confirm for inbound msg %s: %v", msg.Id, err)
				}
			}
		case <-c.ctx.Done():
			c.mu.Lock()
			c.done <- struct{}{}
			c.mu.Unlock()
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

func sendHealthInformation(logger *log.Logger, ctx context.Context, srv pb.ClientService, id string, start time.Time, notAfter time.Time) {
	const op errors.Op = "client.sendHealthInformation"
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	_, err := srv.AddHealth(ctx, &pb.HealthInfo{
		Cpu:     float64(runtime.NumCPU()),
		Ram:     float64(m.Alloc) / 1048576.0, //Megabyte
		Status:  fmt.Sprintf("Version: %s | CertValidUntil: %s", id, notAfter.Format("2006-01-02")),
		Threads: uint32(runtime.NumGoroutine()),
		Uptime:  uint64(time.Since(start).Nanoseconds()),
		Os:      fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
	})
	if err != nil {
		logger.SystemErr(errors.E(op, fmt.Errorf("error sending health information: %w", err)))
		return
	}
	logger.Infof("sent health information to remote endpoint")
}

func (c *Client) Shutdown(ctx context.Context) error {
	const op errors.Op = "client.Shutdown"
	c.cancel()
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		// return errors.New("server shutdown failed: timeout")
		return errors.E(op, "server shutdown failed: timeout")
	}
	return nil
}

func processMessage() {
}

func checkParams(c *Client) error {
	const op errors.Op = "client.checkParams"
	if c == nil {
		return errors.E(op, "client not valid", errors.KindBadRequest)
	}
	if c.ID == "" {
		return errors.E(op, "missing id", errors.KindBadRequest)
	}
	if c.Username == "" {
		return errors.E(op, "missing username", errors.KindBadRequest)
	}
	if c.Password == "" {
		return errors.E(op, "missing password", errors.KindBadRequest)
	}
	if c.URL == "" {
		return errors.E(op, "missing url", errors.KindBadRequest)
	}
	c.mu.Lock()
	if c.done == nil {
		c.done = make(chan struct{}, 1)
	}
	c.mu.Unlock()
	if c.client == nil {
		c.client = &http.Client{}
	}
	return nil
}
