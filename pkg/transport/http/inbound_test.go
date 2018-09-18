package http

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"text/template"

	"gotest.tools/fs"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"

	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/transport"
)

func TestNewInboundPlugin(t *testing.T) {
	_, err := NewInboundPlugin(nil)
	if err == nil {
		t.Errorf("NewInboundPlugin() = %v, want %v", err, nil)
	}

	params := map[string]string{
		"method":   http.MethodGet,
		"response": "200",
	}
	want := &inboundPlugin{
		method: params["method"],
		header: make(map[string]string),
	}
	p, err := NewInboundPlugin(params)
	if err != nil {
		t.Errorf("NewInboundPlugin() = %v, want %v", err, nil)
	}
	assert.DeepEqual(t, p, want, cmpopts.IgnoreUnexported(inboundPlugin{}))
}

func TestParseParameter(t *testing.T) {
	tests := []struct {
		name      string
		parameter map[string]string
		want      *inboundPlugin
		wantErr   bool
	}{
		{
			name:    "NilInput",
			wantErr: true,
		},
		{
			name: "NoMethod",
			parameter: map[string]string{
				"method": "",
			},
			wantErr: true,
		},
		{
			name: "NoRegex",
			parameter: map[string]string{
				"method": http.MethodGet,
			},
			wantErr: true,
		},
		{
			name: "Header",
			parameter: map[string]string{
				"method":   http.MethodGet,
				"response": "200",
				"header":   "Authorization:Bearer:AAA;Content-Type:text/html;Empty:",
			},
			want: &inboundPlugin{
				method:        http.MethodGet,
				responseRegex: regexp.MustCompile("200"),
				header: map[string]string{
					"Authorization": "Bearer:AAA",
					"Content-Type":  "text/html",
					"Empty":         "",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseParameter(tt.parameter)

			if !tt.wantErr {
				assert.NilError(t, err)
			}
			if tt.want != nil {
				assert.DeepEqual(t, got.method, tt.want.method)
				assert.DeepEqual(t, got.header, tt.want.header)
			}
		})
	}

	tmplText := `{{ .Test }}`
	tmplFile := fs.NewFile(t, "client", fs.WithContent(tmplText))
	defer tmplFile.Remove()

	tmpl, err := template.ParseFiles(tmplFile.Path())
	if err != nil {
		t.Errorf("failed to parse input template: %v", err)
	}

	data := struct {
		Test string
	}{
		"TestMessage",
	}

	want := new(bytes.Buffer)
	if err := tmpl.Execute(want, data); err != nil {
		t.Errorf("failed to execute input template: %v", err)
	}

	got, err := parseParameter(map[string]string{
		"method":              "GET",
		"response":            "200",
		"message_template":    tmplFile.Path(),
		"attachment_template": tmplFile.Path(),
	})
	assert.NilError(t, err)

	msgGot := new(bytes.Buffer)
	if err := got.msgTemplate.Execute(msgGot, data); err != nil {
		t.Errorf("failed to execute message template: %v", err)
	}

	atcGot := new(bytes.Buffer)
	if err := got.atcTemplate.Execute(atcGot, data); err != nil {
		t.Errorf("failed to execute attachment template: %v", err)
	}

	comparator := func(got, want *bytes.Buffer) cmp.Comparison {
		return func() cmp.Result {
			x := got.Bytes()
			y := want.Bytes()
			if bytes.Compare(x, y) == 0 {
				return cmp.ResultSuccess
			}
			return cmp.ResultFailure(
				fmt.Sprintf("%q dit not match %q", x, y))
		}
	}

	assert.Assert(t, comparator(msgGot, want))
	assert.Assert(t, comparator(atcGot, want))
}

type testHandler struct {
	status int
	msg    string
	err    error
	data   []byte
}

func (h *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.err != nil {
		http.Error(w, h.err.Error(), h.status)
		return
	}
	defer r.Body.Close()

	var err error
	h.data, err = ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintln(w, h.msg)
}

func TestProcessMessage(t *testing.T) {
	th := &testHandler{}
	srv := httptest.NewServer(th)

	p := &inboundPlugin{
		method: http.MethodGet,
		client: srv.Client(),
	}

	tests := []struct {
		name        string
		method      string
		response    string
		header      map[string]string
		handler     *testHandler
		msg         *pb.Message
		want        *pb.Confirm
		wantErr     bool
		containsErr string
	}{
		{
			name:    "NilInput",
			wantErr: true,
		},
		{
			name: "EmptyFilename",
			msg: &pb.Message{
				Properties: map[string]string{
					"filename": "",
				},
			},
			wantErr: true,
		},
		{
			name:   "BadMethod",
			method: "'%=`4#",
			msg: &pb.Message{
				Properties: map[string]string{
					"filename": srv.URL,
				},
			},
			wantErr:     true,
			containsErr: "failed to create message process request",
		},
		{
			name:     "HttpNotFound",
			method:   http.MethodGet,
			response: "200",
			handler: &testHandler{
				status: http.StatusNotFound,
				err:    errors.New(http.StatusText(http.StatusNotFound)),
			},
			msg: &pb.Message{
				Properties: map[string]string{
					"filename": srv.URL,
				},
			},
			wantErr:     true,
			containsErr: http.StatusText(http.StatusNotFound),
		},
		{
			name:     "HttpOk",
			method:   http.MethodGet,
			response: "200",
			header: map[string]string{
				"Authorization": "Bearer",
			},
			handler: &testHandler{
				status: http.StatusOK,
				msg:    "Data",
			},
			msg: &pb.Message{
				Id:        "test",
				ProcessId: "testus",
				Properties: map[string]string{
					"filename": srv.URL,
				},
			},
			want: &pb.Confirm{
				Id:         "test",
				ProcessId:  "testus",
				Logs:       transport.AddLog([]*pb.Log{}, pb.Log_INFO, "message pushed %q", srv.URL),
				Success:    true,
				StatusCode: 200,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.method != "" {
				p.method = tt.method
			}
			if tt.response != "" {
				p.responseRegex = regexp.MustCompile(tt.response)
			}
			if tt.header != nil {
				p.header = tt.header
			}
			if tt.handler != nil {
				th.status = tt.handler.status
				th.err = tt.handler.err
				th.msg = tt.handler.msg
			}
			got, err := p.ProcessMessage(context.Background(), tt.msg)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.containsErr)
			} else {
				assert.NilError(t, err)
			}
			assert.DeepEqual(t, got, tt.want)
		})
	}

	file := fs.NewFile(t, "client", fs.WithContent(`{{ index .Properties "before" }}{{ .Content | printf "%s" }}{{ index .Properties "after" }}`))
	defer file.Remove()

	p = &inboundPlugin{
		method:        "POST",
		client:        srv.Client(),
		msgTemplate:   template.Must(template.ParseFiles(file.Path())),
		responseRegex: regexp.MustCompile("200"),
	}

	msg := &pb.Message{
		Id:        "test",
		ProcessId: "testus",
		Properties: map[string]string{
			"filename": srv.URL,
			"before":   "<item>",
			"after":    "</item>",
		},
		Content: []byte("Hello World"),
	}

	_, err := p.ProcessMessage(context.Background(), msg)
	assert.NilError(t, err)

	comparator := func(got, want []byte) cmp.Comparison {
		return func() cmp.Result {
			if bytes.Compare(got, want) == 0 {
				return cmp.ResultSuccess
			}
			return cmp.ResultFailure(
				fmt.Sprintf("%q dit not match %q", got, want))
		}
	}

	assert.Assert(t, comparator(th.data, []byte("<item>Hello World</item>")))
}

