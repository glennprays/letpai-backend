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
)

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
