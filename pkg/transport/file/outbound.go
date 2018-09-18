package file

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/transport"
	"github.com/pkg/errors"
)

type folder struct {
	path      string
	extension string
}

type outboundFilePlugin struct {
	msgFolders    []folder
	atcFolders    []folder
	successFolder string
	errorFolder   string
	processID     string
	parameter     map[string]string
	waitTime      int
	msgProcessor  transport.MessageProcessor
	atcProcessor  transport.AttachmentProcessor
}

// NewOutboundFilePlugin returns new OutPlugin and checks for success, error and waittime parameter.
func NewOutboundPlugin(pid string, msgProcessor transport.MessageProcessor, atcProcessor transport.AttachmentProcessor, parameter map[string]string) (transport.OutboundPlugin, error) {
	p := &outboundFilePlugin{
		waitTime:     15,
		parameter:    parameter,
		processID:    pid,
		msgProcessor: msgProcessor,
		atcProcessor: atcProcessor,
	}

	if pid == "" {
		return nil, fmt.Errorf("no process id provided")
	}

	for k, v := range parameter {
		path, ext := splitPathExtension(v)
		if !strings.HasPrefix(k, "folder") {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("error outbound folder does not exist: %v", path)
		}
		p.msgFolders = append(p.msgFolders, folder{
			path:      path,
			extension: ext,
		})
	}

	for k, v := range parameter {
		path, ext := splitPathExtension(v)
		if !strings.HasPrefix(k, "attachmentfolder") {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, fmt.Errorf("error attachment folder does not exist: %v", path)
		}
		p.atcFolders = append(p.atcFolders, folder{
			path:      path,
			extension: ext,
		})
	}
	if v := parameter["success"]; v != "" {
		if _, err := os.Stat(v); os.IsNotExist(err) {
			return nil, fmt.Errorf("error success folder does not exist: %v", v)
		}
		p.successFolder = v
	}

	if v := parameter["error"]; v != "" {
		if _, err := os.Stat(v); os.IsNotExist(err) {
			return nil, fmt.Errorf("error folder does not exist: %v", v)
		}
		p.errorFolder = v
	}

	if v := parameter["waittime"]; v != "" {
		wt, err := strconv.Atoi(v)
		if wt < 0 || err != nil {
			return nil, errors.Wrapf(err, "error while converting waittime to integer: %v", v)
		}
		p.waitTime = wt
	}

	log.Infof("using waittime=%v, successFolder=%v, errorFolder=%v", p.waitTime, p.successFolder, p.errorFolder)

	return p, nil
}

// ListMessages lists all messages found within message folder. Each file gets
// serialized into an pb.Message.
func (p *outboundFilePlugin) ListMessages(ctx context.Context) ([]*pb.Message, error) {
	var files []string
	for _, f := range p.msgFolders {
		fs, err := listFilesLastModifiedBefore(f.path, f.extension, time.Now().Add(-time.Duration(p.waitTime)*time.Second))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list files within %s", f.path)
		}
		files = append(files, fs...)
	}

	messages, err := p.convertToMessages(files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert message list")
	}
	return messages, nil
}

// ListAttachments lists all attachments found within attachment folder. Each file gets
// serialized into an pb.Attachment.
func (p *outboundFilePlugin) ListAttachments(ctx context.Context) ([]*pb.Attachment, error) {
	var files []string
	for _, f := range p.atcFolders {
		fs, err := listFilesLastModifiedBefore(f.path, f.extension, time.Now().Add(-time.Duration(p.waitTime)*time.Second))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list files within %s", f.path)
		}
		files = append(files, fs...)
	}

	attachments, err := p.convertToAttachments(files)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert attachment list")
	}
	return attachments, nil
}

// ProcessMessage processes an message and transmits it to the platform.
func (p *outboundFilePlugin) ProcessMessage(ctx context.Context, msg *pb.Message) (*pb.Confirm, error) {
	return p.process(ctx, msg)
}

// ProcessAttachment processes an attachment and transmits it to the platform.
func (p *outboundFilePlugin) ProcessAttachment(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
	return p.process(ctx, atc)
}

