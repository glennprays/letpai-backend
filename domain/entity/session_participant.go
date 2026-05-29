package entity

import (
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/google/uuid"
)

var ErrInvalidPaymentStatusTransitionError = errors.New("invalid payment status transition")

// SessionParticipant represents a participant in a session
type SessionParticipant struct {
	ParticipantID       uuid.UUID                 `json:"participant_id" db:"participant_id"`
	SessionID           uuid.UUID                 `json:"session_id" db:"session_id"`
	ContactID           *uuid.UUID                `json:"contact_id,omitempty" db:"contact_id"`
	CustomName          string                    `json:"custom_name,omitempty" db:"custom_name"`
	CustomWhatsApp      string                    `json:"custom_whatsapp,omitempty" db:"custom_whatsapp"`
	ShareAmount         float64                   `json:"share_amount" db:"share_amount"`
	PaymentStatus       valueobject.PaymentStatus `json:"payment_status" db:"payment_status"`
	PaymentProofURL     *string                   `json:"payment_proof_url,omitempty" db:"payment_proof_url"`
	RejectionCount      int                       `json:"rejection_count" db:"rejection_count"`
	RejectionReason     *string                   `json:"rejection_reason,omitempty" db:"rejection_reason"`
	NotificationCount   int                       `json:"notification_count" db:"notification_count"`
	LastNotificationAt  *time.Time                `json:"last_notification_at,omitempty" db:"last_notification_at"`
	PaidManually        bool                      `json:"paid_manually" db:"paid_manually"`
	JoinedAt            time.Time                 `json:"joined_at" db:"joined_at"`
	UpdatedAt           time.Time                 `json:"updated_at" db:"updated_at"`

	// Joined fields (populated by FindBySessionIDWithContactInfo, not stored on the row)
	ContactAvatarURL *string `json:"contact_avatar_url,omitempty" db:"contact_avatar_url"`
	ContactName      *string `json:"-" db:"contact_name"`
	ContactWhatsApp  *string `json:"-" db:"contact_whatsapp"`
}

// NewParticipantFromContact creates a new participant from a contact.
//
// name and whatsappNumber are snapshotted from the contact at join
// time and stored on custom_name / custom_whatsapp. Two reasons:
//
//  1. The session_participants table has UNIQUE (session_id,
//     custom_whatsapp). Leaving custom_whatsapp empty for contact-
//     linked participants meant the second contact added to the same
//     session collided on (session_id, "") and the INSERT failed.
//  2. GetSessionDetail's response reads custom_name directly, so an
//     empty value renders a blank row in the UI. Snapshotting gives
//     the FE something to display even if the join to contacts
//     drifts later (contact renamed or soft-deleted).
//
// Matches the pattern NewCustomParticipant already uses below.
func NewParticipantFromContact(sessionID, contactID uuid.UUID, name, whatsappNumber string) *SessionParticipant {
	now := time.Now()
	return &SessionParticipant{
		ParticipantID:  uuid.New(),
		SessionID:      sessionID,
		ContactID:      &contactID,
		CustomName:     name,
		CustomWhatsApp: whatsappNumber,
		ShareAmount:    0,
		PaymentStatus:  valueobject.PaymentStatusPending,
		JoinedAt:       now,
		UpdatedAt:      now,
	}
}

// NewCustomParticipant creates a new custom participant
func NewCustomParticipant(sessionID uuid.UUID, name, whatsappNumber string) *SessionParticipant {
	now := time.Now()
	return &SessionParticipant{
		ParticipantID:  uuid.New(),
		SessionID:      sessionID,
		CustomName:     name,
		CustomWhatsApp: whatsappNumber,
		ShareAmount:    0,
		PaymentStatus:  valueobject.PaymentStatusPending,
		JoinedAt:       now,
		UpdatedAt:      now,
	}
}

// Update updates the participant information
func (p *SessionParticipant) Update(name, whatsappNumber string) {
	if p.ContactID == nil {
		// Only update custom fields for custom participants
		if name != "" {
			p.CustomName = name
		}
		if whatsappNumber != "" {
			p.CustomWhatsApp = whatsappNumber
		}
		p.UpdatedAt = time.Now()
	}
}

