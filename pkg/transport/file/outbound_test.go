package file

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gotest.tools/fs"

	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/transport"
	"github.com/pkg/errors"
)

func Test_outboundFilePlugin_backupFileToFolder(t *testing.T) {
	dir := fs.NewDir(t, "client",
		fs.WithDir("test"),
		fs.WithFile("test.txt", "Hello World"))
	defer dir.Remove()

	tests := []struct {
		name    string
		file    string
		folder  string
		wantErr bool
	}{
		{
			name:    "Nil",
			wantErr: false,
		},
		{
			name:    "NonExistingFile",
			file:    filepath.Join(dir.Path(), "bad.txt"),
			folder:  filepath.Join(dir.Path(), "test"),
			wantErr: true,
		},
		{
			name:    "NonExistingFolder",
			file:    filepath.Join(dir.Path(), "test.txt"),
			folder:  filepath.Join(dir.Path(), "bad"),
			wantErr: true,
		},
		{
			name:    "ExistingFileAndFolder",
			file:    filepath.Join(dir.Path(), "test.txt"),
			folder:  filepath.Join(dir.Path(), "test"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := backupFileToFolder(tt.file, tt.folder); (err != nil) != tt.wantErr {
				t.Errorf("outboundFilePlugin.backupFileToFolder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_outboundFilePlugin_listFilesLastModifiedBefore(t *testing.T) {
	ts := time.Now().Add(-30 * time.Second)
	opts := []fs.PathOp{
		fs.WithDir("empty"),
		fs.WithFile("test_0.csv", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_1.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_2.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_3.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_4.txt", "Hello World"),
		fs.WithFile("test_5.txt", "Hello World"),
		fs.WithFile("test_6.txt", "Hello World"),
	}
	dir := fs.NewDir(t, "client", opts...)
	defer dir.Remove()

	files := []string{
		filepath.Join(dir.Path(), "test_0.csv"),
		filepath.Join(dir.Path(), "test_1.txt"),
		filepath.Join(dir.Path(), "test_2.txt"),
		filepath.Join(dir.Path(), "test_3.txt"),
		filepath.Join(dir.Path(), "test_4.txt"),
		filepath.Join(dir.Path(), "test_5.txt"),
		filepath.Join(dir.Path(), "test_6.txt"),
	}

	tests := []struct {
		name      string
		path      string
		extension string
		time      time.Time
		want      []string
		wantErr   bool
	}{
		{
			name:    "NilInput",
			want:    []string{},
			wantErr: true,
		},
		{
			name:      "BadPath",
			path:      ":",
			extension: "txt",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "EmptyFolder",
			path:      filepath.Join(dir.Path(), "empty"),
			extension: "txt",
			want:      []string{},
			wantErr:   false,
		},
		{
			name:      "TxtFiles",
			path:      dir.Path(),
			extension: "txt",
			time:      time.Now().Add(-5 * time.Second),
			want:      files[1:4],
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := listFilesLastModifiedBefore(tt.path, tt.extension, tt.time)
			if (err != nil) != tt.wantErr {
				t.Errorf("outboundFilePlugin.listFilesModifiedBefore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("outboundFilePlugin.listFilesModifiedBefore() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockProcessor struct {
	confirm *pb.Confirm
	err     error
}

func (p *mockProcessor) ProcessMessage(ctx context.Context, msg *pb.Message) (*pb.Confirm, error) {
	return p.confirm, p.err
}

func (p *mockProcessor) ProcessAttachment(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
	return p.confirm, p.err
}

func TestProcess(t *testing.T) {
	errProcessor := &mockProcessor{
		err: errors.New("planned failing"),
	}

	dir := fs.NewDir(t, "client", fs.WithFile("move.txt", ""), fs.WithFile("remove.txt", ""))
	defer dir.Remove()

	tests := []struct {
		name          string
		msgProcessor  transport.MessageProcessor
		atcProcessor  transport.AttachmentProcessor
		ctx           context.Context
		obj           interface{}
		want          *pb.Confirm
		wantErr       bool
		successFolder string
	}{
		{
			name:         "EmptyMessage",
			msgProcessor: errProcessor.ProcessMessage,
			atcProcessor: errProcessor.ProcessAttachment,
			ctx:          context.Background(),
			obj:          &pb.Message{},
			wantErr:      true,
		},
		{
			name:         "EmptyAttachment",
			msgProcessor: errProcessor.ProcessMessage,
			atcProcessor: errProcessor.ProcessAttachment,
			ctx:          context.Background(),
			obj:          &pb.Attachment{},
			wantErr:      true,
		},
		{
			name:    "NoMsgProcessor",
			ctx:     context.Background(),
			obj:     &pb.Message{},
			wantErr: true,
		},
		{
			name:    "NoAtcProcessor",
			ctx:     context.Background(),
			obj:     &pb.Attachment{},
			wantErr: true,
		},
		{
			name: "NoConfirm",
			ctx:  context.Background(),
			obj:  &pb.Attachment{},
			atcProcessor: func(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
				return nil, nil
			},
			wantErr: true,
		},
		{
			name: "ConfirmNoSuccess",
			ctx:  context.Background(),
			obj:  &pb.Attachment{},
			atcProcessor: func(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
				return &pb.Confirm{
					Success: false,
				}, nil
			},
			wantErr: true,
		},
		{
			name: "ConfirmSuccessNoFile",
			ctx:  context.Background(),
			obj:  &pb.Attachment{},
			atcProcessor: func(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
				return &pb.Confirm{
					Success: true,
				}, nil
			},
			successFolder: dir.Path(),
			wantErr:       true,
		},
		{
			name: "ConfirmSuccessFile",
			ctx:  context.Background(),
			obj: &pb.Attachment{
				Filename: filepath.Join(dir.Path(), "move.txt"),
			},
			atcProcessor: func(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
				return &pb.Confirm{
					Success: true,
				}, nil
			},
			successFolder: dir.Path(),
			wantErr:       false,
		},
		{
			name: "ConfirmSuccessFileNoFolder",
			ctx:  context.Background(),
			obj: &pb.Attachment{
				Filename: filepath.Join(dir.Path(), "remove.txt"),
			},
			atcProcessor: func(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
				return &pb.Confirm{
					Success: true,
				}, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &outboundFilePlugin{
				msgProcessor:  tt.msgProcessor,
				atcProcessor:  tt.atcProcessor,
				successFolder: tt.successFolder,
			}
			got, err := p.process(tt.ctx, tt.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("process() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewOutboundFilePlugin(t *testing.T) {
	dir := fs.NewDir(t, "client")
	defer dir.Remove()

	type args struct {
		processid string
		parameter map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    transport.OutboundPlugin
		wantErr bool
	}{
		{
			name:    "NilInput",
			wantErr: true,
		},
		{
			name: "WithSuccessFolderNotExist",
			args: args{
				processid: "4711",
				parameter: map[string]string{
					"success": "/success",
				},
			},
			wantErr: true,
		},
		{
			name: "WithSuccessFolderExist",
			args: args{
				processid: "4711",
				parameter: map[string]string{
					"success": dir.Path(),
				},
			},
			want: &outboundFilePlugin{
				processID:     "4711",
				waitTime:      15,
				successFolder: dir.Path(),
				parameter: map[string]string{
					"success": dir.Path(),
				},
			},
			wantErr: false,
		},
		{
			name: "WithErrorFolderNotExist",
			args: args{
				processid: "4711",
				parameter: map[string]string{
					"error": "/error",
				},
			},
			wantErr: true,
		},
		{
			name: "WithErrorFolderExist",
			args: args{
				processid: "4711",
				parameter: map[string]string{
					"error": dir.Path(),
				},
			},
			want: &outboundFilePlugin{
				processID:   "4711",
				waitTime:    15,
				errorFolder: dir.Path(),
				parameter: map[string]string{
					"error": dir.Path(),
				},
			},
			wantErr: false,
		},
		{
			name: "WithWaittimeError",
			args: args{
				processid: "4711",
				parameter: map[string]string{
					"waittime": "13f",
				},
			},
			wantErr: true,
		},
		{
			name: "WithWaittime",
			args: args{
				processid: "4711",
				parameter: map[string]string{
					"waittime": "13",
				},
			},
			want: &outboundFilePlugin{
				processID: "4711",
				waitTime:  13,
				parameter: map[string]string{
					"waittime": "13",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewOutboundPlugin(tt.args.processid, nil, nil, tt.args.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOutboundPlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewOutboundPlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_outboundFilePlugin_ProcessMessage(t *testing.T) {
	mc := &pb.Confirm{
		Id: "testConfirm",
	}
	p := &outboundFilePlugin{
		msgProcessor: func(ctx context.Context, msg *pb.Message) (*pb.Confirm, error) {
			return mc, nil
		},
	}
	_, err := p.ProcessMessage(context.Background(), &pb.Message{})
	if err == nil {
		t.Error(err)
	}
}

func Test_outboundFilePlugin_ProcessAttachment(t *testing.T) {
	mc := &pb.Confirm{
		Id: "testConfirm",
	}
	p := &outboundFilePlugin{
		atcProcessor: func(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
			return mc, nil
		},
	}
	_, err := p.ProcessAttachment(context.Background(), &pb.Attachment{})
	if err == nil {
		t.Error(err)
	}
}

func Test_outboundFilePlugin_convertToMessages(t *testing.T) {
	ts := time.Now().Add(-30 * time.Second)
	opts := []fs.PathOp{
		fs.WithDir("error"),
		fs.WithFile("test_1.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_2.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_3.txt", "Hello World", fs.WithTimestamps(ts, ts)),
	}
	dir := fs.NewDir(t, "client", opts...)
	defer dir.Remove()

	files := []string{
		filepath.Join(dir.Path(), "test_1.txt"),
		filepath.Join(dir.Path(), "test_2.txt"),
		filepath.Join(dir.Path(), "test_3.txt"),
	}

	tests := []struct {
		name      string
		processID string
		files     []string
		want      []*pb.Message
		wantErr   bool
	}{
		{
			name:    "NilInput",
			wantErr: false,
			want:    []*pb.Message{},
		},
		{
			name:      "Messages",
			processID: "testID",
			files:     files,
			want: []*pb.Message{
				{
					Id:        files[0],
					ProcessId: "testID",
					Content:   []byte("Hello World"),
				},
				{
					Id:        files[1],
					ProcessId: "testID",
					Content:   []byte("Hello World"),
				},
				{
					Id:        files[2],
					ProcessId: "testID",
					Content:   []byte("Hello World"),
				},
			},
		},
		{
			name:      "InvalidFile",
			processID: "testID",
			files:     []string{"t89083"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &outboundFilePlugin{
				processID:   tt.processID,
				errorFolder: filepath.Join(dir.Path(), "error"),
			}
			Messages, err := p.convertToMessages(tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(Messages, tt.want) {
				t.Errorf("convertToMessages() = %v, want = %v", Messages, tt.want)
			}
		})
	}
}

func Test_outboundFilePlugin_convertToAttachments(t *testing.T) {
	ts := time.Now().Add(-30 * time.Second)
	opts := []fs.PathOp{
		fs.WithDir("error"),
		fs.WithFile("test_1.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_2.txt", "Hello World", fs.WithTimestamps(ts, ts)),
		fs.WithFile("test_3.txt", "Hello World", fs.WithTimestamps(ts, ts)),
	}
	dir := fs.NewDir(t, "client", opts...)
	defer dir.Remove()

	files := []string{
		filepath.Join(dir.Path(), "test_1.txt"),
		filepath.Join(dir.Path(), "test_2.txt"),
		filepath.Join(dir.Path(), "test_3.txt"),
	}

	tests := []struct {
		name      string
		processID string
		files     []string
		want      []*pb.Attachment
		wantErr   bool
	}{
		{
			name:    "NilInput",
			wantErr: false,
			want:    []*pb.Attachment{},
		},
		{
			name:      "Attachments",
			processID: "testID",
			files:     files,
			want: []*pb.Attachment{
				{
					Filename: files[0],
					Content: &pb.Attachment_Data{
						Data: []byte("Hello World"),
					},
				},
				{
					Filename: files[1],
					Content: &pb.Attachment_Data{
						Data: []byte("Hello World"),
					},
				},
				{
					Filename: files[2],
					Content: &pb.Attachment_Data{
						Data: []byte("Hello World"),
					},
				},
			},
		},
		{
			name:      "InvalidFile",
			processID: "testID",
			files:     []string{"t89083"},
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &outboundFilePlugin{
				processID:   tt.processID,
				errorFolder: filepath.Join(dir.Path(), "error"),
			}
			atcs, err := p.convertToAttachments(tt.files)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToAttachments() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(atcs, tt.want) {
				t.Errorf("convertToAttachments() = %v, want = %v", atcs, tt.want)
			}
		})
	}
}
