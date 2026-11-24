package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// The gateway's json QR payload is {"qr_code": "...", "timeout": 60} —
// no expires_in. The SDK's ExpiresIn stays 0 and the old math produced
// expires == now, i.e. an instantly expired QR.
func TestGetQRCodeDefaultsTTLOnMissingExpiresIn(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/status":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"authenticated":false}`))
		case "/login/qr_code/json":
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"qr_code":"data:image/png;base64,AAAA","timeout":60}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	svc := NewWhatsAppService(srv.URL, "test-jwt")
	qr, expires, err := svc.GetQRCode(context.Background())
	if err != nil {
		t.Fatalf("GetQRCode: %v", err)
	}
	if qr == "" {
		t.Fatal("empty qr")
	}
	if gotAuth != "Bearer test-jwt" {
		t.Fatalf("missing bearer header: %q", gotAuth)
	}
	// Old code: expires ≈ now (ExpiresIn=0). New code: now+60s ±1s slack.
	left := time.Until(expires)
	if left < 58*time.Second || left > 61*time.Second {
		t.Fatalf("TTL wrong: got %v, want ~60s", left)
	}
}