// SetShareAmount sets the share amount
func (p *SessionParticipant) SetShareAmount(amount float64) {
	p.ShareAmount = amount
	p.UpdatedAt = time.Now()
}

// UpdatePaymentStatus updates the payment status
func (p *SessionParticipant) UpdatePaymentStatus(newStatus valueobject.PaymentStatus) error {
	if !p.PaymentStatus.CanTransitionTo(newStatus) {
		return ErrInvalidPaymentStatusTransitionError
	}
	p.PaymentStatus = newStatus
	p.UpdatedAt = time.Now()
	return nil
}

// SubmitPayment submits a payment proof (keeps rejection count for resubmissions)
func (p *SessionParticipant) SubmitPayment(proofURL string) error {
	if err := p.UpdatePaymentStatus(valueobject.PaymentStatusSubmitted); err != nil {
		return err
	}
	p.PaymentProofURL = &proofURL
	// Don't reset rejection count on resubmit - it persists across re-submissions
	p.UpdatedAt = time.Now()
	return nil
}

// ApprovePayment approves the payment and resets rejection count
func (p *SessionParticipant) ApprovePayment() error {
	if err := p.UpdatePaymentStatus(valueobject.PaymentStatusPaid); err != nil {
		return err
	}
	p.RejectionCount = 0
	p.RejectionReason = nil
	p.PaidManually = false
	return nil
}

// MarkPaidManually moves the participant directly to Paid without a proof
// upload. Used by the host's "Mark as paid" action. Returns an error if
// the participant is already paid (a no-op the caller can surface as 409).
func (p *SessionParticipant) MarkPaidManually() error {
	if p.PaymentStatus == valueobject.PaymentStatusPaid {
		return ErrInvalidPaymentStatusTransitionError
	}
	// Forced transition: paid_manually bypasses the regular state machine.
	p.PaymentStatus = valueobject.PaymentStatusPaid
	p.PaidManually = true
	p.RejectionCount = 0
	p.RejectionReason = nil
	p.PaymentProofURL = nil
	p.UpdatedAt = time.Now()
	return nil
}

// RejectPayment rejects the payment and increments the rejection count
func (p *SessionParticipant) RejectPayment(reason string) error {
	if err := p.UpdatePaymentStatus(valueobject.PaymentStatusRejected); err != nil {
		return err
	}
	p.RejectionCount++
	p.UpdatedAt = time.Now()
	if reason != "" {
		p.RejectionReason = &reason
	}
	return nil
}

// IsCustom checks if this is a custom participant (not from contacts)
func (p *SessionParticipant) IsCustom() bool {
	return p.ContactID == nil
}

// GetName returns the participant's name — preferring the joined contact
// name when available (set by FindBySessionIDWithContactInfo) to avoid an
// N+1 contact lookup at the call site.
func (p *SessionParticipant) GetName() string {
	if p.ContactName != nil && *p.ContactName != "" {
		return *p.ContactName
	}
	return p.CustomName
}

// GetWhatsAppNumber returns the participant's WhatsApp number — preferring
// the joined contact value when populated.
func (p *SessionParticipant) GetWhatsAppNumber() string {
	if p.ContactWhatsApp != nil && *p.ContactWhatsApp != "" {
		return *p.ContactWhatsApp
	}
	return p.CustomWhatsApp
}

// BumpNotificationCount records that a notification was just sent —
// powers reminder rate-limiting and the "last reminded X minutes ago" UI.
func (p *SessionParticipant) BumpNotificationCount() {
	p.NotificationCount++
	now := time.Now()
	p.LastNotificationAt = &now
	p.UpdatedAt = now
}

// IsPaid checks if the participant has paid
func (p *SessionParticipant) IsPaid() bool {
	return p.PaymentStatus == valueobject.PaymentStatusPaid
}

// HasPendingPayment checks if payment is pending
func (p *SessionParticipant) HasPendingPayment() bool {
	return p.PaymentStatus == valueobject.PaymentStatusPending
}
