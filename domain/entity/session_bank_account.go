package entity

import (
	"time"

	"github.com/google/uuid"
)

// SessionBankAccount is one transfer destination the host has
// associated with a session. The participant sees these on their
// public payment page so they know where to pay.
//
// At least one of BankName / AccountNumber / AccountHolder must be
// non-empty (mirrors the DB CHECK constraint in migration 000023).
type SessionBankAccount struct {
	AccountID     uuid.UUID  `json:"account_id" db:"account_id"`
	SessionID     uuid.UUID  `json:"session_id" db:"session_id"`
	Ordinal       int        `json:"ordinal" db:"ordinal"`
	BankName      *string    `json:"bank_name,omitempty" db:"bank_name"`
	AccountNumber *string    `json:"account_number,omitempty" db:"account_number"`
	AccountHolder *string    `json:"account_holder,omitempty" db:"account_holder"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt     *time.Time `json:"-" db:"deleted_at"`
}

// IsEmpty reports whether every meaningful field is nil or only
// whitespace. The BulkReplace use case uses this to drop entries the
// host typed and then cleared before saving.
func (a *SessionBankAccount) IsEmpty() bool {
	return strOrNil(a.BankName) == nil &&
		strOrNil(a.AccountNumber) == nil &&
		strOrNil(a.AccountHolder) == nil
}

// Normalize trims whitespace and converts blank strings to nil so the
// CHECK constraint round-trips cleanly.
func (a *SessionBankAccount) Normalize() {
	a.BankName = strOrNil(a.BankName)
	a.AccountNumber = strOrNil(a.AccountNumber)
	a.AccountHolder = strOrNil(a.AccountHolder)
}

func strOrNil(p *string) *string {
	if p == nil {
		return nil
	}
	trimmed := trimSpace(*p)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func trimSpace(s string) string {
	// Avoid importing "strings" in the entity layer — keep the entity
	// dependency surface minimal. Manual whitespace trim covers the
	// ASCII cases users actually type.
	start, end := 0, len(s)
	for start < end {
		c := s[start]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			start++
			continue
		}
		break
	}
	for end > start {
		c := s[end-1]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			end--
			continue
		}
		break
	}
	return s[start:end]
}
