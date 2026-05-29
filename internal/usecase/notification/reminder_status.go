package notification

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/internal/service"
)

// ReminderStatusResponse mirrors the 429 shape the rate-limit middleware
// returns, but as a 200 read so the FE can pre-disable the Remind button
// on page load when a cooldown is already in effect.
type ReminderStatusResponse struct {
	CanSend           bool   `json:"can_send"`
	RetryAfterSeconds int    `json:"retry_after_seconds"`
	NextAvailableAt   string `json:"next_available_at,omitempty"`
}

// ReminderStatusUseCase reads the per-participant reminder cooldown.
type ReminderStatusUseCase struct {
	rateLimitSvc *service.RateLimitService
}

func NewReminderStatusUseCase(rateLimitSvc *service.RateLimitService) *ReminderStatusUseCase {
	return &ReminderStatusUseCase{rateLimitSvc: rateLimitSvc}
}

// Execute returns the current reminder cooldown for the participant.
// No auth scoping inside the use case because the route caller has
// already authenticated and authorized the request — the rate limiter
// only knows about the participant key.
func (uc *ReminderStatusUseCase) Execute(ctx context.Context, participantID string) (*ReminderStatusResponse, error) {
	canSend, retryAfter, nextResetAt, err := uc.rateLimitSvc.GetReminderStatus(ctx, participantID)
	if err != nil {
		return nil, err
	}
	resp := &ReminderStatusResponse{
		CanSend:           canSend,
		RetryAfterSeconds: int(retryAfter.Seconds()),
	}
	if !canSend && !nextResetAt.IsZero() {
		resp.NextAvailableAt = nextResetAt.UTC().Format(time.RFC3339)
	}
	return resp, nil
}