func (p *outboundFilePlugin) convertToMessages(files []string) ([]*pb.Message, error) {
	messages := make([]*pb.Message, 0)
	for _, f := range files {
		buffer, err := ioutil.ReadFile(f)
		if err != nil {
			if err := backupFileToFolder(f, p.errorFolder); err != nil {
				log.Errorf("%v", err)
			}
			return nil, errors.Wrapf(err, "error while reading message %s", f)
		}
		messages = append(messages, &pb.Message{
			Id:        f,
			ProcessId: p.processID,
			Content:   buffer,
		})
	}
	return messages, nil
}

func (p *outboundFilePlugin) convertToAttachments(files []string) ([]*pb.Attachment, error) {
	attachments := make([]*pb.Attachment, 0)
	for _, f := range files {
		buffer, err := ioutil.ReadFile(f)
		if err != nil {
			if err := backupFileToFolder(f, p.errorFolder); err != nil {
				log.Errorf("%v", err)
			}
			return nil, errors.Wrapf(err, "error while reading attachment %s", f)
		}
		attachments = append(attachments, &pb.Attachment{
			Filename: f,
			Content: &pb.Attachment_Data{
				Data: buffer,
			},
		})
	}
	return attachments, nil
}

func (p *outboundFilePlugin) process(ctx context.Context, obj interface{}) (*pb.Confirm, error) {
	var file string
	var confirm *pb.Confirm
	var err error
	switch v := obj.(type) {
	case *pb.Message:
		file = v.Id
		if p.msgProcessor == nil {
			return nil, errors.New("message processor not provided")
		}
		confirm, err = p.msgProcessor(ctx, v)
	case *pb.Attachment:
		file = v.Filename
		if p.atcProcessor == nil {
			return nil, errors.New("attachment processor not provided")
		}
		confirm, err = p.atcProcessor(ctx, v)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "error while sending file %s", file)
	}
	if confirm == nil {
		return nil, fmt.Errorf("error no confirm received for file %s", file)
	}
	transport.PrintLogs(confirm.Logs)

	if !confirm.Success {
		if err := backupFileToFolder(file, p.errorFolder); err != nil {
			log.Errorf("%v", err)
		}
		var msgs []string
		for _, l := range confirm.Logs {
			msgs = append(msgs, l.Description)
		}
		return nil, fmt.Errorf("error from confirm for file %s: %d", confirm.Id, confirm.StatusCode)
	}
	if p.successFolder != "" {
		newfile := filepath.Join(p.successFolder, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(file)))

		if _, err := move(file, newfile); err != nil {
			return nil, errors.Wrapf(err, "error while moving file %s", file)
		}
		log.Infof("file %q moved to %q", file, newfile)
		return nil, nil
	}

	if err := os.Remove(file); err != nil {
		return nil, errors.Wrapf(err, "error while deleting file %s", file)
	}
	log.Infof("file '%s' deleted", file)

	return confirm, nil
}

// listFilesLastModifiedBefore lists all files last modified before t for path and extension
func listFilesLastModifiedBefore(path, extension string, t time.Time) ([]string, error) {
	files := []string{}

	log.Debugf("searching folder %s for files with extension %s", path, extension)

	fs, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read directory %s", path)
	}

	for _, f := range fs {
		if f == nil || f.IsDir() {
			continue
		}

		fp := filepath.Join(path, f.Name())
		if f.ModTime().Before(t) && strings.HasSuffix(fp, extension) {
			files = append(files, fp)
		}
	}

	return files, nil
}

// backupFileToFolder backups a file prefixed with current timestamp
func backupFileToFolder(filename, folder string) error {
	if filename == "" || folder == "" {
		return nil
	}

	f := filepath.Join(folder, fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(filename)))

	log.Debugf("trying to backup file %v to %v", filename, f)
	if _, err := move(filename, f); err != nil {
		return errors.Wrapf(err, "failed to backup file %s to %s", filename, f)
	}
	log.Infof("backuped %s to %s", filename, f)

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

	defer os.Remove(src)
	return io.Copy(out, in)
}
