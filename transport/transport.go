package transport

import (
	"context"
)

type Message struct {
	Id       string
	Content  []byte
	Metadata map[string]string
}

type Attachment struct {
	Filename string
	Content  []byte
}

type Finalizer interface {
	Finalize(context.Context, any, error) error
}

type OutboundTransport interface {
	MessageLister
}

type InboundTransport interface {
	MessageProcessor
}

// MessageLister is the interface implemented by an plugin that supports listing messages.
type MessageLister interface {
	// ListMessages lists all found messages.
	ListMessages(ctx context.Context) ([]Message, error)
}

// AttachmentLister is the interface implemented by an plugin that supports listing attachments.
type AttachmentLister interface {
	// ListAttachments lists all found attachments.
	ListAttachments(ctx context.Context) ([]Attachment, error)
}

// Processor is the interface implemented by an transport that supports processing messages.
type MessageProcessor interface {
	// ProcessMessage processes an message.
	ProcessMessage(context.Context, Message) error
}

// Processor is the interface implemented by an transport that supports processing attachments.
type AttachmentProcessor interface {
	// ProcessAttachmetn processes an attachment.
	ProcessAttachment(context.Context, Attachment) error
	// Return if attachments should be processed by specific processor
	HandleAttachments() bool
}
