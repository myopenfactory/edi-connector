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

// OutboundPlugin is the interface that groups the basic List and Process methods.
type OutboundPlugin interface {
	MessageLister
}

// InboundPlugin describes a plugin which processes received messages and attachments.
type InboundPlugin interface {
	MessageProcessor
	HandleAttachment() bool
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

// Processor is the interface implemented by an Plugin that supports processing messages and attachments.
type MessageProcessor interface {
	// ProcessMessage processes an message.
	ProcessMessage(context.Context, Message) error
}

type AttachmentProcessor interface {
	// ProcessAttachmetn processes an attachment.
	ProcessAttachment(context.Context, Attachment) error
}
