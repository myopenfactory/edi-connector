package ediconnector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/myopenfactory/edi-connector/client"
	"github.com/myopenfactory/edi-connector/config"
	"github.com/myopenfactory/edi-connector/transport"
	"github.com/myopenfactory/edi-connector/transport/file"
	"github.com/myopenfactory/edi-connector/version"
)

// Config configures variables for the client
type Connector struct {
	logger      *slog.Logger
	id          string
	runWaitTime time.Duration

	// plugins
	inbounds  map[string]transport.InboundPlugin
	outbounds map[string]transport.OutboundPlugin

	ediClient *client.Client
}

// New creates client with given options
func New(logger *slog.Logger, cfg config.Config) (*Connector, error) {
	if proxy := cfg.Proxy; proxy != "" {
		os.Setenv("HTTP_PROXY", proxy)
		os.Setenv("HTTPS_PROXY", proxy)
	}

	ediClient, err := client.New(cfg.Url, cfg.Username, cfg.Password, cfg.ClientCertificate, cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform client: %w", err)
	}

	c := &Connector{
		logger:      logger,
		id:          fmt.Sprintf("Core_%s", version.Version),
		runWaitTime: cfg.RunWaitTime,
		ediClient:   ediClient,
	}

	logger.Info("Configured connector", "runWaitTime", c.runWaitTime, "id", c.id)

	c.inbounds = make(map[string]transport.InboundPlugin)
	c.outbounds = make(map[string]transport.OutboundPlugin)
	for _, pc := range cfg.Outbounds {
		switch pc.Type {
		case "FILE":
			c.outbounds[pc.Id], err = file.NewOutboundPlugin(c.logger, pc.Id, pc.Settings, ediClient)
			if err != nil {
				return nil, fmt.Errorf("failed to load plugin: processid: %v: %w", pc.Id, err)
			}
		}
	}
	for _, pc := range cfg.Inbounds {
		switch pc.Type {
		case "FILE":
			c.inbounds[pc.Id], err = file.NewInboundPlugin(c.logger, pc.Id, pc.Settings, ediClient)
			if err != nil {
				return nil, fmt.Errorf("failed to load plugin: processid: %v: %w", pc.Id, err)
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
			for configId, plugin := range c.outbounds {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := c.outboundAttachments(ctx, plugin); err != nil {
					c.logger.Error("error processing outbound attachment: %s", err)
				}
				cancel()

				ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := c.outboundMessages(ctx, plugin, configId); err != nil {
					c.logger.Error("error processing outbound message: %s", err)
				}
				cancel()
			}

			for configId, plugin := range c.inbounds {
				if err := c.inboundMessages(ctx, plugin, configId); err != nil {
					c.logger.Error("error processing inbound transmissions", "configId", configId, "error", err)
				}

			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Connector) outboundMessages(ctx context.Context, plugin transport.OutboundPlugin, configId string) error {
	messages, err := plugin.ListMessages(ctx)
	if err != nil {
		return fmt.Errorf("failed to list messages")
	}

	finalizer, isFinalizer := plugin.(transport.Finalizer)
	for _, msg := range messages {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := c.ediClient.AddTransmission(ctx, configId, msg.Content); err != nil {
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

func (c *Connector) outboundAttachments(ctx context.Context, plugin transport.OutboundPlugin) error {
	attachmentLister, isAttachmentLister := plugin.(transport.AttachmentLister)
	if !isAttachmentLister {
		return nil
	}
	attachments, err := attachmentLister.ListAttachments(ctx)
	if err != nil {
		c.logger.Error("error while reading attachment: %v", err)
	}

	finalizer, isFinalizer := plugin.(transport.Finalizer)
	for _, attachment := range attachments {
		if err := c.ediClient.AddAttachment(ctx, attachment.Content, attachment.Filename); err != nil {
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

func (c *Connector) inboundMessages(ctx context.Context, plugin transport.InboundPlugin, configId string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	transmissions, err := c.ediClient.ListTransmissions(ctx, configId)
	if err != nil {
		return fmt.Errorf("failed to list transmissions: %w", err)
	}
	cancel()

	for _, transmission := range transmissions {
		if plugin.HandleAttachment() {
			if err := c.inboundAttachments(ctx, plugin, transmission); err != nil {
				return fmt.Errorf("could not process attachment for %s: %w", transmission.Id, err)
			}
		}

		data, err := c.ediClient.DownloadTransmission(transmission)
		if err != nil {
			c.logger.Error("failed to download transmission", "error", err)
			continue
		}

		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		err = plugin.ProcessMessage(ctx, transport.Message{
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
		err = c.ediClient.ConfirmTransmission(ctx, transmission.Id)
		if err != nil {
			return fmt.Errorf("could not confirm inbound transmission %s: %w", transmission.Id, err)
		}
		cancel()
	}
	return nil
}

func (c *Connector) inboundAttachments(ctx context.Context, plugin transport.InboundPlugin, transmission client.Transmission) error {
	attachmentProcessor, isAttachmentProcessor := plugin.(transport.AttachmentProcessor)
	if !isAttachmentProcessor {
		return nil
	}
	messageId, ok := transmission.Metadata["TID"]
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	attachments, err := c.ediClient.ListMessageAttachments(ctx, messageId)
	if err != nil {
		return fmt.Errorf("failed to list message attachments for %s: %w", messageId, err)
	}
	for _, attachment := range attachments {
		data, filename, err := c.ediClient.DownloadAttachment(attachment)
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
