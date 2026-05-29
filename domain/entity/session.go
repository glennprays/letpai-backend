package entity

import (
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/google/uuid"
)

var ErrInvalidStatusTransitionError = errors.New("invalid status transition")

// Session represents a bill splitting session
type Session struct {
	SessionID   uuid.UUID                 `json:"session_id" db:"session_id"`
	UserID      uuid.UUID                 `json:"user_id" db:"user_id"`
	Title       string                    `json:"title" db:"title"`
	Description string                    `json:"description" db:"description"`
	Status      valueobject.SessionStatus `json:"status" db:"status"`
	TotalAmount float64                   `json:"total_amount" db:"total_amount"`
	Currency    string                    `json:"currency" db:"currency"`
	SessionDate *time.Time                `json:"session_date,omitempty" db:"session_date"`

	// Bank transfer destination the host wants participants to pay to.
	// Shown on the participant's public payment page. All three are
	// optional so the host can fill in just what's relevant (e.g.
	// e-wallet account holder + number without a bank_name).
	BankName          *string `json:"bank_name,omitempty" db:"bank_name"`
	BankAccountNumber *string `json:"bank_account_number,omitempty" db:"bank_account_number"`
	BankAccountHolder *string `json:"bank_account_holder,omitempty" db:"bank_account_holder"`

	// LastNotifiedAt is the timestamp of the most recent successful
	// `POST /sessions/:id/send-notifications` call. IsDirtyForNotify
	// compares it against UpdatedAt (which child-touch triggers bump
	// on every participant / bill mutation) to decide whether the
	// host is allowed to fire a fresh batch.
	LastNotifiedAt *time.Time `json:"last_notified_at,omitempty" db:"last_notified_at"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Computed fields (not in database)
	ParticipantCount *int `json:"participant_count,omitempty" db:"participant_count"`
	BillItemCount    *int `json:"bill_item_count,omitempty" db:"bill_item_count"`
	PaidCount        *int `json:"paid_count,omitempty" db:"paid_count"`
}

// SetBankInfo updates the bank transfer destination. Pass nil pointers
// to leave a field untouched, or a pointer to an empty string to clear
// it. Empty strings normalize to nil so partial entries round-trip
// cleanly through the public payment page.
func (s *Session) SetBankInfo(name, number, holder *string) {
	if name != nil {
		s.BankName = normalizeBankField(name)
	}
	if number != nil {
		s.BankAccountNumber = normalizeBankField(number)
	}
	if holder != nil {
		s.BankAccountHolder = normalizeBankField(holder)
	}
	s.UpdatedAt = time.Now()
}

func normalizeBankField(p *string) *string {
	if p == nil {
		return nil
	}
	if *p == "" {
		return nil
	}
	return p
}

// NewSession creates a new session
func NewSession(userID uuid.UUID, title, description, currency string, sessionDate *time.Time) *Session {
	now := time.Now()
	return &Session{
		SessionID:   uuid.New(),
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      valueobject.SessionStatusActive,
		TotalAmount: 0,
		Currency:    currency,
		SessionDate: sessionDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Update updates the session information
func (s *Session) Update(title, description, currency string, sessionDate *time.Time) {
	if title != "" {
		s.Title = title
	}
	if description != "" {
		s.Description = description
	}
	if currency != "" {
		s.Currency = currency
	}
	s.SessionDate = sessionDate
	s.UpdatedAt = time.Now()
}

// UpdateStatus updates the session status
func (s *Session) UpdateStatus(newStatus valueobject.SessionStatus) error {
	if !s.Status.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransitionError
	}
	s.Status = newStatus
	s.UpdatedAt = time.Now()
	return nil
}

// AddToTotal adds an amount to the total
func (s *Session) AddToTotal(amount float64) {
	s.TotalAmount += amount
	s.UpdatedAt = time.Now()
}

// SetTotal sets the total amount
func (s *Session) SetTotal(amount float64) {
	s.TotalAmount = amount
	s.UpdatedAt = time.Now()
}

// Complete marks the session as completed
func (s *Session) Complete() error {
	return s.UpdateStatus(valueobject.SessionStatusCompleted)
}

// Cancel marks the session as cancelled
func (s *Session) Cancel() error {
	return s.UpdateStatus(valueobject.SessionStatusCancelled)
}

// IsActive checks if the session is active
func (s *Session) IsActive() bool {
	return s.Status == valueobject.SessionStatusActive
}

// IsDirtyForNotify reports whether the host is allowed to fire a fresh
// `POST /sessions/:id/send-notifications` batch. The check is pure
// over the loaded entity — DB triggers guarantee UpdatedAt advances on
// every child mutation, and MarkNotified updates LastNotifiedAt
// without touching UpdatedAt (intentionally — see migration 000021),
// so a successful send naturally collapses the predicate to false.
//
// hasUnpaid is required so that "mark everyone paid" doesn't re-open
// the gate (an updated_at bump from a participant write would, by
// itself, satisfy the timestamp predicate). When every participant is
// settled there's no one left to notify, so dirty is false regardless.
func (s *Session) IsDirtyForNotify(hasUnpaid bool) bool {
	if !hasUnpaid {
		return false
	}
	if s.LastNotifiedAt == nil {
		return true
	}
	return s.UpdatedAt.After(*s.LastNotifiedAt)
}

// MarkNotified updates the in-memory timestamp. The repo call that
// persists this must NOT touch UpdatedAt or the predicate flips back
// to dirty immediately.
func (s *Session) MarkNotified(at time.Time) {
	s.LastNotifiedAt = &at
}

// IsTerminal checks if the session is in a terminal state
func (s *Session) IsTerminal() bool {
	return s.Status.IsTerminal()
}

// SoftDelete marks the session as deleted
func (s *Session) SoftDelete() {
	now := time.Now()
	s.DeletedAt = &now
	s.UpdatedAt = now
}

// IsDeleted checks if the session is deleted
func (s *Session) IsDeleted() bool {
	return s.DeletedAt != nil
}
