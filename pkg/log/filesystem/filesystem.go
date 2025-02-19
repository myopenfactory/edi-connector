package filesystem

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Handler struct {
	internalHandler slog.Handler
	file            *os.File
}

func NewHandler(folder string, options *slog.HandlerOptions) (*Handler, error) {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log folder")
	}

	file, err := os.Create(filepath.Join(folder, "edi.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	internalHandler := slog.NewJSONHandler(file, options)

	return &Handler{
		internalHandler: internalHandler,
		file:            file,
	}, nil
}

func (h *Handler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return h.internalHandler.Enabled(ctx, lvl)
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	return h.internalHandler.Handle(ctx, rec)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.internalHandler.WithAttrs(attrs)
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return h.internalHandler.WithGroup(name)
}
