package service

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestParseJWTExpiry(t *testing.T) {
	claims, _ := json.Marshal(map[string]any{"exp": time.Now().Add(365 * 24 * time.Hour).Unix(), "sub": "6281555487630"})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	tok := "eyJhbGciOiJIUzI1NiJ9." + payload + ".sig"
	got, ok := ParseJWTExpiry(tok)
	if !ok {
		t.Fatal("failed to parse well-formed JWT")
	}
	if time.Until(got) < 364*24*time.Hour {
		t.Fatalf("expiry too close: %v", got)
	}
	if _, ok := ParseJWTExpiry("not-a-jwt"); ok {
		t.Fatal("parsed garbage")
	}
	if _, ok := ParseJWTExpiry("a." + base64.RawURLEncoding.EncodeToString([]byte(`{"norg":1}`)) + ".c"); ok {
		t.Fatal("parsed JWT without exp")
	}
}
