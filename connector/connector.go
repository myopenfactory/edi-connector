package connector

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/myopenfactory/edi-connector/v2/config"
	"github.com/myopenfactory/edi-connector/v2/platform"
	"github.com/myopenfactory/edi-connector/v2/transport"
	"github.com/myopenfactory/edi-connector/v2/transport/file"
)

// Config configures variables for the client
type Connector struct {
	logger      *slog.Logger
	runWaitTime time.Duration

	// transports
	inbounds  []transport.InboundTransport
	outbounds []transport.OutboundTransport

	platformClient *platform.Client
}

// New creates client with given options
func New(logger *slog.Logger, cfg config.Config) (*Connector, error) {
	platformClient, err := platform.NewClient(cfg.Url, cfg.CAFile, cfg.Proxy)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform client: %w", err)
	}

	c := &Connector{
		logger:         logger,
		runWaitTime:    cfg.RunWaitTime,
		platformClient: platformClient,
	}

	logger.Info("Configured connector", "runWaitTime", c.runWaitTime)

	c.inbounds = []transport.InboundTransport{}
	c.outbounds = []transport.OutboundTransport{}
	for _, pc := range cfg.Outbounds {
		switch pc.Type {
		case "FILE":
			outbound, err := file.NewOutboundTransport(c.logger, pc.Id, pc.AuthName, pc.Settings)
			if err != nil {
				return nil, fmt.Errorf("failed to load transport: processid: %v: %w", pc.Id, err)
			}
			c.outbounds = append(c.outbounds, outbound)
		}
	}
	for _, pc := range cfg.Inbounds {
		switch pc.Type {
		case "FILE":
			inbound, err := file.NewInboundTransport(c.logger, pc.Id, pc.AuthName, pc.Settings)
			if err != nil {
				return nil, fmt.Errorf("failed to load transport: processid: %v: %w", pc.Id, err)
			}
			c.inbounds = append(c.inbounds, inbound)
		}
	}

	return c, nil
}

