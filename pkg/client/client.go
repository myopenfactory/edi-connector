package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"os"

	"github.com/pkg/errors"
	"github.com/twitchtv/twirp"

	pb "github.com/myopenfactory/client/api"

	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/transport"
	"github.com/myopenfactory/client/pkg/transport/file"
)

var (
	defaultRunWaitTime    = time.Minute
	defaultHealthWaitTime = 15 * time.Minute
)

// Config configures variables for the client
type Client struct {
	Username       string // Username for the plattform
	Password       string // Password for teh plattform
	URL            string // URL of the plattform https://myopenfactory.net/pb/ for example
	ClientCert     string // User client certificate in pem format
	CA             string // ca file for connections to the plattform
	ctx            context.Context
	cancel         context.CancelFunc
	ID             string
	RunWaitTime    time.Duration
	HealthWaitTime time.Duration
	done           chan struct{}
	mu             sync.Mutex // guards done
	client         pb.HTTPClient
	ticker         *time.Ticker
}

type Option func(*Client)

// New creates client with given options
func New(identifier string, options ...Option) (*Client, error) {
	c := &Client{
		RunWaitTime:    defaultRunWaitTime,
		HealthWaitTime: defaultHealthWaitTime,
	}
	for _, option := range options {
		option(c)
	}
	c.ID = identifier
	c.ctx, c.cancel = context.WithCancel(context.Background())
	if c.client == nil {
		var err error
		c.client, err = createHTTPClient(c.ClientCert, c.CA)
		if err != nil {
			return nil, errors.Wrap(err, "http client creation failed")
		}
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

func WithCert(cert string) Option {
	return func(c *Client) {
		c.ClientCert = cert
	}
}

func WithCA(ca string) Option {
	return func(c *Client) {
		c.CA = ca
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

// Runs client until context is closed
func (c *Client) Run() error {
	start := time.Now()
	if err := checkParams(c); err != nil {
		return err
	}

	clientpb := pb.NewClientServiceProtobufClient(c.URL, c.client)

	header := make(http.Header)
	auth := []byte(c.Username + ":" + c.Password)
	header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(auth))
	reqCxt, err := twirp.WithHTTPRequestHeaders(context.Background(), header)
	if err != nil {
		return errors.Wrapf(err, "failed to set authorization header")
	}

	configs, err := clientpb.ListConfigs(reqCxt, &pb.Empty{})
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve configs")
	}

	inPP := make(map[string]transport.InboundPlugin)
	outPP := make(map[string]transport.OutboundPlugin)
	for _, pc := range configs.Inbound {
		switch pc.Type {
		case "FILE":
			inPP[pc.ProcessId], err = file.NewInboundPlugin(pc.Parameter)
			if err != nil {
				return errors.Wrapf(err, "error while loading plugin from processid %s", pc.ProcessId)
			}
		}
	}
	for _, pc := range configs.Outbound {
		switch pc.Type {
		case "FILE":
			outPP[pc.ProcessId], err = file.NewOutboundPlugin(pc.ProcessId, clientpb.AddMessage, clientpb.AddAttachment, pc.Parameter)
			if err != nil {
				return errors.Wrapf(err, "error while loading plugin from processid %s", pc.ProcessId)
			}
		}
	}

	log.Infof("using runwaittime=%s and healthwaittime=%s", c.RunWaitTime, c.HealthWaitTime)

	healthTicker := time.NewTicker(c.HealthWaitTime)
	go func() {
		cc, err := loadKeyPair(c.ClientCert)
		if err != nil {
			log.Errorf("loading client cert din't work: %v", err)
			os.Exit(1)
		}
		var notAfter time.Time
		for _, certbytes := range cc.Certificate {
			x509Cert, err := x509.ParseCertificate(certbytes)
			if err != nil {
				log.Errorf("faild to load certificate: %v", err)
				os.Exit(1)
			}
			if x509Cert.IsCA {
				continue
			}
			notAfter = x509Cert.NotAfter
		}
		for range healthTicker.C {
			sendHealthInformation(reqCxt, clientpb, c.ID, start, notAfter)
		}
	}()
	defer healthTicker.Stop()

	c.ticker = time.NewTicker(c.RunWaitTime)
	for {
		select {
		case <-c.ticker.C:
			for _, plugin := range outPP {
				messages, err := plugin.ListMessages(reqCxt)
				if err != nil {
					log.Errorf("error while reading messages: %v", err)
				}

				for _, msg := range messages {
					if _, err := plugin.ProcessMessage(reqCxt, msg); err != nil {
						log.Errorf("error while processing message %v: %v", msg.Id, err)
					}
				}

				attachments, err := plugin.ListAttachments(reqCxt)
				if err != nil {
					log.Errorf("error while reading attachment: %v", err)
				}

				for _, atc := range attachments {
					if _, err := plugin.ProcessAttachment(reqCxt, atc); err != nil {
						log.Errorf("error while processing attachment %v: %v", atc.Filename, err)
					}
				}
			}

			msgs, err := clientpb.ListMessages(reqCxt, &pb.Empty{})
			if err != nil {
				if err != io.EOF {
					log.Errorf("error while loading remote files: %v", err)
				}
				continue
			}
			for _, msg := range msgs.Messages {
				p, ok := inPP[msg.ProcessId]
				if !ok {
					confirm, err := transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "no process configured: %v", msg.ProcessId)
					if err != nil {
						log.Errorf("error while creating confirm: %v", err)
						continue
					}
					_, err = clientpb.ConfirmMessage(reqCxt, confirm)
					if err != nil {
						log.Errorf("error while sending confirm: %v", err)
					}
					continue
				}
				for _, attachment := range msg.Attachments {
					data, filename, err := downloadAttachment(attachment)
					if err != nil {
						confirm, err := transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusInternalError, "error while processing inbound attachment: %v", err)
						if err != nil {
							log.Errorf("error while creating confirm: %v", err)
							continue
						}
						_, err = clientpb.ConfirmMessage(reqCxt, confirm)
						if err != nil {
							log.Errorf("error while sending confirm: %v", err)
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
						log.Errorf("error while creating confirm: %v", err)
						continue
					}
				}
				_, err = clientpb.ConfirmMessage(reqCxt, confirm)
				if err != nil {
					log.Errorf("error while sending confirm for inbound msg %s: %v", msg.Id, err)
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
	url := attachment.GetUrl()
	if url == "" {
		return nil, "", fmt.Errorf("attachment url couldn't be empty")
	}
	if attachment.Filename == "" {
		return nil, "", fmt.Errorf("error no filename found for attachment %s", attachment.Filename)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error while loading attachment with url %q", attachment.GetUrl())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.Wrapf(err, "error bad response for attachment %q", attachment.GetUrl())
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return nil, "", errors.Wrapf(err, "error while parsing header for attachment %q", attachment.GetUrl())
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.Wrapf(err, "error while writing response data for attachment %q", attachment.GetUrl())
	}
	return data, params["filename"], nil
}

func sendHealthInformation(ctx context.Context, srv pb.ClientService, id string, start time.Time, notAfter time.Time) {

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
		log.Errorf("error sending health information to remote endpoint: %v", err)
		return
	}
	log.Infof("sent health information to remote endpoint")
}

func (c *Client) Shutdown(ctx context.Context) error {
	c.cancel()
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return errors.New("server shutdown failed: timeout")
	}
	return nil
}

func processMessage() {
}

func createHTTPClient(cert, ca string) (*http.Client, error) {
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	cc, err := loadKeyPair(cert)
	if err != nil {
		return nil, errors.Wrapf(err, "error while loading client certificate")
	}
	if len(cc.Certificate) > 0 {
		tlsConfig.Certificates = append(tlsConfig.Certificates, cc)
	}

	if ca != "" {
		tlsConfig.RootCAs, err = loadCertPool(ca)
		if err != nil {
			return nil, err
		}
		tlsConfig.BuildNameToCertificate()
	}

	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           http.ProxyFromEnvironment,
	}
	return &http.Client{
		Transport: tr,
	}, nil
}

func loadCertPool(capem string) (*x509.CertPool, error) {
	certs := x509.NewCertPool()
	if capem == "" {
		return certs, nil
	}

	var pemData []byte
	var err error
	if strings.Contains(capem, "-----BEGIN CERTIFICATE-----") {
		pemData = []byte(capem)
	} else {
		pemData, err = ioutil.ReadFile(capem)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error while loading ca certificates")
	}
	certs.AppendCertsFromPEM(pemData)
	return certs, nil
}

func loadKeyPair(cert string) (tls.Certificate, error) {
	if cert == "" {
		return tls.Certificate{}, nil
	}
	if strings.Contains(cert, "-----BEGIN RSA PRIVATE KEY-----") {
		return tls.X509KeyPair([]byte(cert), []byte(cert))
	} else {
		return tls.LoadX509KeyPair(cert, cert)
	}
}

func checkParams(c *Client) error {
	if c == nil {
		return errors.New("client not valid")
	}
	if c.ID == "" {
		return errors.New("missing id")
	}
	if c.Username == "" {
		return errors.New("missing username")
	}
	if c.Password == "" {
		return errors.New("missing password")
	}
	if c.URL == "" {
		return errors.New("missing url")
	}
	if c.RunWaitTime == 0 {
		c.RunWaitTime = defaultRunWaitTime
	}
	if c.HealthWaitTime == 0 {
		c.HealthWaitTime = defaultHealthWaitTime
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
