package transport

import (
	"context"
)

type InboundSettings struct {
	AttachmentWhitelist []string `json:"attachmentWhitelist" yaml:"attachmentWhitelist"`
}

type Object struct {
	Id       string
	Content  []byte
	Metadata map[string]string
}

type ConfigInfo interface {
	AuthName() string
	ConfigId() string
}

type OutboundTransport interface {
	ConfigInfo
	ListMessages(ctx context.Context) ([]Object, error)
	ListAttachments(ctx context.Context) ([]Object, error)
}

type InboundTransport interface {
	ConfigInfo
	ProcessMessage(context.Context, Object) (string, error)
	ProcessAttachment(context.Context, Object) error
	// Return if attachment should be processed by specific processor
	HandleAttachment(url string) bool
}

type Finalizer interface {
	Finalize(context.Context, Object, error) error
}
