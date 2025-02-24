package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/myopenfactory/edi-connector/transport"
)

type watchSetting struct {
	Path       string
	Extensions []string
	WaitTime   time.Duration
}

type outboundFileSettings struct {
	Message     watchSetting
	Attachment  watchSetting
	ErrorPath   string
	SuccessPath string
}

type outboundFileTransport struct {
	logger    *slog.Logger
	processID string
	settings  outboundFileSettings
}

// NewOutboundFileTransport returns new OutTransport and checks for success, error, messagewaittime and attachmentwaittime parameter.
func NewOutboundTransport(logger *slog.Logger, pid string, cfg map[string]any) (transport.OutboundTransport, error) {
	var settings outboundFileSettings
	err := mapstructure.Decode(cfg, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to decode outbounf file settings: %w", err)
	}

	p := &outboundFileTransport{
		logger:    logger,
		settings:  settings,
		processID: pid,
	}

	if pid == "" {
		return nil, fmt.Errorf("no process id provided")
	}

	if _, err := os.Stat(settings.ErrorPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("error folder does not exist: %v", settings.ErrorPath)
	}

	p.logger.Info("configured outbound process", "configId", pid, "successFolder", settings.SuccessPath, "errorFolder", settings.ErrorPath)

	if settings.Message.Path == "" && settings.Attachment.Path == "" {
		return nil, fmt.Errorf("either messages or attachments needs to be configured")
	}

	message := settings.Message
	if _, err := os.Stat(message.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("error outbound folder does not exist: %v", message.Path)
	} else if err != nil {
		return nil, fmt.Errorf("could not verify existence of outbound folder: %w", err)
	}

	if message.WaitTime == time.Duration(0) {
		message.WaitTime = 15 * time.Second
	}

	p.logger.Info("watching folder for messages", "folder", message.Path, "extensions", message.Extensions, "waitTime", message.WaitTime)

	attachment := settings.Attachment
	if _, err := os.Stat(attachment.Path); os.IsNotExist(err) {
		return nil, fmt.Errorf("error attachment folder does not exist: %v", attachment.Path)
	}

	if attachment.WaitTime == time.Duration(0) {
		attachment.WaitTime = 15 * time.Second
	}

	p.logger.Info("watching folder for attachments", "folder", attachment.Path, "extensions", attachment.Extensions, "waitTime", attachment.WaitTime)

	return p, nil
}

// ListMessages lists all messages found within message folder. Each file gets
// serialized into an transport.Message.
func (p *outboundFileTransport) ListMessages(ctx context.Context) ([]transport.Message, error) {
	message := p.settings.Message

	fileInfos, err := p.listFilesLastModifiedBefore(message.Path, time.Now().Add(-message.WaitTime))
	if err != nil {
		return nil, fmt.Errorf("failed to list files within %s: %w", message.Path, err)
	}

	messages := make([]transport.Message, 0)
	for _, fileInfo := range fileInfos {
		fileExtension := filepath.Ext(fileInfo.Name())
		filePath := filepath.Join(message.Path, fileInfo.Name())
		for _, extension := range message.Extensions {
			if fileExtension == extension {
				buffer, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("error while reading attachment %s: %w", filePath, err)
				}
				messages = append(messages, transport.Message{
					Id:      filePath,
					Content: buffer,
				})
			}
		}
	}

	return messages, nil
}

// ListAttachments lists all attachments found within attachment folder. Each file gets
// serialized into an transport.Attachment.
func (p *outboundFileTransport) ListAttachments(ctx context.Context) ([]transport.Attachment, error) {
	attachment := p.settings.Attachment
	if attachment.Path == "" {
		return nil, fmt.Errorf("attachments not configured")
	}

	fileInfos, err := p.listFilesLastModifiedBefore(attachment.Path, time.Now().Add(-attachment.WaitTime))
	if err != nil {
		return nil, fmt.Errorf("failed to list files within %s: %w", attachment.Path, err)
	}

	attachments := make([]transport.Attachment, 0)
	for _, fileInfo := range fileInfos {
		fileExtension := filepath.Ext(fileInfo.Name())
		filePath := filepath.Join(attachment.Path, fileInfo.Name())
		for _, extension := range attachment.Extensions {
			if fileExtension == extension {
				buffer, err := os.ReadFile(filePath)
				if err != nil {
					return nil, fmt.Errorf("error while reading attachment %s: %w", filePath, err)
				}

				attachments = append(attachments, transport.Attachment{
					Filename: filePath,
					Content:  buffer,
				})
			}
		}
	}

	return attachments, nil
}

func (p *outboundFileTransport) Finalize(ctx context.Context, obj any, err error) error {
	var file string
	if message, ok := obj.(transport.Message); ok {
		file = message.Id
	}
	if attachment, ok := obj.(transport.Attachment); ok {
		file = attachment.Filename
	}
	if err != nil {
		destination := filepath.Join(p.settings.ErrorPath, filepath.FromSlash(file))
		if _, err := move(file, destination); err != nil {
			return err
		}
		return nil
	}

	if p.settings.SuccessPath != "" {
		newfile := filepath.Join(p.settings.SuccessPath, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file)))
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

func (p *outboundFileTransport) HandleAttachments() bool {
	if p.settings.Attachment.Path == "" {
		return false
	}
	return true
}

// listFilesLastModifiedBefore lists all files last modified before t for path and extension
func (p *outboundFileTransport) listFilesLastModifiedBefore(path string, t time.Time) ([]os.FileInfo, error) {
	files := []os.FileInfo{}

	p.logger.Debug("searching folder %s for files modified before %t", path, t)

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
