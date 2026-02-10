package file_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/myopenfactory/edi-connector/v2/transport"
	"github.com/myopenfactory/edi-connector/v2/transport/file"
)

func TestListMessages(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outboundDir := t.TempDir()
	waitTime := 100 * time.Millisecond
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound.txt"), []byte("outbound_txt"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound.test"), []byte("outbound_test"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	time.Sleep(waitTime)
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound.csv"), []byte("outbound_csv"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	time.Sleep(waitTime)
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound_noext"), []byte("outbound_noext"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	time.Sleep(waitTime)
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound_new.txt"), []byte("outbound_new_text"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	errorDir := t.TempDir()
	outbound, err := file.NewOutboundTransport(logger, "12345", "", map[string]any{
		"message": map[string]any{
			"path":       outboundDir,
			"extensions": []string{"txt", "csv", ""},
			"waitTime":   waitTime.String(),
		},
		"errorPath": errorDir,
	})
	if err != nil {
		t.Errorf("Failed to create outbound transport: %v", err)
	}
	messages, err := outbound.ListMessages(context.TODO())
	if err != nil {
		t.Errorf("Failed to list messages: %v", err)
	}
	expectedLength := 3
	if len(messages) != expectedLength {
		t.Fatalf("Expected %d messages, got: %d", expectedLength, len(messages))
	}

	message := messages[0]
	expectedContent := []byte("outbound_txt")
	if !bytes.Equal(message.Content, expectedContent) {
		t.Errorf("Expected content %s, got: %s", expectedContent, message.Content)
	}

	message = messages[1]
	expectedContent = []byte("outbound_csv")
	if !bytes.Equal(message.Content, expectedContent) {
		t.Errorf("Expected content %s, got: %s", expectedContent, message.Content)
	}

	message = messages[2]
	expectedContent = []byte("outbound_noext")
	if !bytes.Equal(message.Content, expectedContent) {
		t.Errorf("Expected content %s, got: %s", expectedContent, message.Content)
	}
}

func TestListAttachments(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outboundDir := t.TempDir()
	attachmentDir := t.TempDir()
	waitTime := 100 * time.Millisecond
	if err := os.WriteFile(filepath.Join(attachmentDir, "attachment.pdf"), []byte("attachment_pdf"), 0644); err != nil {
		t.Fatalf("Failed to create attachment file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(attachmentDir, "attachment.ignore"), []byte("attachment_ignore"), 0644); err != nil {
		t.Fatalf("Failed to create attachment file: %v", err)
	}
	time.Sleep(waitTime)
	if err := os.WriteFile(filepath.Join(attachmentDir, "attachment.step"), []byte("attachment_step"), 0644); err != nil {
		t.Fatalf("Failed to create attachment file: %v", err)
	}
	time.Sleep(waitTime)
	if err := os.WriteFile(filepath.Join(attachmentDir, "attachment_noext"), []byte("attachment_noext"), 0644); err != nil {
		t.Fatalf("Failed to create attachment file: %v", err)
	}
	time.Sleep(waitTime)
	if err := os.WriteFile(filepath.Join(attachmentDir, "attachment_new.pdf"), []byte("attachment_new_pdf"), 0644); err != nil {
		t.Fatalf("Failed to create attachment file: %v", err)
	}
	errorDir := t.TempDir()
	outbound, err := file.NewOutboundTransport(logger, "12345", "", map[string]any{
		"message": map[string]any{
			"path":       outboundDir,
			"extensions": []string{"txt"},
			"waitTime":   waitTime.String(),
		},
		"attachment": map[string]any{
			"path":       attachmentDir,
			"extensions": []string{"pdf", "step", ""},
			"waitTime":   waitTime.String(),
		},
		"errorPath": errorDir,
	})
	if err != nil {
		t.Fatalf("Failed to create outbound transport: %v", err)
	}
	attachments, err := outbound.ListAttachments(context.TODO())
	if err != nil {
		t.Errorf("Failed to list attachments: %v", err)
	}
	expectedLength := 3
	if len(attachments) != expectedLength {
		t.Fatalf("Expected %d attachments, got: %d", expectedLength, len(attachments))
	}

	attachment := attachments[0]
	expectedContent := []byte("attachment_pdf")
	if !bytes.Equal(attachment.Content, expectedContent) {
		t.Errorf("Expected content %s, got: %s", expectedContent, attachment.Content)
	}

	attachment = attachments[1]
	expectedContent = []byte("attachment_step")
	if !bytes.Equal(attachment.Content, expectedContent) {
		t.Errorf("Expected content %s, got: %s", expectedContent, attachment.Content)
	}

	attachment = attachments[2]
	expectedContent = []byte("attachment_noext")
	if !bytes.Equal(attachment.Content, expectedContent) {
		t.Errorf("Expected content %s, got: %s", expectedContent, attachment.Content)
	}
}

func TestFinalizeOnSuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outboundDir := t.TempDir()
	attachmentDir := t.TempDir()
	successDir := t.TempDir()
	waitTime := 100 * time.Millisecond
	outboundFilepath := filepath.Join(outboundDir, "outbound.txt")
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound.txt"), []byte("outbound_txt"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	errorDir := t.TempDir()
	outbound, err := file.NewOutboundTransport(logger, "12345", "", map[string]any{
		"message": map[string]any{
			"path":       outboundDir,
			"extensions": []string{"txt"},
			"waitTime":   waitTime.String(),
		},
		"attachment": map[string]any{
			"path":       attachmentDir,
			"extensions": []string{"pdf", "step"},
			"waitTime":   waitTime.String(),
		},
		"errorPath":   errorDir,
		"successPath": successDir,
	})
	if err != nil {
		t.Fatalf("Failed to create outbound transport: %v", err)
	}

	finalizer, ok := outbound.(transport.Finalizer)
	if !ok {
		t.Fatal("Expected finalizer")
	}

	err = finalizer.Finalize(context.TODO(), transport.Object{
		Id: outboundFilepath,
	}, nil)
	if err != nil {
		t.Fatalf("Failed to finalize outbound transport: %v", err)
	}

	if _, err := os.Stat(outboundFilepath); err == nil {
		t.Error("Expected file to not exist but still found it")
	}

	if _, err := os.Stat(filepath.Join(successDir, "outbound.txt")); os.IsNotExist(err) {
		t.Error("Expected file to be moved into success folder but did not find it")
	}
}

