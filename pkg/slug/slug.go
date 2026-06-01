// Package slug generates and validates the short URL-safe handles
// used in user-visible URLs (e.g. /payment/<slug>). UUIDs stay as
// internal PKs; the slug is a UNIQUE secondary key.
package slug

import (
	"crypto/rand"
	"errors"
	"regexp"
)

const (
	// Alphabet is the URL-safe nanoid alphabet: 64 characters.
	// Pairs with Length = 10 for a 64^10 ≈ 1.15e18 keyspace — at
	// 1M rows the birthday-paradox collision probability is ~4e-7.
	// We still retry on UNIQUE violation in the repo so even a
	// rare hit degrades to a second pick rather than a 500.
	Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	Length   = 10
)

// pattern matches the canonical slug shape (exactly 10 alphabet
// chars). Used by the handler boundary to decide "this looks like a
// slug, look it up by slug" vs "this looks like a UUID, look it up
// by id". The strict 10-char check disambiguates against the 36-char
// UUID form.
var pattern = regexp.MustCompile(`^[A-Za-z0-9_\-]{10}$`)

// Is reports whether s has the shape of a generated slug. NOT a
// proof of existence — the repo lookup is the source of truth.
func Is(s string) bool {
	return pattern.MatchString(s)
}

// New returns a fresh random 10-char slug from crypto/rand. Caller
// is responsible for the UNIQUE-violation retry loop when inserting.
func New() (string, error) {
	buf := make([]byte, Length)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.New("slug: rand read failed")
	}
	out := make([]byte, Length)
	for i, b := range buf {
		out[i] = Alphabet[int(b)%len(Alphabet)]
	}
	return string(out), nil
}
