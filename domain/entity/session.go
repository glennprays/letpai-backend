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
	SessionID        uuid.UUID                      `json:"session_id" db:"session_id"`
	UserID           uuid.UUID                      `json:"user_id" db:"user_id"`
	Title            string                         `json:"title" db:"title"`
	Description      string                         `json:"description" db:"description"`
	Status           valueobject.SessionStatus      `json:"status" db:"status"`
	TotalAmount      float64                        `json:"total_amount" db:"total_amount"`
	Currency         string                         `json:"currency" db:"currency"`
	SessionDate      *time.Time                     `json:"session_date,omitempty" db:"session_date"`
	CreatedAt        time.Time                      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at" db:"updated_at"`
	DeletedAt        *time.Time                     `json:"deleted_at,omitempty" db:"deleted_at"`

	// Computed fields (not in database)
	ParticipantCount *int `json:"participant_count,omitempty" db:"participant_count"`
	BillItemCount    *int `json:"bill_item_count,omitempty" db:"bill_item_count"`
	PaidCount        *int `json:"paid_count,omitempty" db:"paid_count"`
}

// NewSession creates a new session
func NewSession(userID uuid.UUID, title, description, currency string, sessionDate *time.Time) *Session {
	now := time.Now()
	return &Session{
		SessionID:    uuid.New(),
		UserID:       userID,
		Title:        title,
		Description:  description,
		Status:       valueobject.SessionStatusActive,
		TotalAmount:  0,
		Currency:     currency,
		SessionDate:  sessionDate,
		CreatedAt:    now,
		UpdatedAt:    now,
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
