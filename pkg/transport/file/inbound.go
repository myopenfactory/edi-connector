package file

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/log"
	"github.com/myopenfactory/client/pkg/transport"
)

// InboundFilePlugin type
type inboundFilePlugin struct {
	base  string
	exist string
}

// NewInboundFilePlugin returns new InPlugin and checks for basefolder and exist parameter.
func NewInboundPlugin(parameter map[string]string) (transport.InboundPlugin, error) {
	base, ok := parameter["basefolder"]
	if !ok {
		return nil, fmt.Errorf("no basefolder found")
	}
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "folder %s does not exist", base)
	}
	exist := parameter["exist"]
	if exist != "append" {
		exist = "count"
	}
	log.Infof("using strategy %s for double messages", exist)
	return &inboundFilePlugin{
		base:  base,
		exist: exist,
	}, nil
}

// ConsumeMessage consumes message from plattform and saves it to a file
func (p *inboundFilePlugin) ProcessMessage(ctx context.Context, msg *pb.Message) (*pb.Confirm, error) {
	if msg == nil {
		return nil, errors.New("error Messages couldn't be nil")
	}

	filename, ok := msg.Properties["filename"]
	if !ok {
		return nil, fmt.Errorf("error filename is not set")
	}
	filename = filepath.Join(p.base, filename)
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) && p.exist == "append" {
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, errors.Wrapf(err, "error while open file %s", filename)
		}
		defer f.Close()
		_, err = f.Write(msg.Content)
		if err != nil {
			return nil, errors.Wrapf(err, "error while writing file %s", filename)
		}
		return transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusOK, "message append to %v", filename)
	}
	filename = createUniqueFilename(filename)
	if err := createFolderFromFile(filename); err != nil {
		return nil, errors.Wrapf(err, "error while creating message folder %s", filename)
	}
	log.Infof("Creating file '%v'", filename)
	if err := ioutil.WriteFile(filename, msg.Content, 0644); err != nil {
		return nil, errors.Wrapf(err, "error while writing file %s", filename)
	}
	return transport.CreateConfirm(msg.Id, msg.ProcessId, 200, "file created with name %q", filename)
}

// ProcessAttachment processes the attachment and writes it to specified path. In case of already existing file a
// new filename is derived.
func (p *inboundFilePlugin) ProcessAttachment(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
	filename := filepath.Join(p.base, atc.Filename)
	filename = createUniqueFilename(filename)
	if err := createFolderFromFile(filename); err != nil {
		return nil, errors.Wrapf(err, "error while creating attachment folder %s", filename)
	}
	f, err := os.Create(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open target file: %s", filename)
	}
	defer f.Close()

	_, err = f.Write(atc.GetData())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write attachment to file %q", filename)
	}

	return transport.CreateConfirm(atc.Filename, "unkown", transport.StatusOK, "attachment created with name %q", filename)
}

func createFolderFromFile(filename string) error {
	if filename == "" {
		return fmt.Errorf("error filename couldn't be nil")
	}
	folder := filepath.Dir(filename)
	if err := os.MkdirAll(folder, 755); err != nil {
		return errors.Wrapf(err, "error cannot create folder %s", folder)
	}
	return nil
}

func createUniqueFilename(fn string) string {
	if fn == "" {
		return ""
	}

	ext := filepath.Ext(fn)
	base := strings.TrimSuffix(fn, ext)

	_, err := os.Stat(fn)
	for i := 1; i < 10000; i++ {
		if os.IsNotExist(err) {
			break
		}
		fn = fmt.Sprintf("%s_%d%s", base, i, ext)
		_, err = os.Stat(fn)
	}

	return fn
}
