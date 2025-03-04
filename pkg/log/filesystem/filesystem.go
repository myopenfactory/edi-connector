package filesystem

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

type logWriter struct {
	m        sync.Mutex
	folder   string
	filename string
	file     *os.File
	count    int
	keep     int
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.m.Lock()
	defer w.m.Unlock()
	return w.file.Write(p)
}

func (w *logWriter) Rotate() error {
	w.m.Lock()
	defer w.m.Unlock()

	w.file.Close()

	oldFilename := filepath.Join(w.folder, w.filename)
	newFilename := filepath.Join(w.folder, w.filename+fmt.Sprintf(".%d", w.count+1))
	w.count++
	if w.count >= w.keep {
		w.count = 0
	}
	err := os.Rename(oldFilename, newFilename)
	if err != nil {
		return fmt.Errorf("failed to rename old file: %w", err)
	}

	file, err := os.Create(filepath.Join(w.folder, w.filename))
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	w.file = file

	return nil
}

func (w *logWriter) Close() error {
	return w.file.Close()
}

type Handler struct {
	options         *slog.HandlerOptions
	internalHandler slog.Handler
	logWriter       *logWriter
}

func NewHandler(folder string, filename string, keep int, options *slog.HandlerOptions) (*Handler, error) {
	if err := os.MkdirAll(folder, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log folder")
	}

	file, err := os.Create(filepath.Join(folder, filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	logWriter := &logWriter{
		m:        sync.Mutex{},
		folder:   folder,
		filename: filename,
		file:     file,
		keep:     keep,
	}

	internalHandler := slog.NewJSONHandler(logWriter, options)

	return &Handler{
		options:         options,
		internalHandler: internalHandler,
		logWriter:       logWriter,
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

func (h *Handler) Rotate() error {
	return h.logWriter.Rotate()
}

func (h *Handler) Close() error {
	return h.logWriter.Close()
}
