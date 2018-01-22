package http

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	pb "github.com/myopenfactory/client/api"
	"github.com/myopenfactory/client/pkg/transport"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type inboundPlugin struct {
	method        string
	header        map[string]string
	responseRegex *regexp.Regexp
	msgTemplate   *template.Template
	atcTemplate   *template.Template
	client        HttpClient
}

// NewInboundPlugin creates an InboundPlugin with given parameters.
// `message_template` or `attachment_template` could be used to specify template files.
func NewInboundPlugin(parameter map[string]string) (transport.InboundPlugin, error) {
	p, err := parseParameter(parameter)
	if err != nil {
		return nil, err
	}
	p.client = &http.Client{}
	return p, nil
}

func parseParameter(parameter map[string]string) (*inboundPlugin, error) {
	cl := &inboundPlugin{}

	method, ok := parameter["method"]
	if !ok || method == "" {
		return nil, fmt.Errorf("no method found")
	}
	cl.method = method

	headers := make(map[string]string)
	header, ok := parameter["header"]
	pairs := strings.Split(header, ";")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			headers[kv[0]] = kv[1]
		}
	}
	cl.header = headers

	rep, ok := parameter["response"]
	if !ok || rep == "" {
		return nil, fmt.Errorf("no response regex found")
	}
	re, err := regexp.Compile(rep)
	if err != nil {
		return nil, errors.Wrap(err, "failed to compile response regesx")
	}
	cl.responseRegex = re

	tmpl, ok := parameter["message_template"]
	if ok && tmpl != "" {
		t, err := template.ParseFiles(tmpl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse message template")
		}
		cl.msgTemplate = t
	}

	tmpl, ok = parameter["attachment_template"]
	if ok && tmpl != "" {
		t, err := template.ParseFiles(tmpl)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse attachment template")
		}
		cl.atcTemplate = t
	}

	return cl, nil
}

// ProcessAttachment processes the message and invokes a http request on an remote endpoint.
func (p *inboundPlugin) ProcessMessage(ctx context.Context, msg *pb.Message) (*pb.Confirm, error) {
	if msg == nil {
		return nil, errors.New("failed to process message: nil message")
	}

	filename, ok := msg.Properties["filename"]
	if !ok || filename == "" {
		return nil, errors.New("failed to process message: filename not set")
	}

	data := new(bytes.Buffer)
	if p.msgTemplate != nil {
		if err := p.msgTemplate.Execute(data, msg); err != nil {
			return nil, errors.Wrap(err, "failed to execute message template")
		}
	} else {
		if _, err := data.Write(msg.GetContent()); err != nil {
			return nil, errors.Wrap(err, "failed to write message")
		}
	}

	req, err := http.NewRequest(p.method, filename, data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create message process request: %v", filename)
	}
	for k, v := range p.header {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke message process request")
	}
	if p.responseRegex == nil {
		return nil, errors.New("process message: no response regex")
	}
	if !p.responseRegex.MatchString(strconv.Itoa(resp.StatusCode)) {
		return nil, errors.Errorf("process message: bad response: %v", resp.Status)
	}

	return transport.CreateConfirm(msg.Id, msg.ProcessId, transport.StatusOK, "message pushed %q", filename)
}

// ProcessAttachment processes the attachment and invokes a http request on an remote endpoint.
func (p *inboundPlugin) ProcessAttachment(ctx context.Context, atc *pb.Attachment) (*pb.Confirm, error) {
	if atc == nil {
		return nil, errors.New("failed to process attachment: nil attachment")
	}

	data := new(bytes.Buffer)
	if p.atcTemplate != nil {
		if err := p.atcTemplate.Execute(data, atc); err != nil {
			return nil, errors.Wrap(err, "failed to execute attachment template")
		}
	} else {
		if _, err := data.Write(atc.GetData()); err != nil {
			return nil, errors.Wrap(err, "failed to write attachment")
		}
	}

	req, err := http.NewRequest(p.method, atc.Filename, data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create attachment process request: %v", atc.Filename)
	}
	for k, v := range p.header {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", atc.GetContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke attachment process request")
	}
	if p.responseRegex == nil {
		return nil, errors.New("process message: no response regex")
	}
	if !p.responseRegex.MatchString(strconv.Itoa(resp.StatusCode)) {
		return nil, errors.Errorf("process message: bad response: %v", resp.Status)
	}

	return transport.CreateConfirm(atc.Filename, "unknown", transport.StatusOK, "attachment pushed %q", atc.Filename)
}
