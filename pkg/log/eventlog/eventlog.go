//go:build !windows

package eventlog

import (
	"context"
	"fmt"
	"log/slog"
)

type Handler struct{}

func NewHandler(name string, opts *slog.HandlerOptions) (*Handler, error) {
	return nil, fmt.Errorf("not supported on unix")
}

func (h *Handler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return false
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	return fmt.Errorf("not implemented")
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return nil
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return nil
}
