package file

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gotest.tools/assert"

	"gotest.tools/fs"

	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/transport"

)

func TestNewInboundFilePlugin(t *testing.T) {
	dir := fs.NewDir(t, "client")
	defer dir.Remove()

	type args struct {
	}
	tests := []struct {
		name      string
		parameter map[string]string
		want      transport.InboundPlugin
		wantErr   bool
	}{
		{
			name:      "NilParameter",
			parameter: nil,
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "EmptyParameter",
			parameter: map[string]string{},
			want:      nil,
			wantErr:   true,
		},
		{
			name: "FolderNotExist",
			parameter: map[string]string{
				"basefolder": "1client2",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "BaseFolder",
			parameter: map[string]string{
				"basefolder": dir.Path(),
			},
			want: &inboundFilePlugin{
				base:  dir.Path(),
				exist: "count",
			},
			wantErr: false,
		},
		{
			name: "BaseFolderAppend",
			parameter: map[string]string{
				"basefolder": dir.Path(),
				"exist":      "append",
			},
			want: &inboundFilePlugin{
				base:  dir.Path(),
				exist: "append",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewInboundPlugin(tt.parameter)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewInboundFilePlugin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInboundFilePlugin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessAttachment(t *testing.T) {
	dir := fs.NewDir(t, "client")
	defer dir.Remove()

	p := &inboundFilePlugin{
		base: dir.Path(),
	}
	_, err := p.ProcessAttachment(context.Background(), &pb.Attachment{
		Filename: "test.txt",
		Content: &pb.Attachment_Data{
			Data: []byte("Hello World"),
		},
	})
	if err != nil {
		t.Fatalf("ProcessAttachment() error = %v, wantErr %v", err, false)
	}

	expected := fs.Expected(t, fs.WithFile("test.txt", "Hello World"))
	assert.Assert(t, fs.Equal(dir.Path(), expected))
}

func TestInboundFilePlugin_ConsumeMessage(t *testing.T) {
	testString := "Hello World"

	dir := fs.NewDir(t, "client",
		fs.WithFile("test.txt", ""),
		fs.WithDir("exist"),
		fs.WithFile("newfile2.txt", testString))
	defer dir.Remove()

	testName := "text.txt"
	tsAttachmentWrong := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, testString)
	}))
	defer tsAttachmentWrong.Close()

	tsAttachmentCorrect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Properties := r.FormValue("Properties")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\""+testName+"\"%s", Properties))
		fmt.Fprint(w, testString)
	}))
	defer tsAttachmentCorrect.Close()

	type file struct {
		filename, content string
	}

	tests := []struct {
		name       string
		args       *pb.Message
		want       *pb.Confirm
		wantErr    bool
		checkFiles []file
		count      string
	}{
		{
			name:    "ConsumNil",
			args:    nil,
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumAttachmentNilURL",
			args: &pb.Message{
				Attachments: []*pb.Attachment{
					{
						Filename: "testus.xml",
						Content:  &pb.Attachment_Url{},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumAttachmentNilFilename",
			args: &pb.Message{
				Attachments: []*pb.Attachment{
					{
						Filename: "",
						Content: &pb.Attachment_Url{
							Url: "http://localhost:9999/testus",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumAttachmentURLUnexpectedMedia",
			args: &pb.Message{
				Attachments: []*pb.Attachment{
					{
						Filename: "testus.xml",
						Content: &pb.Attachment_Url{
							Url: tsAttachmentCorrect.URL + "?Properties=?=)",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumAttachmentNoServer",
			args: &pb.Message{
				Attachments: []*pb.Attachment{
					{
						Filename: "testus.xml",
						Content: &pb.Attachment_Url{
							Url: "http://localhost:9999/testus",
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumAttachmentEmptyFile",
			args: &pb.Message{
				Attachments: []*pb.Attachment{
					{
						Content: &pb.Attachment_Url{
							Url: tsAttachmentWrong.URL,
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumAttachmentBadFile",
			args: &pb.Message{
				Attachments: []*pb.Attachment{
					{
						Filename: filepath.Join("test.txt", "test.txt"),
						Content: &pb.Attachment_Url{
							Url: tsAttachmentCorrect.URL,
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "ConsumMessageSuccessful",
			args: &pb.Message{
				ProcessId: "4711",
				Id:        "4811",
				Properties: map[string]string{
					"filename": "newfile.txt",
				},
				Content: []byte(testString),
			},
			want: &pb.Confirm{
				ProcessId: "4711",
				Id:        "4811",
				Success:   true,
				Logs: []*pb.Log{
					{Level: pb.Log_INFO, Description: fmt.Sprintf("file created with name %q", filepath.Join(dir.Path(), "newfile.txt"))},
				},
			},
			wantErr: false,
			checkFiles: []file{
				{
					content:  testString,
					filename: filepath.Join(dir.Path(), "newfile.txt"),
				},
			},
		},
		{
			name:  "ConsumMessageSuccessfulAppend",
			count: "append",
			args: &pb.Message{
				ProcessId: "4721",
				Id:        "4821",
				Properties: map[string]string{
					"filename": "newfile2.txt",
				},
				Content: []byte(testString),
			},
			want: &pb.Confirm{
				ProcessId: "4721",
				Id:        "4821",
				Success:   true,
				Logs: []*pb.Log{
					{Level: pb.Log_INFO, Description: fmt.Sprintf("message append to %s", filepath.Join(dir.Path(), "newfile2.txt"))},
				},
			},
			wantErr: false,
			checkFiles: []file{
				{
					content:  testString + testString,
					filename: filepath.Join(dir.Path(), "newfile2.txt"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := NewInboundPlugin(map[string]string{
				"basefolder": dir.Path(),
				"exist":      tt.count,
			})
			if err != nil {
				t.Fatalf("error whule creating newinboundfileplugin")
			}
			got, err := ip.ProcessMessage(context.Background(), tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("InboundFilePlugin.ConsumeMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InboundFilePlugin.ConsumeMessage() = %v, want %v", got, tt.want)
			}
			if len(tt.checkFiles) > 0 {
				for _, f := range tt.checkFiles {
					buf, err := ioutil.ReadFile(f.filename)
					if err != nil {
						t.Error(err)
					}
					content := string(buf)
					if f.content != content {
						t.Errorf("InboundFilePlugin.ConsumeMessage() = %v, want %v", content, f.content)
					}
				}
			}
		})
	}
}

func Test_createUniqueFilename(t *testing.T) {
	dir := fs.NewDir(t, "client", fs.WithFile("test.txt", ""))
	defer dir.Remove()

	file := fs.NewFile(t, "client")
	defer file.Remove()

	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "NilInput",
			args: "",
			want: "",
		},
		{
			name: "NonExists",
			args: filepath.Join(dir.Path(), "test2.txt"),
			want: filepath.Join(dir.Path(), "test2.txt"),
		},
		{
			name: "Exists",
			args: file.Path(),
			want: fmt.Sprintf("%s_1", file.Path()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createUniqueFilename(tt.args); got != tt.want {
				t.Errorf("createUniqueFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_createFolderFromFile(t *testing.T) {
	dir := fs.NewDir(t, "client", fs.WithFile("test.txt", ""))
	defer dir.Remove()

	tests := []struct {
		name     string
		filename string
		wantErr  bool
		basedir  string
	}{
		{
			name:     "EmptyFilename",
			filename: "",
			wantErr:  true,
		},
		{
			name:     "CreateNewFolder",
			filename: filepath.Join(dir.Path(), fmt.Sprintf("%d", rand.Int31()), "tmp.txt"),
			wantErr:  false,
		},
		{
			name:     "CreateExistingFolder",
			filename: filepath.Join(dir.Path(), "tmp.txt"),
			wantErr:  false,
		},
		{
			name:     "CreateExistingFolder",
			filename: filepath.Join(dir.Path(), "test.txt", "test.txt"),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := createFolderFromFile(tt.filename); (err != nil) != tt.wantErr {
				t.Errorf("createFolderFromFile() error = %v, wantErr %v", err, tt.wantErr)
			}
			_, err := os.Stat(filepath.Dir(tt.filename))
			if !tt.wantErr && os.IsNotExist(err) {
				t.Errorf("createFolderFromFile() error folder %v not exist", filepath.Dir(tt.filename))
			}
		})
	}
}