// Runs client until context is closed
func (c *Connector) Run(ctx context.Context) error {
	ticker := time.NewTicker(c.runWaitTime)
	for {
		select {
		case <-ticker.C:
			for _, transport := range c.outbounds {
				ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
				if err := c.outboundAttachments(ctx, transport); err != nil {
					c.logger.Error("error processing outbound attachment", "error", err)
					cancel()
					os.Exit(1)
				}
				cancel()

				ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
				if err := c.outboundMessages(ctx, transport); err != nil {
					c.logger.Error("error processing outbound message", "error", err)
					cancel()
					os.Exit(1)
				}
				cancel()
			}

			for _, transport := range c.inbounds {
				ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
				if err := c.inboundMessages(ctx, transport); err != nil {
					c.logger.Error("error processing inbound transmissions", "configId", transport.ConfigId(), "error", err)
					cancel()
					os.Exit(1)
				}
				cancel()
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Connector) outboundMessages(ctx context.Context, outbound transport.OutboundTransport) error {
	messages, err := outbound.ListMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to list messages")
	}

	finalizer, isFinalizer := outbound.(transport.Finalizer)
	for _, msg := range messages {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := c.platformClient.AddTransmission(ctx, outbound.ConfigId(), outbound.AuthName(), msg.Content); err != nil {
			if isFinalizer {
				finalizerErr := finalizer.Finalize(ctx, msg, err)
				if finalizerErr != nil {
					return fmt.Errorf("could not finalize message %s: %w", msg.Id, finalizerErr)
				}
			}
			return fmt.Errorf("failed tu upload message %s: %w", msg.Id, err)
		}
		if isFinalizer {
			err = finalizer.Finalize(ctx, msg, nil)
			if err != nil {
				return fmt.Errorf("could not finalize message %s: %w", msg.Id, err)
			}
		}
		cancel()
	}

	return nil
}

func (c *Connector) outboundAttachments(ctx context.Context, outbound transport.OutboundTransport) error {
	attachments, err := outbound.ListAttachments(ctx)
	if err != nil {
		c.logger.Error("error while reading attachment: %v", "error", err)
	}

	finalizer, isFinalizer := outbound.(transport.Finalizer)
	for _, attachment := range attachments {
		if err := c.platformClient.AddAttachment(ctx, attachment.Content, attachment.Id, outbound.AuthName()); err != nil {
			if isFinalizer {
				finalizerErr := finalizer.Finalize(ctx, attachment, err)
				if finalizerErr != nil {
					return fmt.Errorf("could not finalize attachment %s: %w", attachment.Id, finalizerErr)
				}
			}
			return fmt.Errorf("failed to upload attachment for %s: %w", attachment.Id, err)
		}
		if isFinalizer {
			err = finalizer.Finalize(ctx, attachment, nil)
			if err != nil {
				return fmt.Errorf("could not finalize attachment %s: %w", attachment.Id, err)
			}
		}
	}
	return nil
}

func (c *Connector) inboundMessages(ctx context.Context, inbound transport.InboundTransport) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	transmissions, err := c.platformClient.ListTransmissions(ctx, inbound.ConfigId(), inbound.AuthName())
	if err != nil {
		return fmt.Errorf("failed to list transmissions: %w", err)
	}
	cancel()

	for _, transmission := range transmissions {
		if err := c.inboundAttachments(ctx, inbound, transmission); err != nil {
			return fmt.Errorf("could not process attachment for %s: %w", transmission.Id, err)
		}

		data, err := c.platformClient.DownloadTransmission(transmission, inbound.AuthName())
		if err != nil {
			c.logger.Error("failed to download transmission", "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		statusMsg, err := inbound.ProcessMessage(ctx, transport.Object{
			Id:       transmission.Id,
			Content:  data,
			Metadata: transmission.Metadata,
		})
		if err != nil {
			return fmt.Errorf("failed to process message: %s", transmission.Id)
		}
		cancel()

		ctx, cancel = context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		err = c.platformClient.ConfirmTransmission(ctx, transmission.Id, inbound.AuthName(), statusMsg)
		if err != nil {
			return fmt.Errorf("could not confirm inbound transmission %s: %w", transmission.Id, err)
		}
		cancel()
	}
	return nil
}

func (c *Connector) inboundAttachments(ctx context.Context, inbound transport.InboundTransport, transmission platform.Transmission) error {
	messageId, ok := transmission.Metadata["TID"]
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	attachments, err := c.platformClient.ListMessageAttachments(ctx, messageId, inbound.AuthName())
	if err != nil {
		return fmt.Errorf("failed to list message attachments for %s: %w", messageId, err)
	}

	for _, attachment := range attachments {
		if !inbound.HandleAttachment(attachment.Url) {
			return nil
		}

		data, filename, err := c.downloadAttachment(attachment.Url)
		if err != nil {
			return fmt.Errorf("failed to download attachment for %s: %w", messageId, err)
		}

		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := inbound.ProcessAttachment(ctx, transport.Object{
			Id:      generateId(),
			Content: data,
			Metadata: map[string]string{
				"filename": filename,
			},
		}); err != nil {
			return fmt.Errorf("error processing attachment: %w", err)
		}
	}
	return nil
}

func generateId() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

func (c *Connector) downloadAttachment(attachmentUrl string) ([]byte, string, error) {
	if attachmentUrl == "" {
		return nil, "", fmt.Errorf("attachment url couldn't be empty")
	}
	resp, err := http.Get(attachmentUrl)
	if err != nil {
		return nil, "", fmt.Errorf("error while loading attachment with url %q: %w", attachmentUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("error bad response for attachment %q: %w", attachmentUrl, err)
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		return nil, "", fmt.Errorf("invalid content-disposition on attachment %q: %w", attachmentUrl, err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("error while writing response data for attachment %q: %w", attachmentUrl, err)
	}

	filename, ok := params["filename"]
	if !ok {
		url, err := url.Parse(attachmentUrl)
		if err != nil {
			return nil, "", fmt.Errorf("failed to parse attachment url: %w", err)
		}
		slashIndex := strings.LastIndex(url.Path, "/")
		if slashIndex != -1 {
			filename = url.Path[slashIndex+1:]
		}
	}

	return data, filename, nil
}
