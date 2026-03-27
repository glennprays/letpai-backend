package dashboard

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// DashboardResponse represents dashboard data
type DashboardResponse struct {
	ActiveSessions    int     `json:"active_sessions"`
	CompletedSessions int     `json:"completed_sessions"`
	PendingPayments   int     `json:"pending_payments"`
	TotalPending      float64 `json:"total_pending"`
}

// GetDashboardUseCase retrieves dashboard statistics
type GetDashboardUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
}

// NewGetDashboardUseCase creates a new dashboard use case
func NewGetDashboardUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
) *GetDashboardUseCase {
	return &GetDashboardUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
	}
}

// Execute retrieves dashboard statistics for a user
func (uc *GetDashboardUseCase) Execute(ctx context.Context, userID string) (*DashboardResponse, error) {
	// Count active sessions
	activeStatus := valueobject.SessionStatusActive

	activeResult, err := uc.sessionRepo.FindAll(ctx, userID, &ports.SessionFilterOptions{
		Status: &activeStatus,
		Limit:  10000,
	})
	if err != nil {
		return nil, err
	}

	// Count completed sessions
	completedStatus := valueobject.SessionStatusCompleted
	completedResult, err := uc.sessionRepo.FindAll(ctx, userID, &ports.SessionFilterOptions{
		Status: &completedStatus,
		Limit:  10000,
	})
	if err != nil {
		return nil, err
	}

	pendingCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, userID, "pending")
	if err != nil {
		return nil, err
	}

	return &DashboardResponse{
		ActiveSessions:    len(activeResult.Sessions),
		CompletedSessions: len(completedResult.Sessions),
		PendingPayments:   pendingCount,
		TotalPending:      0,
	}, nil
}
