package entity

import (
	"time"

	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/google/uuid"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusQueued NotificationStatus = "queued"
	NotificationStatusSent   NotificationStatus = "sent"
	NotificationStatusFailed NotificationStatus = "failed"

	// Outbox lifecycle states (migration 000030).
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSending NotificationStatus = "sending"
	NotificationStatusDead    NotificationStatus = "dead"
)

// rank orders the positive delivery lifecycle. queued is the legacy alias
// for pending. Failure states are not on this ladder (they are handled
// explicitly in NextWebhookStatus).
func (s NotificationStatus) rank() int {
	switch s {
	case NotificationStatusQueued, NotificationStatusPending:
		return 0
	case NotificationStatusSending:
		return 1
	case NotificationStatusSent:
		return 2
	default:
		return -1
	}
}

// NextWebhookStatus encodes the forward-only state machine for gateway
// status webhooks. It returns the status to transition to, or ("", false)
// when the event should be ignored — duplicate, out-of-order, or terminal.
// This is what makes webhook processing idempotent: a re-delivered or stale
// event can never move a row backwards (e.g. a late message.queued can't
// downgrade an already-sent row). The WAGA gateway only emits message.queued
// / message.sent / message.failed; anything else is ignored.
func NextWebhookStatus(current NotificationStatus, event string) (NotificationStatus, bool) {
	switch event {
	case "message.sent":
		if NotificationStatusSent.rank() > current.rank() {
			return NotificationStatusSent, true
		}
		return "", false
	case "message.failed":
		// Apply unless already in a terminal failure state.
		if current == NotificationStatusFailed || current == NotificationStatusDead {
			return "", false
		}
		return NotificationStatusFailed, true
	default:
		// message.queued carries no information beyond what recording the
		// send already told us, so it is a deliberate no-op.
		return "", false
	}
}

// NotificationLog represents a record of a WhatsApp notification sent to a participant
type NotificationLog struct {
	LogID             uuid.UUID                    `json:"log_id" db:"log_id"`
	ParticipantID     uuid.UUID                    `json:"participant_id" db:"participant_id"`
	NotificationType  valueobject.NotificationType `json:"notification_type" db:"notification_type"`
	WhatsAppMessageID *string                      `json:"whatsapp_message_id,omitempty" db:"whatsapp_message_id"`
	MessageContent    string                       `json:"message_content" db:"message_content"`
	SentAt            time.Time                    `json:"sent_at" db:"sent_at"`
	Status            NotificationStatus           `json:"status" db:"status"`
	ErrorMessage      *string                      `json:"error_message,omitempty" db:"error_message"`
	// Phone is the recipient MSISDN snapshot at enqueue time, so the worker
	// sends to the number captured when the notification was created rather
	// than re-resolving a possibly-changed contact.
	Phone *string `json:"phone,omitempty" db:"phone"`
}

// NewNotificationLog creates a new notification log entry
func NewNotificationLog(participantID uuid.UUID, notifType valueobject.NotificationType, messageContent string) *NotificationLog {
	now := time.Now()
	return &NotificationLog{
		LogID:            uuid.New(),
		ParticipantID:    participantID,
		NotificationType: notifType,
		MessageContent:   messageContent,
		SentAt:           now,
		Status:           NotificationStatusQueued,
	}
}

// MarkAsSent marks the notification as sent with the WhatsApp message ID
func (n *NotificationLog) MarkAsSent(whatsappMessageID string) {
	n.WhatsAppMessageID = &whatsappMessageID
	n.Status = NotificationStatusSent
}

// MarkAsFailed marks the notification as failed with an error message
func (n *NotificationLog) MarkAsFailed(errorMessage string) {
	n.ErrorMessage = &errorMessage
	n.Status = NotificationStatusFailed
}

// IsSent checks if the notification was sent successfully
func (n *NotificationLog) IsSent() bool {
	return n.Status == NotificationStatusSent
}

// IsFailed checks if the notification failed
func (n *NotificationLog) IsFailed() bool {
	return n.Status == NotificationStatusFailed
}

// IsQueued checks if the notification is still queued
func (n *NotificationLog) IsQueued() bool {
	return n.Status == NotificationStatusQueued
}
