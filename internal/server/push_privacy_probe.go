package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"

	errs "github.com/batuhan/easymatrix/internal/errors"
)

const (
	pushPrivacyTestAppID   = "com.silvermarcs.relay.push-privacy-test"
	pushPrivacyTestPushKey = "relay-push-privacy-test"
)

type pushPrivacyTestService struct {
	mu           sync.RWMutex
	registered   bool
	gatewayURL   string
	routeToken   string
	registeredAt time.Time
	last         *pushPrivacyTestNotification
}

type pushPrivacyTestNotification struct {
	ReceivedAt  time.Time `json:"receivedAt"`
	EventID     string    `json:"eventID,omitempty"`
	RoomID      string    `json:"roomID,omitempty"`
	UnreadCount int       `json:"unreadCount,omitempty"`
	DeviceCount int       `json:"deviceCount"`
	Fields      []string  `json:"fields"`
	HadContent  bool      `json:"hadContent"`
	HadSender   bool      `json:"hadSender"`
}

type pushPrivacyTestOutput struct {
	Registered   bool                         `json:"registered"`
	GatewayURL   string                       `json:"gatewayURL,omitempty"`
	RegisteredAt *time.Time                   `json:"registeredAt,omitempty"`
	Last         *pushPrivacyTestNotification `json:"lastNotification,omitempty"`
}

type matrixPushNotificationRequest struct {
	Notification struct {
		EventID string          `json:"event_id"`
		RoomID  string          `json:"room_id"`
		Sender  string          `json:"sender"`
		Content json.RawMessage `json:"content"`
		Counts  struct {
			Unread int `json:"unread"`
		} `json:"counts"`
		Devices []struct {
			PushKey string `json:"pushkey"`
			Data    struct {
				RouteToken string `json:"route_token"`
			} `json:"data"`
		} `json:"devices"`
	} `json:"notification"`
}

func newPushPrivacyTestService() *pushPrivacyTestService {
	return &pushPrivacyTestService{}
}

func (p *pushPrivacyTestService) snapshot() pushPrivacyTestOutput {
	p.mu.RLock()
	defer p.mu.RUnlock()
	output := pushPrivacyTestOutput{
		Registered: p.registered,
		GatewayURL: p.gatewayURL,
	}
	if !p.registeredAt.IsZero() {
		registeredAt := p.registeredAt
		output.RegisteredAt = &registeredAt
	}
	if p.last != nil {
		last := *p.last
		last.Fields = append([]string(nil), p.last.Fields...)
		output.Last = &last
	}
	return output
}

func (p *pushPrivacyTestService) configure(gatewayURL, routeToken string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.registered = true
	p.gatewayURL = gatewayURL
	p.routeToken = routeToken
	p.registeredAt = time.Now().UTC()
	p.last = nil
}

func (p *pushPrivacyTestService) clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.registered = false
	p.gatewayURL = ""
	p.routeToken = ""
	p.registeredAt = time.Time{}
	p.last = nil
}

func (p *pushPrivacyTestService) record(input matrixPushNotificationRequest, fields []string) []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	rejected := make([]string, 0)
	authorizedDevices := 0
	for _, device := range input.Notification.Devices {
		if p.registered && device.PushKey == pushPrivacyTestPushKey && device.Data.RouteToken == p.routeToken {
			authorizedDevices++
		} else if device.PushKey != "" {
			rejected = append(rejected, device.PushKey)
		}
	}
	if authorizedDevices > 0 {
		p.last = &pushPrivacyTestNotification{
			ReceivedAt:  time.Now().UTC(),
			EventID:     input.Notification.EventID,
			RoomID:      input.Notification.RoomID,
			UnreadCount: input.Notification.Counts.Unread,
			DeviceCount: authorizedDevices,
			Fields:      append([]string(nil), fields...),
			HadContent:  len(input.Notification.Content) > 0 && string(input.Notification.Content) != "null",
			HadSender:   strings.TrimSpace(input.Notification.Sender) != "",
		}
	}
	return rejected
}

