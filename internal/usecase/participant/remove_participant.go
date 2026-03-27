package participant

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// RemoveParticipantResponse represents the response after removing a participant
type RemoveParticipantResponse struct {
	Message string `json:"message"`
}

// RemoveParticipantUseCase handles removing a participant from a session
type RemoveParticipantUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
}

// NewRemoveParticipantUseCase creates a new remove participant use case
func NewRemoveParticipantUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
) *RemoveParticipantUseCase {
	return &RemoveParticipantUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
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

	return &RemoveParticipantResponse{
		Message: "Participant removed successfully",
	}, nil
}
