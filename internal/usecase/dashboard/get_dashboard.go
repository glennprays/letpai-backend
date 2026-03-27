
package dashboard

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// DashboardResponse represents dashboard data
type DashboardResponse struct {
	ActiveSessions   int     `json:"active_sessions"`
	CompletedSessions int     `json:"completed_sessions"`
	PendingPayments   int     `json:"pending_payments"`
	TotalPending    float64 `json:"total_pending"`
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
	active := "active"
	completed := "completed"

	
	activeResult, err := uc.sessionRepo.FindAll(ctx, userID, &ports.SessionFilterOptions{
		Status: stringPtr("active"),
		Limit: 10000,
	})
	if err != nil {
		return nil, err
	}

	// Count completed sessions
	completedResult, err := uc.sessionRepo.FindAll(ctx, userID, &ports.SessionFilterOptions{
		Status: stringPtr("completed"),
		Limit: 10000,
	})
	if err != nil {
		return nil, err
	}


	pendingCount, err := uc.participantRepo.CountBySessionIDAndStatus(ctx, userID, "pending")
	if err != nil {
		return nil, err
	}

	return &DashboardResponse{
		ActiveSessions:   len(activeResult.Sessions),
		CompletedSessions: len(completedResult.Sessions),
		PendingPayments: pendingCount,
		TotalPending: 0,
	}, nil
	}
