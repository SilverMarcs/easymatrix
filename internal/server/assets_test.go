package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/batuhan/easymatrix/internal/config"
)

func TestRemoveUploadDeletesOnlySelectedEphemeralUpload(t *testing.T) {
	ephemeralRoot := t.TempDir()
	server := &Server{cfg: config.Config{EphemeralDir: ephemeralRoot}}
	uploadID := "safe-upload-id"
	uploadDir := filepath.Join(ephemeralRoot, "api-uploads", uploadID)
	if err := os.MkdirAll(uploadDir, 0o700); err != nil {
		t.Fatalf("failed to create upload directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(uploadDir, "file"), []byte("payload"), 0o600); err != nil {
		t.Fatalf("failed to create upload: %v", err)
	}

	if err := server.removeUpload(uploadID); err != nil {
		t.Fatalf("removeUpload returned error: %v", err)
	}
	if _, err := os.Stat(uploadDir); !os.IsNotExist(err) {
		t.Fatalf("upload directory still exists: %v", err)
	}
	if err := server.removeUpload("../outside"); err == nil {
		t.Fatal("expected unsafe upload ID to be rejected")
	}
}
