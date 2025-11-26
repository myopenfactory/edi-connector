package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/myopenfactory/edi-connector/v2/config"
	"github.com/myopenfactory/edi-connector/v2/transport"
)

type watchSetting struct {
	Path       string   `json:"path" yaml:"path"`
	Extensions []string `json:"extensions" yaml:"extensions"`
	WaitTime   string   `json:"waitTime" yaml:"waitTime"`
}

type outboundFileSettings struct {
	Message     watchSetting `json:"message" yaml:"message"`
	Attachment  watchSetting `json:"attachment" yaml:"attachment"`
	ErrorPath   string       `json:"errorPath" yaml:"errorPath"`
	SuccessPath string       `json:"successPath" yaml:"successPath"`
}

type outboundFileTransport struct {
	logger   *slog.Logger
	configId string
	authName string
	settings outboundFileSettings
}

func (p *outboundFileTransport) isMessageEnabled() bool {
	return p.settings.Message.Path != ""
}

func (p *outboundFileTransport) isAttachmentEnabled() bool {
	return p.settings.Attachment.Path != ""
}

// NewOutboundFileTransport returns new OutTransport and checks for success, error, messagewaittime and attachmentwaittime parameter.
func NewOutboundTransport(logger *slog.Logger, configId, authName string, cfg map[string]any) (transport.OutboundTransport, error) {
	var settings outboundFileSettings
	err := config.Decode(cfg, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to decode outbound file settings: %w", err)
	}

	p := &outboundFileTransport{
		logger:   logger,
		settings: settings,
		configId: configId,
		authName: authName,
	}

	if configId == "" {
		return nil, fmt.Errorf("no process id provided")
	}

	if p.isMessageEnabled() {
		if _, err := os.Stat(settings.ErrorPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("error folder does not exist: %v", settings.ErrorPath)
		}

		p.logger.Info("configured outbound process", "configId", p.configId, "authName", p.authName, "successFolder", settings.SuccessPath, "errorFolder", settings.ErrorPath)

		message := settings.Message
		if _, err := os.Stat(message.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("error outbound folder does not exist: %v", message.Path)
		} else if err != nil {
			return nil, fmt.Errorf("could not verify existence of outbound folder: %w", err)
		}

		if message.WaitTime == "" {
			message.WaitTime = "15s"
		}
		p.logger.Info("watching folder for messages", "folder", message.Path, "extensions", message.Extensions, "waitTime", message.WaitTime)
	} else {
		p.logger.Info("message polling disabled")
	}

	if p.isAttachmentEnabled() {
		attachment := settings.Attachment
		if _, err := os.Stat(attachment.Path); attachment.Path != "" && os.IsNotExist(err) {
			return nil, fmt.Errorf("error attachment folder does not exist: %v", attachment.Path)
		}

		if attachment.WaitTime == "" {
			attachment.WaitTime = "15s"
		}

		p.logger.Info("watching folder for attachments", "folder", attachment.Path, "extensions", attachment.Extensions, "waitTime", attachment.WaitTime)
	} else {
		p.logger.Info("attachment polling disabled")
	}

	return p, nil
}

func (p *outboundFileTransport) ConfigId() string {
	return p.configId
}

func (p *outboundFileTransport) AuthName() string {
	return p.authName
}

// ListMessages lists all messages found within message folder. Each file gets
// serialized into an transport.Object.
func (p *outboundFileTransport) ListMessages(ctx context.Context) ([]transport.Object, error) {
	messages := make([]transport.Object, 0)
	if !p.isMessageEnabled() {
		return messages, nil
	}

	message := p.settings.Message
	duration, err := time.ParseDuration(message.WaitTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}
	fileInfos, err := p.listFilesLastModifiedBefore(message.Path, time.Now().Add(-duration))
	if err != nil {
		return nil, fmt.Errorf("failed to list files within %s: %w", message.Path, err)
	}

	for _, fileInfo := range fileInfos {
		fileExtension := filepath.Ext(fileInfo.Name())[1:]
		filePath := filepath.Join(message.Path, fileInfo.Name())
		for _, extension := range message.Extensions {
			if fileExtension == extension {
				buffer, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("error while reading attachment %s: %w", filePath, err)
				}
				messages = append(messages, transport.Object{
					Id:      filePath,
					Content: buffer,
				})
			}
		}
	}

	return messages, nil
}

// ListAttachments lists all attachments found within attachment folder. Each file gets
// serialized into an transport.Object.
func (p *outboundFileTransport) ListAttachments(ctx context.Context) ([]transport.Object, error) {
	attachments := make([]transport.Object, 0)
	if !p.isAttachmentEnabled() {
		return attachments, nil
	}
	attachment := p.settings.Attachment

	duration, err := time.ParseDuration(attachment.WaitTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse duration: %w", err)
	}
	fileInfos, err := p.listFilesLastModifiedBefore(attachment.Path, time.Now().Add(-duration))
	if err != nil {
		return nil, fmt.Errorf("failed to list files within %s: %w", attachment.Path, err)
	}

	for _, fileInfo := range fileInfos {
		fileExtension := filepath.Ext(fileInfo.Name())[1:]
		filePath := filepath.Join(attachment.Path, fileInfo.Name())
		for _, extension := range attachment.Extensions {
			if fileExtension == extension {
				buffer, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("error while reading attachment %s: %w", filePath, err)
				}

				attachments = append(attachments, transport.Object{
					Id:      filePath,
					Content: buffer,
				})
			}
		}
	}

	return attachments, nil
}

func (p *outboundFileTransport) Finalize(ctx context.Context, obj transport.Object, err error) error {
	file := obj.Id
	if err != nil {
		destination := filepath.Join(p.settings.ErrorPath, filepath.Base(file))
		if _, err := move(file, destination); err != nil {
			return err
		}
		return nil
	}

	if p.settings.SuccessPath != "" {
		newfile := filepath.Join(p.settings.SuccessPath, filepath.Base(file))
		if _, err := move(file, newfile); err != nil {
			return fmt.Errorf("error while moving file %s: %w", file, err)
		}
		p.logger.Info("file moved", "source", file, "destination", newfile)
		return nil
	}

	if err := os.Remove(file); err != nil {
		return fmt.Errorf("error while deleting file %s: %w", file, err)
	}
	p.logger.Info("file deleted", "path", file)

	return nil
}

// listFilesLastModifiedBefore lists all files last modified before t for path and extension
func (p *outboundFileTransport) listFilesLastModifiedBefore(path string, t time.Time) ([]os.FileInfo, error) {
	files := []os.FileInfo{}

	p.logger.Debug("searching folder for files modified before", "folder", path, "time", t)

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry == nil || dirEntry.IsDir() {
			continue
		}

		fileInfo, err := dirEntry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve file info: %w", err)
		}
		if fileInfo.ModTime().Before(t) {
			files = append(files, fileInfo)
		}
	}

	slices.SortFunc(files, func(a os.FileInfo, b os.FileInfo) int {
		if a.ModTime().Before(b.ModTime()) {
			return -1
		} else if a.ModTime().After(b.ModTime()) {
			return 1
		}
		return 0
	})

	return files, nil
}

// move copys src to dst and removes the src file.
// Introduced for compatibility issues with os.Rename
// as it doesn't handle smart-links(by docker) very well.
func move(src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	stat, err := in.Stat()
	if err != nil {
		return 0, err
	}

	if !stat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	out, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	n, err := io.Copy(out, in)
	if err != nil {
		return n, err
	}

	in.Close()
	if err := os.Remove(src); err != nil {
		return 0, err
	}

	return n, nil
}
