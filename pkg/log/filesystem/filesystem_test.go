package filesystem_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/myopenfactory/edi-connector/pkg/log/filesystem"
)

func TestHandler(t *testing.T) {
	logsFolder := t.TempDir()
	handler, err := filesystem.NewHandler(logsFolder, "test.log", 1, &slog.HandlerOptions{})
	if err != nil {
		t.Fatalf("Failed to initialize log handler: %v", err)
	}
	defer handler.Close()

	time, err := time.Parse(time.RFC3339, "2012-11-01T22:08:41+00:00")
	if err != nil {
		t.Fatalf("Failed to initialize record time: %v", err)
	}
	record := slog.NewRecord(time, slog.LevelInfo, "testlog", 0)
	err = handler.Handle(context.TODO(), record)
	if err != nil {
		t.Fatalf("Failed to handle log: %v", err)
	}

	logFile := filepath.Join(logsFolder, "test.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	expectedLog := append([]byte(`{"time":"2012-11-01T22:08:41Z","level":"INFO","msg":"testlog"}`), '\n')
	if !bytes.Equal(expectedLog, data) {
		t.Errorf("Expectd log to contain: %x, got: %x", expectedLog, data)
	}
}

func TestHandlerRotate(t *testing.T) {
	logsFolder := t.TempDir()
	handler, err := filesystem.NewHandler(logsFolder, "test.log", 1, &slog.HandlerOptions{})
	if err != nil {
		t.Fatalf("Failed to initialize log handler: %v", err)
	}
	defer handler.Close()

	time, err := time.Parse(time.RFC3339, "2012-11-01T22:08:41+00:00")
	if err != nil {
		t.Fatalf("Failed to initialize record time: %v", err)
	}
	record := slog.NewRecord(time, slog.LevelInfo, "testlog", 0)
	subHandler := handler.WithGroup("group").WithAttrs([]slog.Attr{
		slog.String("key", "value"),
	})
	err = subHandler.Handle(context.TODO(), record)
	if err != nil {
		t.Fatalf("Failed to handle log: %v", err)
	}

	logFile := filepath.Join(logsFolder, "test.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	expectedLog := append([]byte(`{"time":"2012-11-01T22:08:41Z","level":"INFO","msg":"testlog","group":{"key":"value"}}`), '\n')
	if !bytes.Equal(expectedLog, data) {
		t.Errorf("Expectd log to contain: %s, got: %s", expectedLog, data)
	}

	err = handler.Rotate()
	if err != nil {
		t.Fatalf("Failed rotating handler: %v", err)
	}

	record = slog.NewRecord(time, slog.LevelInfo, "after rotation", 0)
	err = subHandler.Handle(context.TODO(), record)
	if err != nil {
		t.Fatalf("Failed to handle log: %v", err)
	}

	logFile = filepath.Join(logsFolder, "test.log")
	data, err = os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	expectedLog = append([]byte(`{"time":"2012-11-01T22:08:41Z","level":"INFO","msg":"after rotation","group":{"key":"value"}}`), '\n')
	if !bytes.Equal(expectedLog, data) {
		t.Errorf("Expectd log to contain: %s, got: %s", expectedLog, data)
	}

	logFile = filepath.Join(logsFolder, "test.log.1")
	data, err = os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	expectedLog = append([]byte(`{"time":"2012-11-01T22:08:41Z","level":"INFO","msg":"testlog","group":{"key":"value"}}`), '\n')
	if !bytes.Equal(expectedLog, data) {
		t.Errorf("Expectd old log to contain: %s, got: %s", expectedLog, data)
	}
}

func TestHandlerRotateMax(t *testing.T) {
	logsFolder := t.TempDir()
	handler, err := filesystem.NewHandler(logsFolder, "test.log", 7, &slog.HandlerOptions{})
	if err != nil {
		t.Fatalf("Failed to initialize log handler: %v", err)
	}
	defer handler.Close()

	for i := 0; i < 10; i++ {
		err := handler.Rotate()
		if err != nil {
			t.Fatalf("Failed to rotate log handler: %v", err)
		}
	}

	entries, err := os.ReadDir(logsFolder)
	if err != nil {
		t.Fatalf("Failed to list logs folder: %v", err)
	}

	if len(entries) != 8 {
		t.Errorf("Expected %d log files, got: %d", 8, len(entries))
	}
}
