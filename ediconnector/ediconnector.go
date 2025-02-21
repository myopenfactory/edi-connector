package ediconnector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/myopenfactory/edi-connector/config"
	"github.com/myopenfactory/edi-connector/platform"
	"github.com/myopenfactory/edi-connector/transport"
	"github.com/myopenfactory/edi-connector/transport/file"
	"github.com/myopenfactory/edi-connector/version"
)

// Config configures variables for the client
type Connector struct {
	logger      *slog.Logger
	id          string
	runWaitTime time.Duration

	// transports
	inbounds  map[string]transport.InboundTransport
	outbounds map[string]transport.OutboundTransport

	platformClient *platform.Client
}

// New creates client with given options
func New(logger *slog.Logger, cfg config.Config) (*Connector, error) {
	if proxy := cfg.Proxy; proxy != "" {
		os.Setenv("HTTP_PROXY", proxy)
		os.Setenv("HTTPS_PROXY", proxy)
	}

	platformClient, err := platform.NewClient(cfg.Url, cfg.Username, cfg.Password, cfg.ClientCertificate, cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform client: %w", err)
	}

	c := &Connector{
		logger:         logger,
		id:             fmt.Sprintf("Core_%s", version.Version),
		runWaitTime:    cfg.RunWaitTime,
		platformClient: platformClient,
	}

	logger.Info("Configured connector", "runWaitTime", c.runWaitTime, "id", c.id)

	c.inbounds = make(map[string]transport.InboundTransport)
	c.outbounds = make(map[string]transport.OutboundTransport)
	for _, pc := range cfg.Outbounds {
		switch pc.Type {
		case "FILE":
			c.outbounds[pc.Id], err = file.NewOutboundTransport(c.logger, pc.Id, pc.Settings, platformClient)
			if err != nil {
				return nil, fmt.Errorf("failed to load transport: processid: %v: %w", pc.Id, err)
			}
		}
	}
	for _, pc := range cfg.Inbounds {
		switch pc.Type {
		case "FILE":
			c.inbounds[pc.Id], err = file.NewInboundTransport(c.logger, pc.Id, pc.Settings, platformClient)
			if err != nil {
				return nil, fmt.Errorf("failed to load transport: processid: %v: %w", pc.Id, err)
			}
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
			for configId, transport := range c.outbounds {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := c.outboundAttachments(ctx, transport); err != nil {
					c.logger.Error("error processing outbound attachment: %s", "error", err)
				}
				cancel()

				ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := c.outboundMessages(ctx, transport, configId); err != nil {
					c.logger.Error("error processing outbound message: %s", "error", err)
				}
				cancel()
			}

			for configId, transport := range c.inbounds {
				if err := c.inboundMessages(ctx, transport, configId); err != nil {
					c.logger.Error("error processing inbound transmissions", "configId", configId, "error", err)
				}

			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Connector) outboundMessages(ctx context.Context, outbound transport.OutboundTransport, configId string) error {
	messages, err := outbound.ListMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to list messages")
	}

	finalizer, isFinalizer := outbound.(transport.Finalizer)
	for _, msg := range messages {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := c.platformClient.AddTransmission(ctx, configId, msg.Content); err != nil {
			if isFinalizer {
				err = finalizer.Finalize(ctx, msg, err)
				if err != nil {
					return fmt.Errorf("could not finalize message %s: %w", msg.Id, err)
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
	attachmentLister, isAttachmentLister := outbound.(transport.AttachmentLister)
	if !isAttachmentLister {
		return nil
	}
	attachments, err := attachmentLister.ListAttachments(ctx)
	if err != nil {
		c.logger.Error("error while reading attachment: %v", "error", err)
	}

	finalizer, isFinalizer := outbound.(transport.Finalizer)
	for _, attachment := range attachments {
		if err := c.platformClient.AddAttachment(ctx, attachment.Content, attachment.Filename); err != nil {
			if isFinalizer {
				err = finalizer.Finalize(ctx, attachment, err)
				if err != nil {
					return fmt.Errorf("could not finalize attachment %s: %w", attachment.Filename, err)
				}
			}
			return fmt.Errorf("failed to upload attachment for %s: %w", attachment.Filename, err)
		}
		if isFinalizer {
			err = finalizer.Finalize(ctx, attachment, nil)
			if err != nil {
				return fmt.Errorf("could not finalize attachment %s: %w", attachment.Filename, err)
			}
		}
	}
	return nil
}

func (c *Connector) inboundMessages(ctx context.Context, inbound transport.InboundTransport, configId string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	transmissions, err := c.platformClient.ListTransmissions(ctx, configId)
	if err != nil {
		return fmt.Errorf("failed to list transmissions: %w", err)
	}
	cancel()

	for _, transmission := range transmissions {
		if err := c.inboundAttachments(ctx, inbound, transmission); err != nil {
			return fmt.Errorf("could not process attachment for %s: %w", transmission.Id, err)
		}

		data, err := c.platformClient.DownloadTransmission(transmission)
		if err != nil {
			c.logger.Error("failed to download transmission", "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		err = inbound.ProcessMessage(ctx, transport.Message{
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
		err = c.platformClient.ConfirmTransmission(ctx, transmission.Id)
		if err != nil {
			return fmt.Errorf("could not confirm inbound transmission %s: %w", transmission.Id, err)
		}
		cancel()
	}
	return nil
}

func (c *Connector) inboundAttachments(ctx context.Context, inbound transport.InboundTransport, transmission platform.Transmission) error {
	// only process if transport supports processing attachments
	attachmentProcessor, isAttachmentProcessor := inbound.(transport.AttachmentProcessor)
	if !isAttachmentProcessor {
		return nil
	}
	// only process if transport config has enabled processing attachments
	if !attachmentProcessor.HandleAttachments() {
		return nil
	}
	messageId, ok := transmission.Metadata["TID"]
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	attachments, err := c.platformClient.ListMessageAttachments(ctx, messageId)
	if err != nil {
		return fmt.Errorf("failed to list message attachments for %s: %w", messageId, err)
	}
	for _, attachment := range attachments {
		data, filename, err := c.platformClient.DownloadAttachment(attachment)
		if err != nil {
			return fmt.Errorf("failed to download attachment for %s: %w", messageId, err)
		}
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := attachmentProcessor.ProcessAttachment(ctx, transport.Attachment{
			Filename: filename,
			Content:  data,
		}); err != nil {
			return fmt.Errorf("error processing attachment: %w", err)
		}
		cancel()
	}
	return nil
}
