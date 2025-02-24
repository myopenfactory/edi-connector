package transport

import (
	"context"
)

type Object struct {
	Id       string
	Content  []byte
	Metadata map[string]string
}

type OutboundTransport interface {
	ListMessages(ctx context.Context) ([]Object, error)
}

type InboundTransport interface {
	ProcessMessage(context.Context, Object) error
}

type Finalizer interface {
	Finalize(context.Context, any, error) error
}

// AttachmentLister is the interface implemented by an plugin that supports listing attachments.
type AttachmentLister interface {
	// ListAttachments lists all found attachments.
	ListAttachments(ctx context.Context) ([]Object, error)
	// Return if attachments should be processed by specific processor
	HandleAttachments() bool
}

// Processor is the interface implemented by an transport that supports processing attachments.
type AttachmentProcessor interface {
	// ProcessAttachmetn processes an attachment.
	ProcessAttachment(context.Context, Object) error
	// Return if attachments should be processed by specific processor
	HandleAttachments() bool
}
