package participant

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/usecase/billing"
)

// RemoveParticipantResponse represents the response after removing a participant
type RemoveParticipantResponse struct {
	Message string `json:"message"`
}

// RemoveParticipantUseCase handles removing a participant from a session.
//
// Removes are unconditional with respect to payment status — the host
// is free to drop a paid participant. The FE wraps the destructive
// action in a strongly-worded confirmation, so the backend doesn't
// need a second gate. The rationale: refunds happen outside the app,
// and the host knows whether they've been settled before clicking.
//
// After deletion the use case re-runs CalculateSplits so the
// remaining participants' share_amount values aren't left stale.
type RemoveParticipantUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
	calcSplits      *billing.CalculateSplitsUseCase
}

// NewRemoveParticipantUseCase creates a new remove participant use case
func NewRemoveParticipantUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	calcSplits *billing.CalculateSplitsUseCase,
) *RemoveParticipantUseCase {
	return &RemoveParticipantUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		calcSplits:      calcSplits,
	}
}

// Execute removes a participant from a session
func (uc *RemoveParticipantUseCase) Execute(ctx context.Context, userID, sessionID, participantID string) (*RemoveParticipantResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow removing participants from active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot remove participants from a completed or cancelled session"))
	}

	if err := uc.participantRepo.Delete(ctx, participantID); err != nil {
		return nil, err
	}

	// Best-effort rebalance: errors here shouldn't fail the delete
	// itself (the row is already gone). CalculateSplits also refuses
	// if there are 0 remaining participants, which we treat as
	// "nothing to rebalance" rather than an error.
	_, _ = uc.calcSplits.Execute(ctx, userID, sessionID)

	return &RemoveParticipantResponse{
		Message: "Participant removed successfully",
	}, nil
}
