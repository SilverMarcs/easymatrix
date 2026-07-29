package server

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"maunium.net/go/mautrix/event"
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

func TestGroupPushPayloadUsesThreeTextLevels(t *testing.T) {
	payload, ok := makePushPayload(compatRecord{
		"id":          "message-1",
		"chatID":      "chat-1",
		"senderName":  "Alice",
		"chatTitle":   "Weekend Plans",
		"isGroupChat": true,
		"text":        "Dinner at seven?",
	})
	if !ok {
		t.Fatal("makePushPayload() skipped an incoming group message")
	}

	aps := payload["aps"].(map[string]any)
	alert := aps["alert"].(map[string]string)
	if alert["title"] != "Weekend Plans" {
		t.Fatalf("title = %q, want group title", alert["title"])
	}
	if alert["subtitle"] != "Alice" {
		t.Fatalf("subtitle = %q, want sender name", alert["subtitle"])
	}
	if alert["body"] != "Dinner at seven?" {
		t.Fatalf("body = %q, want message body", alert["body"])
	}
}

func TestDirectPushPayloadOmitsSubtitle(t *testing.T) {
	payload, ok := makePushPayload(compatRecord{
		"id":          "message-1",
		"chatID":      "chat-1",
		"senderName":  "Alice",
		"chatTitle":   "Alice",
		"isGroupChat": false,
		"text":        "Hello",
	})
	if !ok {
		t.Fatal("makePushPayload() skipped an incoming direct message")
	}

	aps := payload["aps"].(map[string]any)
	alert := aps["alert"].(map[string]string)
	if alert["title"] != "Alice" {
		t.Fatalf("title = %q, want sender name", alert["title"])
	}
	if _, exists := alert["subtitle"]; exists {
		t.Fatal("direct message payload should not include a subtitle")
	}
}

func TestRoomAccountDataAllowsPushHonorsMuteState(t *testing.T) {
	tests := []struct {
		name       string
		mutedUntil int64
		want       bool
	}{
		{name: "unmuted", mutedUntil: 0, want: true},
		{name: "muted forever", mutedUntil: -1, want: false},
		{name: "future mute", mutedUntil: time.Now().Add(time.Hour).UnixMilli(), want: false},
		{name: "expired mute", mutedUntil: time.Now().Add(-time.Hour).UnixMilli(), want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, err := json.Marshal(event.BeeperMuteEventContent{
				MutedUntil: test.mutedUntil,
			})
			if err != nil {
				t.Fatalf("marshal mute content: %v", err)
			}
			accountData := []*database.AccountData{{
				Type:    event.AccountDataBeeperMute.Type,
				Content: content,
			}}

			if got := roomAccountDataAllowsPush(accountData); got != test.want {
				t.Fatalf("roomAccountDataAllowsPush() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestRoomAccountDataAllowsPushDefaultsToEnabled(t *testing.T) {
	if !roomAccountDataAllowsPush(nil) {
		t.Fatal("roomAccountDataAllowsPush() disabled push without mute account data")
	}
}
