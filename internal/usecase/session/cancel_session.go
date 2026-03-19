package session

import (
	"errors"
	"context"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// CancelSessionResponse represents the response after cancelling a session
type CancelSessionResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// CancelSessionUseCase handles cancelling a session
type CancelSessionUseCase struct {
	sessionRepo ports.SessionRepository
}

// NewCancelSessionUseCase creates a new cancel session use case
func NewCancelSessionUseCase(
	sessionRepo ports.SessionRepository,
) *CancelSessionUseCase {
	return &CancelSessionUseCase{
		sessionRepo: sessionRepo,
	}
}

// Execute cancels a session
func (uc *CancelSessionUseCase) Execute(ctx context.Context, userID, sessionID string) (*CancelSessionResponse, error) {
	// Fetch existing session
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow cancelling active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot cancel a completed or cancelled session"))
	}

	// Cancel session
	if err := session.Cancel(); err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return &CancelSessionResponse{
		Message: "Session cancelled successfully",
		Status:  session.Status.String(),
	}, nil
}
