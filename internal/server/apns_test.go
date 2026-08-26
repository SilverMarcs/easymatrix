package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/batuhan/easymatrix/internal/config"
)

func TestNewAPNSProviderAcceptsInlineKey(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate test key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal test key: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	escapedKey := strings.ReplaceAll(string(pemKey), "\n", `\n`)

	provider, err := newAPNSProvider(config.Config{
		APNSKey:         escapedKey,
		APNSKeyID:       "KEYID",
		APNSTeamID:      "TEAMID",
		APNSTopic:       "com.example.relay",
		APNSEnvironment: "production",
	})
	if err != nil {
		t.Fatalf("newAPNSProvider returned error: %v", err)
	}
	if provider.keyID != "KEYID" || provider.baseURL != "https://api.push.apple.com" {
		t.Fatalf("unexpected provider configuration: %#v", provider)
	}
}
