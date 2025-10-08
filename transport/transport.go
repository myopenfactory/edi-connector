package transport

import (
	"context"
)

type InboundSettings struct {
	AttachmentWhitelist []string
}

type Object struct {
	Id       string
	Content  []byte
	Metadata map[string]string
}

type OutboundTransport interface {
	ListMessages(ctx context.Context) ([]Object, error)
	ListAttachments(ctx context.Context) ([]Object, error)
}

type InboundTransport interface {
	ProcessMessage(context.Context, Object) (string, error)
	ProcessAttachment(context.Context, Object) error
	// Return if attachment should be processed by specific processor
	HandleAttachment(url string) bool
}

type Finalizer interface {
	Finalize(context.Context, Object, error) error
}
