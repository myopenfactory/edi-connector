package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/myopenfactory/edi-connector/v2/transport"
)

type inboundFileSettings struct {
	transport.InboundSettings `mapstructure:",squash"`
	Path                      string
	AttachmentPath            string
	Mode                      string
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

	if settings.Mode == "" {
		settings.Mode = "create"
	}

	logger.Info("configured inbound process", "configId", pid, "folder", settings.Path, "mode", settings.Mode)
	return &inboundFileTransport{
		logger:   logger,
		settings: settings,
	}, nil
}

func (p *inboundFileTransport) HandleAttachment(url string) bool {
	if p.settings.AttachmentPath == "" || len(p.settings.AttachmentWhitelist) == 0 {
		return false
	}

	for _, whitelist := range p.settings.AttachmentWhitelist {
		if strings.HasPrefix(url, whitelist) {
			return true
		}
	}

	return false
}

// ConsumeMessage consumes message from plattform and saves it to a file
func (p *inboundFileTransport) ProcessMessage(ctx context.Context, msg transport.Object) (string, error) {
	if p.settings.Mode == "append" {
		filename := msg.Id
		if value, ok := msg.Metadata["filename"]; ok {
			filename = value
		}
		path := filepath.Join(p.settings.Path, filename)
		p.logger.Info("Appending to file", "path", path)
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return "", fmt.Errorf("error while open file %s: %w", path, err)
		}
		defer f.Close()
		_, err = f.Write(msg.Content)
		if err != nil {
			return "", fmt.Errorf("error while writing file %s: %w", path, err)
		}
		return fmt.Sprintf("Appending to file: %s", path), nil
	}

	return p.writeObject(msg, p.settings.Path)
}

// ProcessAttachment processes the attachment and writes it to specified path. In case of already existing file a
// new filename is derived.
func (p *inboundFileTransport) ProcessAttachment(ctx context.Context, atc transport.Object) error {
	_, err := p.writeObject(atc, p.settings.AttachmentPath)
	return err
}

func (p *inboundFileTransport) writeObject(obj transport.Object, basePath string) (string, error) {
	filename := obj.Id
	if value, ok := obj.Metadata["filename"]; ok && value != "" {
		filename = value
	}
	path := filepath.Join(basePath, filename)

	p.logger.Info("Creating file", "path", path)
	err := os.WriteFile(path, obj.Content, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to file %q: %w", path, err)
	}

	return fmt.Sprintf("Created file: %s", path), nil
}