func (s *Server) getPushPrivacyTest(w http.ResponseWriter, _ *http.Request) error {
	return writeJSON(w, s.pushPrivacyTest.snapshot())
}

func (s *Server) registerPushPrivacyTest(w http.ResponseWriter, r *http.Request) error {
	var input struct {
		GatewayURL string `json:"gatewayURL"`
	}
	if err := decodeJSON(r, &input); err != nil {
		return err
	}
	gatewayURL, err := normalizePushPrivacyTestURL(input.GatewayURL)
	if err != nil {
		return errs.Validation(map[string]any{"gatewayURL": err.Error()})
	}
	cli := s.rt.Client()
	if cli == nil || cli.Client == nil || cli.Account == nil {
		return errs.Forbidden("A logged-in Matrix session is required")
	}
	routeToken := randomID()
	kind := "http"
	request := map[string]any{
		"app_display_name":    "Relay Push Privacy Test",
		"app_id":              pushPrivacyTestAppID,
		"append":              false,
		"device_display_name": "Relay test device",
		"kind":                kind,
		"lang":                "en",
		"pushkey":             pushPrivacyTestPushKey,
		"data": map[string]any{
			"url":         gatewayURL,
			"format":      "event_id_only",
			"route_token": routeToken,
			"default_payload": map[string]any{
				"aps": map[string]any{
					"alert":           map[string]any{"title": "Relay", "body": "New message"},
					"mutable-content": 1,
				},
			},
		},
	}
	endpoint := cli.Client.BuildClientURL("v3", "pushers", "set")
	if _, err = cli.Client.MakeFullRequest(r.Context(), mautrix.FullRequest{
		Method:           http.MethodPost,
		URL:              endpoint,
		RequestJSON:      request,
		SensitiveContent: true,
	}); err != nil {
		return errs.Internal(fmt.Errorf("Beeper rejected the test pusher: %w", err))
	}
	s.pushPrivacyTest.configure(gatewayURL, routeToken)
	return writeJSON(w, s.pushPrivacyTest.snapshot())
}

func (s *Server) removePushPrivacyTest(w http.ResponseWriter, r *http.Request) error {
	cli := s.rt.Client()
	if cli == nil || cli.Client == nil || cli.Account == nil {
		return errs.Forbidden("A logged-in Matrix session is required")
	}
	request := map[string]any{
		"app_id":  pushPrivacyTestAppID,
		"pushkey": pushPrivacyTestPushKey,
		"kind":    nil,
	}
	endpoint := cli.Client.BuildClientURL("v3", "pushers", "set")
	if _, err := cli.Client.MakeFullRequest(r.Context(), mautrix.FullRequest{
		Method:           http.MethodPost,
		URL:              endpoint,
		RequestJSON:      request,
		SensitiveContent: true,
	}); err != nil {
		return errs.Internal(fmt.Errorf("failed to remove test pusher: %w", err))
	}
	s.pushPrivacyTest.clear()
	return writeJSON(w, s.pushPrivacyTest.snapshot())
}

func (s *Server) receivePushPrivacyTest(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return errs.Validation(map[string]any{"error": "invalid Matrix push payload"})
	}
	notificationRaw, ok := raw["notification"]
	if !ok {
		return errs.Validation(map[string]any{"notification": "is required"})
	}
	var input matrixPushNotificationRequest
	if err := json.Unmarshal(notificationRaw, &input.Notification); err != nil {
		return errs.Validation(map[string]any{"notification": "is invalid"})
	}
	var notificationFields map[string]json.RawMessage
	_ = json.Unmarshal(notificationRaw, &notificationFields)
	fields := make([]string, 0, len(notificationFields))
	for field := range notificationFields {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	rejected := s.pushPrivacyTest.record(input, fields)
	return writeJSON(w, map[string]any{"rejected": rejected})
}

func normalizePushPrivacyTestURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("enter the temporary HTTPS tunnel URL")
	}
	parsed.Path = "/_matrix/push/v1/notify"
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
