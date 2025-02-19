package eventlog

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"golang.org/x/sys/windows/svc/eventlog"
)

type Handler struct {
	opts *slog.HandlerOptions
	goas []groupOrAttrs
	mu   *sync.Mutex
	log  *eventlog.Log
}

func NewHandler(name string, opts *slog.HandlerOptions) (*Handler, error) {
	log, err := eventlog.Open(name)
	if err != nil {
		return nil, fmt.Errorf("failed to open eventlog: %w", err)
	}
	return &Handler{
		opts: opts,
		goas: make([]groupOrAttrs, 0),
		mu:   &sync.Mutex{},
		log:  log,
	}, nil
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	buf := make([]byte, 0, 1024)
	if h.opts.AddSource && rec.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{rec.PC})
		f, _ := fs.Next()
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", f.File, f.Line)))
	}
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, rec.Message))
	goas := h.goas
	if rec.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list; they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}

	currentGroup := ""
	for _, goa := range goas {
		if goa.group != "" {
			if currentGroup == "" {
				currentGroup = goa.group
			} else {
				currentGroup = currentGroup + "." + goa.group
			}
		}
	}

	for _, goa := range goas {
		for _, a := range goa.attrs {
			if currentGroup != "" {
				a.Key = currentGroup + "." + a.Key
			}
			buf = h.appendAttr(buf, a)
		}
	}

	rec.Attrs(func(a slog.Attr) bool {
		if currentGroup != "" {
			a.Key = currentGroup + "." + a.Key
		}
		buf = h.appendAttr(buf, a)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()
	var err error
	text := string(buf)
	switch rec.Level {
	case slog.LevelDebug:
		// not supported by eventlog
	case slog.LevelInfo:
		err = h.log.Info(1, text)
	case slog.LevelError:
		err = h.log.Error(1, text)
	case slog.LevelWarn:
		err = h.log.Warning(1, text)

	}
	return err
}

func (h *Handler) appendAttr(buf []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}
	switch a.Value.Kind() {
	case slog.KindString:
		buf = fmt.Appendf(buf, "%s=%q ", a.Key, a.Value.String())
	case slog.KindTime:
		// ignore time eventlog has it's own time
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}
		// If the key is non-empty, write it out and indent the rest of the attrs.
		// Otherwise, inline the attrs.
		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s", a.Key)
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
	default:
		buf = fmt.Appendf(buf, "%s=%q ", a.Key, a.Value)
	}
	return buf
}

func (h *Handler) withGroupOrAttrs(goa groupOrAttrs) *Handler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})

}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}
