package server

import (
	"path/filepath"
	"testing"
	"time"
)

func TestPushRegistrationsPersist(t *testing.T) {
	stateDir := t.TempDir()
	service := &pushService{
		devices:   make(map[string]pushDevice),
		storePath: filepath.Join(stateDir, "push", "devices.json"),
	}
	device := pushDevice{
		Token:     "aabbcc",
		Platform:  "apple",
		UpdatedAt: time.Now().UTC(),
	}
	if err := service.register(device); err != nil {
		t.Fatalf("register: %v", err)
	}

	reloaded := &pushService{
		devices:   make(map[string]pushDevice),
		storePath: service.storePath,
	}
	if err := reloaded.load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	tokens := reloaded.tokens()
	if len(tokens) != 1 || tokens[0] != device.Token {
		t.Fatalf("tokens = %v, want [%s]", tokens, device.Token)
	}

	if err := reloaded.delete(device.Token); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if len(reloaded.tokens()) != 0 {
		t.Fatalf("tokens were not deleted")
	}
}
