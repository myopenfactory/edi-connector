package file_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/myopenfactory/edi-connector/v2/transport"
	"github.com/myopenfactory/edi-connector/v2/transport/file"
)

func TestProcessMessage(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inboundDir := t.TempDir()
	inbound, err := file.NewInboundTransport(logger, "12345", map[string]any{
		"path": inboundDir,
	})
	if err != nil {
		t.Errorf("Failed to create inbound transport: %v", err)
	}

	statusMsg, err := inbound.ProcessMessage(context.TODO(), transport.Object{
		Id:      "78i7987129878921798",
		Content: []byte("test"),
		Metadata: map[string]string{
			"filename": "inbound.csv",
		},
	})
	if err != nil {
		t.Fatalf("Failed to process message: %v", err)
	}
	if !(strings.HasPrefix(statusMsg, "Created file:") && strings.HasSuffix(statusMsg, "inbound.csv")) {
		t.Errorf("Unexpected message, got: %v", statusMsg)
	}
	filePath := filepath.Join(inboundDir, "inbound.csv")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Expected inbound file to exist")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Could not read test file: %v", err)
	}

	expectedData := []byte("test")
	if !bytes.Equal(data, expectedData) {
		t.Errorf("Expected data: %s, got: %s", expectedData, data)
	}
}

func TestProcessMessageAppend(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inboundDir := t.TempDir()
	inbound, err := file.NewInboundTransport(logger, "12345", map[string]any{
		"path": inboundDir,
		"mode": "append",
	})
	if err != nil {
		t.Errorf("Failed to create inbound transport: %v", err)
	}

	filePath := filepath.Join(inboundDir, "inbound.csv")
	err = os.WriteFile(filePath, []byte("first line\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write existing inbound file: %v", err)
	}

	statusMsg, err := inbound.ProcessMessage(context.TODO(), transport.Object{
		Id:      "78i7987129878921798",
		Content: []byte("test"),
		Metadata: map[string]string{
			"filename": "inbound.csv",
		},
	})
	if err != nil {
		t.Fatalf("Failed to process message: %v", err)
	}
	if !(strings.HasPrefix(statusMsg, "Appending to file:") && strings.HasSuffix(statusMsg, "inbound.csv")) {
		t.Errorf("Unexpected message, got: %v", statusMsg)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Expected inbound file to exist")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Could not read test file: %v", err)
	}

	expectedData := []byte("first line\ntest")
	if !bytes.Equal(data, expectedData) {
		t.Errorf("Expected data: %s, got: %s", expectedData, data)
	}
}

func TestHandleAttachment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inboundDir := t.TempDir()
	attachmentDir := t.TempDir()
	inbound, err := file.NewInboundTransport(logger, "12345", map[string]any{
		"path":           inboundDir,
		"attachmentPath": attachmentDir,
		"attachmentWhitelist": []string{
			"http://whitelisted:8443/",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create inbound transport: %v", err)
	}

	handle := "https://test/attachment"
	if inbound.HandleAttachment(handle) {
		t.Errorf("Should not handle %s", handle)
	}

	handle = "http://whitelisted:8443/attachment"
	if !inbound.HandleAttachment(handle) {
		t.Errorf("Should handle %s", handle)
	}
}

func TestProcessAttachment(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	inboundDir := t.TempDir()
	attachmentDir := t.TempDir()
	inbound, err := file.NewInboundTransport(logger, "12345", map[string]any{
		"path":           inboundDir,
		"attachmentPath": attachmentDir,
	})
	if err != nil {
		t.Errorf("Failed to create inbound transport: %v", err)
	}

	err = inbound.ProcessAttachment(context.TODO(), transport.Object{
		Id:      "78i7987129878921798",
		Content: []byte("attachment"),
		Metadata: map[string]string{
			"filename": "attachment.pdf",
		},
	})
	if err != nil {
		t.Fatalf("Failed to process attachment: %v", err)
	}

	filePath := filepath.Join(attachmentDir, "attachment.pdf")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Expected inbound attachment to exist")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Could not read attachment file: %v", err)
	}

	expectedData := []byte("attachment")
	if !bytes.Equal(data, expectedData) {
		t.Errorf("Expected data: %s, got: %s", expectedData, data)
	}
}
