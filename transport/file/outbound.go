package file

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/myopenfactory/edi-connector/client"
	"github.com/myopenfactory/edi-connector/transport"
)

type watchSetting struct {
	Path       string
	Extensions []string
	WaitTime   time.Duration
}

type outboundFileSettings struct {
	Messages    []watchSetting
	Attachments []watchSetting
	ErrorPath   string
	SuccessPath string
}

type outboundFilePlugin struct {
	logger    *slog.Logger
	processID string
	settings  outboundFileSettings
	client    *client.Client
}

// NewOutboundFilePlugin returns new OutPlugin and checks for success, error, messagewaittime and attachmentwaittime parameter.
func NewOutboundPlugin(logger *slog.Logger, pid string, cfg map[string]any, client *client.Client) (transport.OutboundPlugin, error) {
	var settings outboundFileSettings
	err := mapstructure.Decode(cfg, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to decode outbounf file settings: %w", err)
	}

	p := &outboundFilePlugin{
		logger:    logger,
		settings:  settings,
		processID: pid,
		client:    client,
	}

	if pid == "" {
		return nil, fmt.Errorf("no process id provided")
	}

	if _, err := os.Stat(settings.ErrorPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("error folder does not exist: %v", settings.ErrorPath)
	}

	p.logger.Info("configured outbound process", "configId", pid, "successFolder", settings.SuccessPath, "errorFolder", settings.ErrorPath)

	if len(settings.Messages) == 0 && len(settings.Attachments) == 0 {
		return nil, fmt.Errorf("either messages or attachments needs to be configured")
	}

	for _, message := range settings.Messages {
		if _, err := os.Stat(message.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("error outbound folder does not exist: %v", message.Path)
		} else if err != nil {
			return nil, fmt.Errorf("could not verify existence of outbound folder: %w", err)
		}

		if message.WaitTime == time.Duration(0) {
			message.WaitTime = 15 * time.Second
		}

		p.logger.Info("watching folder for messages", "folder", message.Path, "extensions", message.Extensions, "waitTime", message.WaitTime)
	}

	for _, attachment := range settings.Attachments {
		if _, err := os.Stat(attachment.Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("error attachment folder does not exist: %v", attachment.Path)
		}

		if attachment.WaitTime == time.Duration(0) {
			attachment.WaitTime = 15 * time.Second
		}

		p.logger.Info("watching folder for attachments", "folder", attachment.Path, "extensions", attachment.Extensions, "waitTime", attachment.WaitTime)
	}

	return p, nil
}

// ListMessages lists all messages found within message folder. Each file gets
// serialized into an transport.Message.
func (p *outboundFilePlugin) ListMessages(ctx context.Context) ([]transport.Message, error) {
	var files []string
	for _, message := range p.settings.Messages {
		for _, extension := range message.Extensions {
			fs, err := listFilesLastModifiedBefore(p.logger, message.Path, extension, time.Now().Add(-message.WaitTime))
			if err != nil {
				return nil, fmt.Errorf("failed to list files within %s: %w", message.Path, err)
			}
			files = append(files, fs...)
		}
	}

	messages, err := p.convertToMessages(files)
	if err != nil {
		return nil, fmt.Errorf("failed to convert message list: %w", err)
	}
	return messages, nil
}

// ListAttachments lists all attachments found within attachment folder. Each file gets
// serialized into an transport.Attachment.
func (p *outboundFilePlugin) ListAttachments(ctx context.Context) ([]transport.Attachment, error) {
	var files []string
	for _, attachment := range p.settings.Attachments {
		for _, extension := range attachment.Extensions {
			fs, err := listFilesLastModifiedBefore(p.logger, attachment.Path, extension, time.Now().Add(-attachment.WaitTime))
			if err != nil {
				return nil, fmt.Errorf("failed to list files within %s: %w", attachment.Path, err)
			}
			files = append(files, fs...)
		}
	}

	attachments, err := p.convertToAttachments(files)
	if err != nil {
		return nil, fmt.Errorf("failed to convert attachment list: %w", err)
	}
	return attachments, nil
}

func (p *outboundFilePlugin) convertToMessages(files []string) ([]transport.Message, error) {
	messages := make([]transport.Message, 0)
	for _, f := range files {
		buffer, err := os.ReadFile(f)
		if err != nil {
			if err := backupFileToFolder(p.logger, f, p.settings.ErrorPath); err != nil {
				p.logger.Error("failed to move file to error folder", "error", err)
			}
			return nil, fmt.Errorf("error while reading message %s: %w", f, err)
		}
		messages = append(messages, transport.Message{
			Id:      f,
			Content: buffer,
		})
	}
	return messages, nil
}

func (p *outboundFilePlugin) convertToAttachments(files []string) ([]transport.Attachment, error) {
	attachments := make([]transport.Attachment, 0)
	for _, f := range files {
		buffer, err := os.ReadFile(f)
		if err != nil {
			if err := backupFileToFolder(p.logger, f, p.settings.ErrorPath); err != nil {
				p.logger.Error("failed to move attachment to error folder", "error", err)
			}
			return nil, fmt.Errorf("error while reading attachment %s: %w", f, err)
		}
		attachments = append(attachments, transport.Attachment{
			Filename: f,
			Content:  buffer,
		})
	}
	return attachments, nil
}

func (p *outboundFilePlugin) Finalize(ctx context.Context, obj any, err error) error {
	var file string
	if message, ok := obj.(transport.Message); ok {
		file = message.Id
	}
	if attachment, ok := obj.(transport.Attachment); ok {
		file = attachment.Filename
	}
	if err != nil {
		if err := backupFileToFolder(p.logger, file, p.settings.ErrorPath); err != nil {
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

// listFilesLastModifiedBefore lists all files last modified before t for path and extension
func listFilesLastModifiedBefore(logger *slog.Logger, path, extension string, t time.Time) ([]string, error) {
	files := []string{}

	logger.Debug("searching folder %s for files with extension %s", path, extension)

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry == nil || dirEntry.IsDir() {
			continue
		}

		fp := filepath.Join(path, dirEntry.Name())
		fileInfo, err := dirEntry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve file info: %w", err)
		}
		if fileInfo.ModTime().Before(t) && strings.HasSuffix(fp, extension) {
			files = append(files, fp)
		}
	}

	return files, nil
}

// backupFileToFolder backups a file prefixed with current timestamp
func backupFileToFolder(logger *slog.Logger, filename, folder string) error {
	if filename == "" || folder == "" {
		return nil
	}

	f := filepath.Join(folder, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(filename)))

	logger.Debug("trying to backup file %v to %v", filename, f)
	if _, err := move(filename, f); err != nil {
		return fmt.Errorf("failed to backup file %s to %s: %w", filename, f, err)
	}
	logger.Info("backuped %s to %s", filename, f)

	return nil
}

func splitPathExtension(pathextension string) (string, string) {
	path := pathextension
	extension := ""

	seps := strings.Split(pathextension, ";")
	if len(seps) > 1 {
		path = seps[0]
		extension = seps[1]
	}

	path = filepath.Clean(strings.TrimSpace(path))

	return path, extension
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
