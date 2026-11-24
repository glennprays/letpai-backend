package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// SecretBox encrypts short secrets (gateway JWTs) at rest with
// AES-256-GCM. Ciphertexts are versioned ("v1:<nonce>:<ct>") so the
// format can evolve without a flag-day: Open rejects unknown versions
// explicitly instead of misparsing them as base64.
//
// The key never leaves memory unencrypted; a lost key makes every
// stored token unreadable (recovery = re-register the phone).
type SecretBox struct {
	aead cipher.AEAD
}

const secretBoxVersion = "v1"

// NewSecretBox builds an AES-256-GCM box. The key must be exactly 32
// bytes: callers pass base64-decoded WHATSAPP_TOKEN_ENC_KEY material.
func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secretbox: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	return &SecretBox{aead: aead}, nil
}

// Seal encrypts plaintext into "v1:<b64(nonce)>:<b64(ct+tag)>".
func (sb *SecretBox) Seal(plaintext string) (string, error) {
	nonce := make([]byte, sb.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secretbox: nonce: %w", err)
	}
	ct := sb.aead.Seal(nil, nonce, []byte(plaintext), nil)
	return secretBoxVersion + ":" +
		base64.RawStdEncoding.EncodeToString(nonce) + ":" +
		base64.RawStdEncoding.EncodeToString(ct), nil
}

// Open decrypts a Seal-produced string. Wrong key, tampered
// ciphertext, or unknown version → error (GCM authenticates first).
func (sb *SecretBox) Open(ciphertext string) (string, error) {
	parts := strings.SplitN(ciphertext, ":", 3)
	if len(parts) != 3 || parts[0] != secretBoxVersion {
		return "", errors.New("secretbox: unrecognized ciphertext format")
	}
	nonce, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("secretbox: nonce: %w", err)
	}
	ct, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("secretbox: ciphertext: %w", err)
	}
	pt, err := sb.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("secretbox: decrypt: %w", err)
	}
	return string(pt), nil
}
