package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// ParticipantRepository defines the interface for participant data operations.
//
// SECURITY: methods on this interface DO NOT verify session/user ownership.
// Callers reached via an authenticated HTTP route MUST first verify the
// session belongs to the user (via SessionRepository.FindByID(ctx, sessionID,
// userID)) before touching participants. Public endpoints (e.g. /payments/{id}
// /submit) intentionally skip that check; they identify the participant by
// path UUID alone.
type ParticipantRepository interface {
	// Create creates a new participant
	Create(ctx context.Context, participant *entity.SessionParticipant) error

	// FindByID finds a participant by ID. See SECURITY note on the interface.
	FindByID(ctx context.Context, participantID string) (*entity.SessionParticipant, error)

	// FindBySessionID finds all participants for a session
	FindBySessionID(ctx context.Context, sessionID string) ([]*entity.SessionParticipant, error)

	// FindBySessionIDWithContactInfo finds all participants for a session with contact info
	FindBySessionIDWithContactInfo(ctx context.Context, sessionID string) ([]*entity.SessionParticipant, error)

	// Update updates a participant. See SECURITY note on the interface.
	Update(ctx context.Context, participant *entity.SessionParticipant) error

	// Delete deletes a participant
	Delete(ctx context.Context, participantID string) error

	// DeleteBySessionID deletes all participants for a session
	DeleteBySessionID(ctx context.Context, sessionID string) error

	// UpdateShareAmount updates the share amount for a participant
	UpdateShareAmount(ctx context.Context, participantID string, amount float64) error

	// UpdatePaymentStatus updates the payment status for a participant
	UpdatePaymentStatus(ctx context.Context, participantID string, status string) error

	// MarkSubmittedWithProof atomically transitions a participant from
	// 'pending' to 'submitted' with the supplied proof URL. Returns
	// (claimed=true) only if the row was actually updated — concurrent
	// submitters lose to the first one, preventing double-uploads from
	// both persisting.
	MarkSubmittedWithProof(ctx context.Context, participantID, proofURL string) (claimed bool, err error)

	// BulkUpdateShareAmounts updates share amounts for multiple participants
	BulkUpdateShareAmounts(ctx context.Context, updates map[string]float64) error

	// CountBySessionIDAndStatus counts participants by payment status for a session
	CountBySessionIDAndStatus(ctx context.Context, sessionID string, status string) (int, error)

	// CountByUserIDAndStatus tallies participants across every active
	// session owned by the host. Used by the dashboard tile.
	CountByUserIDAndStatus(ctx context.Context, userID string, status string) (int, error)

	// SumPendingShareByUserID sums share_amount across every unpaid
	// participant in every active session owned by the host (pending +
	// submitted + rejected statuses). Drives the dashboard's "total
	// pending" amount.
	SumPendingShareByUserID(ctx context.Context, userID string) (float64, error)
}
