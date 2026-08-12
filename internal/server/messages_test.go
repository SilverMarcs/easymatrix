package server

import (
	"testing"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"maunium.net/go/mautrix/event"

	"github.com/batuhan/easymatrix/internal/compat"
)

func TestApplyStoredMessageContentRetainsRedactedText(t *testing.T) {
	evt := &database.Event{
		Type:       event.EventMessage.Type,
		Content:    []byte(`{"msgtype":"m.text","body":"Retained text"}`),
		RedactedBy: "$redaction",
	}
	message := compat.Message{IsDeleted: evt.RedactedBy != ""}

	if err := applyStoredMessageContent(&message, evt, event.EventMessage.Type); err != nil {
		t.Fatalf("apply stored message content: %v", err)
	}
	if !message.IsDeleted {
		t.Fatal("expected message to remain marked deleted")
	}
	if message.Text != "Retained text" {
		t.Fatalf("expected retained text, got %q", message.Text)
	}
}