func TestFinalizeOnSuccessWithoutDirectory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outboundDir := t.TempDir()
	attachmentDir := t.TempDir()
	waitTime := 100 * time.Millisecond
	outboundFilepath := filepath.Join(outboundDir, "outbound.txt")
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound.txt"), []byte("outbound_txt"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	errorDir := t.TempDir()
	outbound, err := file.NewOutboundTransport(logger, "12345", "", map[string]any{
		"message": map[string]any{
			"path":       outboundDir,
			"extensions": []string{"txt"},
			"waitTime":   waitTime.String(),
		},
		"attachment": map[string]any{
			"path":       attachmentDir,
			"extensions": []string{"pdf", "step"},
			"waitTime":   waitTime.String(),
		},
		"errorPath": errorDir,
	})
	if err != nil {
		t.Fatalf("Failed to create outbound transport: %v", err)
	}

	finalizer, ok := outbound.(transport.Finalizer)
	if !ok {
		t.Fatal("Expected finalizer")
	}

	err = finalizer.Finalize(context.TODO(), transport.Object{
		Id: outboundFilepath,
	}, nil)
	if err != nil {
		t.Fatalf("Failed to finalize outbound transport: %v", err)
	}

	if _, err := os.Stat(outboundFilepath); err == nil {
		t.Error("Expected file to not exist but still found it")
	}
}

func TestFinalizeOnError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	outboundDir := t.TempDir()
	attachmentDir := t.TempDir()
	successDir := t.TempDir()
	waitTime := 100 * time.Millisecond
	outboundFilepath := filepath.Join(outboundDir, "outbound.txt")
	if err := os.WriteFile(filepath.Join(outboundDir, "outbound.txt"), []byte("outbound_txt"), 0644); err != nil {
		t.Fatalf("Failed to create outbound file: %v", err)
	}
	errorDir := t.TempDir()
	outbound, err := file.NewOutboundTransport(logger, "12345", "", map[string]any{
		"message": map[string]any{
			"path":       outboundDir,
			"extensions": []string{"txt"},
			"waitTime":   waitTime.String(),
		},
		"attachment": map[string]any{
			"path":       attachmentDir,
			"extensions": []string{"pdf", "step"},
			"waitTime":   waitTime.String(),
		},
		"errorPath":   errorDir,
		"successPath": successDir,
	})
	if err != nil {
		t.Fatalf("Failed to create outbound transport: %v", err)
	}

	finalizer, ok := outbound.(transport.Finalizer)
	if !ok {
		t.Fatal("Expected finalizer")
	}

	err = finalizer.Finalize(context.TODO(), transport.Object{
		Id: outboundFilepath,
	}, fmt.Errorf("fake error"))
	if err != nil {
		t.Fatalf("Failed to finalize outbound transport: %v", err)
	}

	if _, err := os.Stat(outboundFilepath); err == nil {
		t.Error("Expected file to not exist but still found it")
	}

	if _, err := os.Stat(filepath.Join(errorDir, "outbound.txt")); os.IsNotExist(err) {
		t.Error("Expected file to be moved into error folder but did not find it")
	}

	successEntries, err := os.ReadDir(successDir)
	if err != nil {
		t.Fatalf("Failed to list success dir: %v", err)
	}
	if len(successEntries) != 0 {
		t.Errorf("Expected %d success entries, got: %d", 0, len(successEntries))
	}
}