func TestProcessAttachment(t *testing.T) {
	th := &testHandler{}
	srv := httptest.NewServer(th)

	p := &inboundPlugin{
		method: "GET",
		client: srv.Client(),
	}

	tests := []struct {
		name        string
		method      string
		response    string
		header      map[string]string
		handler     *testHandler
		atc         *pb.Attachment
		want        *pb.Confirm
		wantErr     bool
		containsErr string
	}{
		{
			name:    "NilInput",
			wantErr: true,
		},
		{
			name: "EmptyFilename",
			atc: &pb.Attachment{
				Filename: "",
			},
			wantErr: true,
		},
		{
			name:   "BadMethod",
			method: "'%=`4#",
			atc: &pb.Attachment{
				Filename: srv.URL,
			},
			wantErr:     true,
			containsErr: "failed to create attachment process request",
		},
		{
			name:     "HttpNotFound",
			method:   http.MethodGet,
			response: "200",
			handler: &testHandler{
				status: http.StatusNotFound,
				err:    errors.New(http.StatusText(http.StatusNotFound)),
			},
			atc: &pb.Attachment{
				Filename: srv.URL,
			},
			wantErr:     true,
			containsErr: http.StatusText(http.StatusNotFound),
		},
		{
			name:     "HttpOk",
			method:   "GET",
			response: "200",
			header: map[string]string{
				"Authorization": "Bearer",
			},
			handler: &testHandler{
				status: http.StatusOK,
				msg:    "Data",
			},
			atc: &pb.Attachment{
				Filename: srv.URL,
			},
			want: &pb.Confirm{
				Id:         srv.URL,
				ProcessId:  "unknown",
				Logs:       transport.AddLog([]*pb.Log{}, pb.Log_INFO, "attachment pushed %q", srv.URL),
				Success:    true,
				StatusCode: 200,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.method != "" {
				p.method = tt.method
			}
			if tt.response != "" {
				p.responseRegex = regexp.MustCompile(tt.response)
			}
			if tt.header != nil {
				p.header = tt.header
			}
			if tt.handler != nil {
				th.status = tt.handler.status
				th.err = tt.handler.err
				th.msg = tt.handler.msg
			}
			got, err := p.ProcessAttachment(context.Background(), tt.atc)
			if tt.wantErr {
				assert.ErrorContains(t, err, tt.containsErr)
			} else {
				assert.NilError(t, err)
			}
			assert.DeepEqual(t, got, tt.want)
		})
	}

	file := fs.NewFile(t, "client", fs.WithContent(`<item>{{ .Content.Data | printf "%s" }}</item>`))
	defer file.Remove()

	p = &inboundPlugin{
		method:        "POST",
		client:        srv.Client(),
		atcTemplate:   template.Must(template.ParseFiles(file.Path())),
		responseRegex: regexp.MustCompile("200"),
	}

	atc := &pb.Attachment{
		Filename: srv.URL,
		Content: &pb.Attachment_Data{
			Data: []byte("Hello World"),
		},
	}

	_, err := p.ProcessAttachment(context.Background(), atc)
	assert.NilError(t, err)

	comparator := func(got, want []byte) cmp.Comparison {
		return func() cmp.Result {
			if bytes.Compare(got, want) == 0 {
				return cmp.ResultSuccess
			}
			return cmp.ResultFailure(
				fmt.Sprintf("%q dit not match %q", got, want))
		}
	}

	assert.Assert(t, comparator(th.data, []byte("<item>Hello World</item>")))
}
