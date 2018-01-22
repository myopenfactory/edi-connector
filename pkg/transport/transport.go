package transport

import (
	"context"

	pb "github.com/myopenfactory/client/api"
)

const (
	StatusOK            int32 = 200
	StatusDuplicate           = 409
	StatusMsgSizeToBig        = 413
	StatusInternalError       = 500
)

// MessageProcessor describes a function which takes an message and transmits it.
type MessageProcessor func(context.Context, *pb.Message) (*pb.Confirm, error)

// AttachmentProcessor describes a function which takes an attachment and transmits it.
type AttachmentProcessor func(context.Context, *pb.Attachment) (*pb.Confirm, error)

// InboundPlugin describes a plugin which processes received messages and attachments.
type InboundPlugin interface {
	Processor
}

// OutboundPlugin is the interface that groups the basic List and Process methods.
type OutboundPlugin interface {
	Lister
	Processor
}

// Lister is the interface implemented by an Plugin that supports listing messages and attachments.
type Lister interface {
	// ListMessages lists all found messages.
	ListMessages(ctx context.Context) ([]*pb.Message, error)
	// ListAttachments lists all found attachments.
	ListAttachments(ctx context.Context) ([]*pb.Attachment, error)
}

// Processor is the interface implemented by an Plugin that supports processing messages and attachments.
type Processor interface {
	// ProcessMessage processes an message.
	ProcessMessage(context.Context, *pb.Message) (*pb.Confirm, error)
	// ProcessAttachmetn processes an attachment.
	ProcessAttachment(context.Context, *pb.Attachment) (*pb.Confirm, error)
}
