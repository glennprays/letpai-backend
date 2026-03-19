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
	ParticipantID    uuid.UUID                   `json:"participant_id" db:"participant_id"`
	SessionID        uuid.UUID                   `json:"session_id" db:"session_id"`
	ContactID        *uuid.UUID                  `json:"contact_id,omitempty" db:"contact_id"`
	CustomName       string                      `json:"custom_name,omitempty" db:"custom_name"`
	CustomWhatsApp   string                      `json:"custom_whatsapp,omitempty" db:"custom_whatsapp"`
	ShareAmount      float64                     `json:"share_amount" db:"share_amount"`
	PaymentStatus    valueobject.PaymentStatus   `json:"payment_status" db:"payment_status"`
	PaymentProofURL  *string                     `json:"payment_proof_url,omitempty" db:"payment_proof_url"`
	JoinedAt         time.Time                   `json:"joined_at" db:"joined_at"`
	UpdatedAt        time.Time                   `json:"updated_at" db:"updated_at"`

	// Joined fields (not in database)
	ContactAvatarURL *string `json:"contact_avatar_url,omitempty" db:"contact_avatar_url"`
}

// NewParticipantFromContact creates a new participant from a contact
func NewParticipantFromContact(sessionID, contactID uuid.UUID) *SessionParticipant {
	now := time.Now()
	return &SessionParticipant{
		ParticipantID: uuid.New(),
		SessionID:     sessionID,
		ContactID:     &contactID,
		ShareAmount:   0,
		PaymentStatus: valueobject.PaymentStatusPending,
		JoinedAt:      now,
		UpdatedAt:     now,
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

// SubmitPayment submits a payment proof
func (p *SessionParticipant) SubmitPayment(proofURL string) error {
	if err := p.UpdatePaymentStatus(valueobject.PaymentStatusSubmitted); err != nil {
		return err
	}
	p.PaymentProofURL = &proofURL
	return nil
}

// ApprovePayment approves the payment
func (p *SessionParticipant) ApprovePayment() error {
	return p.UpdatePaymentStatus(valueobject.PaymentStatusPaid)
}

// RejectPayment rejects the payment
func (p *SessionParticipant) RejectPayment() error {
	return p.UpdatePaymentStatus(valueobject.PaymentStatusRejected)
}

// IsCustom checks if this is a custom participant (not from contacts)
func (p *SessionParticipant) IsCustom() bool {
	return p.ContactID == nil
}

// GetName returns the participant's name
func (p *SessionParticipant) GetName() string {
	if p.ContactID != nil {
		return "" // Will be fetched from contact
	}
	return p.CustomName
}

// GetWhatsAppNumber returns the participant's WhatsApp number
func (p *SessionParticipant) GetWhatsAppNumber() string {
	if p.ContactID != nil {
		return "" // Will be fetched from contact
	}
	return p.CustomWhatsApp
}

// IsPaid checks if the participant has paid
func (p *SessionParticipant) IsPaid() bool {
	return p.PaymentStatus == valueobject.PaymentStatusPaid
}

// HasPendingPayment checks if payment is pending
func (p *SessionParticipant) HasPendingPayment() bool {
	return p.PaymentStatus == valueobject.PaymentStatusPending
}
