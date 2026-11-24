package service

import (
	"crypto/rand"
	"strings"
	"testing"
)

func newTestKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return key
}

func TestSecretBoxRoundtrip(t *testing.T) {
	sb, err := NewSecretBox(newTestKey(t))
	if err != nil {
		t.Fatalf("NewSecretBox: %v", err)
	}
	plain := "eyJhbGciOiJIUzI1NiJ9.payload.sig"
	sealed, err := sb.Seal(plain)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if !strings.HasPrefix(sealed, "v1:") {
		t.Fatalf("sealed output not versioned: %q", sealed)
	}
	if strings.Contains(sealed, plain) {
		t.Fatal("plaintext leaked into ciphertext")
	}
	got, err := sb.Open(sealed)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got != plain {
		t.Fatalf("roundtrip mismatch: got %q want %q", got, plain)
	}
}

func TestSecretBoxTamperedCiphertext(t *testing.T) {
	sb, _ := NewSecretBox(newTestKey(t))
	sealed, err := sb.Seal("secret")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	parts := strings.SplitN(sealed, ":", 3)
	// Flip one char of the ciphertext body.
	bad := parts[0] + ":" + parts[1] + ":" + flipChar(parts[2])
	if _, err := sb.Open(bad); err == nil {
		t.Fatal("Open accepted tampered ciphertext")
	}
}

func TestSecretBoxWrongKey(t *testing.T) {
	sb1, _ := NewSecretBox(newTestKey(t))
	sb2, _ := NewSecretBox(newTestKey(t))
	sealed, err := sb1.Seal("secret")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := sb2.Open(sealed); err == nil {
		t.Fatal("Open succeeded with wrong key")
	}
}

func TestSecretBoxShortKeyRejected(t *testing.T) {
	if _, err := NewSecretBox(make([]byte, 16)); err == nil {
		t.Fatal("16-byte key accepted")
	}
	if _, err := NewSecretBox(nil); err == nil {
		t.Fatal("nil key accepted")
	}
}

func TestSecretBoxRejectsUnknownFormat(t *testing.T) {
	sb, _ := NewSecretBox(newTestKey(t))
	for _, in := range []string{"", "v1:", "plaintext", "v2:a:b"} {
		if _, err := sb.Open(in); err == nil {
			t.Fatalf("Open accepted %q", in)
		}
	}
}

func flipChar(s string) string {
	b := []byte(s)
	if b[0] == 'A' {
		b[0] = 'B'
	} else {
		b[0] = 'A'
	}
	return string(b)
}
