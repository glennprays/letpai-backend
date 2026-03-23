package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// NotificationLogRepository defines the interface for notification log data operations
type NotificationLogRepository interface {
	// Create creates a new notification log entry
	Create(ctx context.Context, log *entity.NotificationLog) error

	// FindByID finds a notification log by ID
	FindByID(ctx context.Context, logID string) (*entity.NotificationLog, error)

	// FindByParticipantID finds all notification logs for a participant
	FindByParticipantID(ctx context.Context, participantID string) ([]*entity.NotificationLog, error)

	// FindByParticipantIDWithType finds notification logs for a participant filtered by type
	FindByParticipantIDWithType(ctx context.Context, participantID string, notificationType string) ([]*entity.NotificationLog, error)

	// FindLatestByParticipantID finds the latest notification log for a participant
	FindLatestByParticipantID(ctx context.Context, participantID string) (*entity.NotificationLog, error)

	// FindLatestByParticipantIDWithType finds the latest notification log for a participant filtered by type
	FindLatestByParticipantIDWithType(ctx context.Context, participantID string, notificationType string) (*entity.NotificationLog, error)

	// FindBySessionID finds all notification logs for all participants in a session
	FindBySessionID(ctx context.Context, sessionID string) ([]*entity.NotificationLog, error)

	// FindFailed finds all failed notifications
	FindFailed(ctx context.Context) ([]*entity.NotificationLog, error)

	// UpdateStatus updates the status of a notification log
	UpdateStatus(ctx context.Context, logID string, status string) error

	// UpdateStatusWithMessageID updates the status and WhatsApp message ID of a notification log
	UpdateStatusWithMessageID(ctx context.Context, logID string, status string, whatsappMessageID string) error

	// UpdateStatusWithError updates the status and error message of a notification log
	UpdateStatusWithError(ctx context.Context, logID string, status string, errorMessage string) error

	// CountByParticipantIDAndType counts notification logs for a participant filtered by type
	CountByParticipantIDAndType(ctx context.Context, participantID string, notificationType string) (int, error)

	// CountByParticipantIDAndTypeAfterDate counts notification logs for a participant filtered by type and after a date
	CountByParticipantIDAndTypeAfterDate(ctx context.Context, participantID string, notificationType string, afterDate string) (int, error)

	// Delete deletes a notification log
	Delete(ctx context.Context, logID string) error

	// DeleteByParticipantID deletes all notification logs for a participant
	DeleteByParticipantID(ctx context.Context, participantID string) error
}
