package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/myopenfactory/edi-connector/transport"
)

type inboundFileSettings struct {
	Path                string
	AttachmentPath      string
	AttachmentWhitelist []string
	Exist               string
}

// InboundFileTransport type
type inboundFileTransport struct {
	logger   *slog.Logger
	settings inboundFileSettings
}

// NewInboundFileTransport returns new InTransport and checks for basefolder and exist parameter.
func NewInboundTransport(logger *slog.Logger, pid string, cfg map[string]any) (transport.InboundTransport, error) {
	var settings inboundFileSettings
	err := mapstructure.Decode(cfg, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to decode inbound file settings: %w", err)
	}

	if settings.Path == "" {
		return nil, fmt.Errorf("setting an output folder is required")
	}

	if _, err := os.Stat(settings.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("folder %s does not exist: %w", settings.Path, err)
	}
	if _, err := os.Stat(settings.AttachmentPath); settings.AttachmentPath != "" && os.IsNotExist(err) {
		return nil, fmt.Errorf("attachment folder %s does not exist: %w", settings.AttachmentPath, err)
	}

	if settings.Exist != "append" {
		settings.Exist = "count"
	}

	logger.Info("configured inbound process", "configId", pid, "folder", settings.Path, "strategy", settings.Exist)
	return &inboundFileTransport{
		logger:   logger,
		settings: settings,
	}, nil
}

func (p *inboundFileTransport) HandleAttachment() bool {
	if p.settings.AttachmentPath == "" {
		return false
	}
	return true
}

// ConsumeMessage consumes message from plattform and saves it to a file
func (p *inboundFileTransport) ProcessMessage(ctx context.Context, msg transport.Object) error {
	filename := msg.Id
	if value, ok := msg.Metadata["filename"]; ok {
		filename = value
	}
	path := filepath.Join(p.settings.Path, filename)
	if p.settings.Exist == "append" {
		p.logger.Info("Appending to file", "path", path)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("error while open file %s: %w", path, err)
		}
		defer f.Close()
		_, err = f.Write(msg.Content)
		if err != nil {
			return fmt.Errorf("error while writing file %s: %w", path, err)
		}
		return nil
	}

	p.logger.Info("Creating file", "path", path)

	if err := os.WriteFile(filename, msg.Content, 0644); err != nil {
		return fmt.Errorf("error while writing file %s: %w", path, err)
	}
	return nil
}

// ProcessAttachment processes the attachment and writes it to specified path. In case of already existing file a
// new filename is derived.
func (p *inboundFileTransport) ProcessAttachment(ctx context.Context, atc transport.Object) error {
	filename := atc.Id
	if value, ok := atc.Metadata["filename"]; ok {
		filename = value
	}
	path := filepath.Join(p.settings.AttachmentPath, filename)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not open target file: %s: %w", path, err)
	}
	defer f.Close()

	_, err = f.Write(atc.Content)
	if err != nil {
		return fmt.Errorf("failed to write attachment to file %q: %w", path, err)
	}

	return nil
}
