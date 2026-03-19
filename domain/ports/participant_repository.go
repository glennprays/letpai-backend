package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// ParticipantRepository defines the interface for participant data operations
type ParticipantRepository interface {
	// Create creates a new participant
	Create(ctx context.Context, participant *entity.SessionParticipant) error

	// FindByID finds a participant by ID
	FindByID(ctx context.Context, participantID string) (*entity.SessionParticipant, error)

	// FindBySessionID finds all participants for a session
	FindBySessionID(ctx context.Context, sessionID string) ([]*entity.SessionParticipant, error)

	// FindBySessionIDWithContactInfo finds all participants for a session with contact info
	FindBySessionIDWithContactInfo(ctx context.Context, sessionID string) ([]*entity.SessionParticipant, error)

	// Update updates a participant
	Update(ctx context.Context, participant *entity.SessionParticipant) error

	// Delete deletes a participant
	Delete(ctx context.Context, participantID string) error

	// DeleteBySessionID deletes all participants for a session
	DeleteBySessionID(ctx context.Context, sessionID string) error

	// UpdateShareAmount updates the share amount for a participant
	UpdateShareAmount(ctx context.Context, participantID string, amount float64) error

	// UpdatePaymentStatus updates the payment status for a participant
	UpdatePaymentStatus(ctx context.Context, participantID string, status string) error

	// BulkUpdateShareAmounts updates share amounts for multiple participants
	BulkUpdateShareAmounts(ctx context.Context, updates map[string]float64) error

	// CountBySessionIDAndStatus counts participants by payment status for a session
	CountBySessionIDAndStatus(ctx context.Context, sessionID string, status string) (int, error)
}
