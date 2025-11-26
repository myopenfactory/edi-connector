package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/myopenfactory/edi-connector/v2/config"
	"github.com/myopenfactory/edi-connector/v2/pkg/log/eventlog"
	"github.com/myopenfactory/edi-connector/v2/pkg/log/filesystem"
)

func New() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func NewFromConfig(cfg config.LogOptions) (*slog.Logger, error) {
	var parsedLogLevel slog.Leveler
	switch cfg.Level {
	case "ERROR":
		parsedLogLevel = slog.LevelError
	case "INFO":
		parsedLogLevel = slog.LevelInfo
	case "DEBUG":
		parsedLogLevel = slog.LevelDebug
	default:
		parsedLogLevel = slog.LevelInfo
	}

	var logHandler slog.Handler
	switch cfg.Type {
	case "FILE":
		if cfg.Folder == "" {
			return nil, fmt.Errorf("need log folder to use file logging")
		}
		fileHandler, err := filesystem.NewHandler(cfg.Folder, "edi.log", 7, &slog.HandlerOptions{
			Level: parsedLogLevel,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize filesystem logging: %w", err)
		}
		go func() {
			interval := 24 * time.Hour
			now := time.Now()
			firstTick := now.Truncate(interval).Add(interval)
			<-time.After(firstTick.Sub(now))
			t := time.NewTicker(interval)
			for range t.C {
				fileHandler.Rotate()
			}
		}()
		logHandler = fileHandler
	case "EVENT":
		if runtime.GOOS != "windows" {
			return nil, fmt.Errorf("event log is only available on Windows")
		}
		var err error
		logHandler, err = eventlog.NewHandler("EDI-Connector", &slog.HandlerOptions{
			Level: parsedLogLevel,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to setup eventlog: %w", err)
		}
	case "STDOUT_TEXT":
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: parsedLogLevel,
		})
	case "STDOUT_JSON":
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: parsedLogLevel,
		})
	default:
		return nil, fmt.Errorf("unknown log type: %s", cfg.Type)
	}

	return slog.New(logHandler), nil
}
