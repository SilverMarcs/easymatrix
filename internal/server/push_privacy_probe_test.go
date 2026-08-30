package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestNormalizePushPrivacyTestURL(t *testing.T) {
	got, err := normalizePushPrivacyTestURL("https://relay-test.example/path?secret=nope")
	if err != nil {
		t.Fatalf("normalizePushPrivacyTestURL returned error: %v", err)
	}
	if want := "https://relay-test.example/_matrix/push/v1/notify"; got != want {
		t.Fatalf("URL = %q, want %q", got, want)
	}
	if _, err = normalizePushPrivacyTestURL("http://localhost:23373"); err == nil {
		t.Fatal("expected non-HTTPS URL to be rejected")
	}
}

func TestReceivePushPrivacyTestRecordsMetadataWithoutContent(t *testing.T) {
	service := newPushPrivacyTestService()
	service.configure("https://relay-test.example/_matrix/push/v1/notify", "route-secret")
	server := &Server{pushPrivacyTest: service}
	body := []byte(`{
		"notification": {
			"event_id": "$event",
			"room_id": "!room:example.org",
			"sender": "@sender:example.org",
			"content": {"body": "plaintext must not be retained"},
			"counts": {"unread": 3},
			"devices": [{
				"pushkey": "relay-push-privacy-test",
				"data": {"route_token": "route-secret"}
			}]
		}
	}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/_matrix/push/v1/notify", bytes.NewReader(body))
	if err := server.receivePushPrivacyTest(recorder, request); err != nil {
		t.Fatalf("receivePushPrivacyTest returned error: %v", err)
	}

	state := service.snapshot()
	if state.Last == nil {
		t.Fatal("expected notification metadata to be recorded")
	}
	if state.Last.EventID != "$event" || state.Last.RoomID != "!room:example.org" || state.Last.UnreadCount != 3 {
		t.Fatalf("unexpected notification state: %#v", state.Last)
	}
	if !state.Last.HadContent || !state.Last.HadSender {
		t.Fatalf("expected privacy violation flags, got %#v", state.Last)
	}
	wantFields := []string{"content", "counts", "devices", "event_id", "room_id", "sender"}
	if !reflect.DeepEqual(state.Last.Fields, wantFields) {
		t.Fatalf("fields = %#v, want %#v", state.Last.Fields, wantFields)
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to encode state: %v", err)
	}
	if bytes.Contains(encoded, []byte("plaintext must not be retained")) {
		t.Fatalf("state retained push content: %s", encoded)
	}
}

func TestReceivePushPrivacyTestRejectsUnknownRoute(t *testing.T) {
	service := newPushPrivacyTestService()
	service.configure("https://relay-test.example/_matrix/push/v1/notify", "expected-route")
	server := &Server{pushPrivacyTest: service}
	body := []byte(`{
		"notification": {
			"event_id": "$event",
			"devices": [{
				"pushkey": "relay-push-privacy-test",
				"data": {"route_token": "wrong-route"}
			}]
		}
	}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/_matrix/push/v1/notify", bytes.NewReader(body))
	if err := server.receivePushPrivacyTest(recorder, request); err != nil {
		t.Fatalf("receivePushPrivacyTest returned error: %v", err)
	}
	if service.snapshot().Last != nil {
		t.Fatal("unexpected metadata recorded for an unknown route")
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(pushPrivacyTestPushKey)) {
		t.Fatalf("expected rejected push key, got %s", recorder.Body.String())
	}
}
