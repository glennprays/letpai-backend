package billing

import (
	"context"
	"errors"
	"math"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// CalculateSplitsResponse represents the response after calculating splits
type CalculateSplitsResponse struct {
	Message          string              `json:"message"`
	TotalAmount      float64             `json:"total_amount"`
	ParticipantCount int                 `json:"participant_count"`
	SharePerPerson   float64             `json:"share_per_person"`
	Participants     []*ParticipantSplit `json:"participants"`
}

// ParticipantSplit represents a participant's split
type ParticipantSplit struct {
	ParticipantID string  `json:"participant_id"`
	ShareAmount   float64 `json:"share_amount"`
	PaymentStatus string  `json:"payment_status"`
}

// CalculateSplitsUseCase handles calculating equal splits for a session
type CalculateSplitsUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
	billItemRepo    ports.BillItemRepository
}

// NewCalculateSplitsUseCase creates a new calculate splits use case
func NewCalculateSplitsUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	billItemRepo ports.BillItemRepository,
) *CalculateSplitsUseCase {
	return &CalculateSplitsUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		billItemRepo:    billItemRepo,
	}
}

// Execute calculates equal splits for all participants.
//
// CONCURRENCY: this method is NOT transactional. A concurrent AddBillItem /
// DeleteBillItem / AddParticipants / RemoveParticipant between the read
// (participants + bill total) and the write (BulkUpdateShareAmounts) will
// leave the split stale until the next CalculateSplits call. The full fix
// requires a unit-of-work pattern with `SELECT ... FOR UPDATE` on the
// session row across the read and write; see backlog item B1.
// Mitigation today: hosts call CalculateSplits after they're done editing,
// not concurrently with edits.
func (uc *CalculateSplitsUseCase) Execute(ctx context.Context, userID, sessionID string) (*CalculateSplitsResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow calculating splits for active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot calculate splits for a completed or cancelled session"))
	}

	// Fetch all participants
	participants, err := uc.participantRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Get total from bill items (sum of all bills)
	billTotal, err := uc.billItemRepo.SumBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Calculate share per person
	participantCount := len(participants)
	if participantCount == 0 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("no participants in session"))
	}

	sharePerPerson := math.Round(billTotal/float64(participantCount)*100) / 100

	// Update all participants' share amounts
	updates := make(map[string]float64)
	participantSplits := make([]*ParticipantSplit, 0, participantCount)

	for _, p := range participants {
		p.SetShareAmount(sharePerPerson)
		updates[p.ParticipantID.String()] = sharePerPerson

		participantSplits = append(participantSplits, &ParticipantSplit{
			ParticipantID: p.ParticipantID.String(),
			ShareAmount:   sharePerPerson,
			PaymentStatus: p.PaymentStatus.String(),
		})
	}

	// Bulk update share amounts
	if err := uc.participantRepo.BulkUpdateShareAmounts(ctx, updates); err != nil {
		return nil, err
	}

	return &CalculateSplitsResponse{
		Message:          "Splits calculated successfully",
		TotalAmount:      billTotal,
		ParticipantCount: participantCount,
		SharePerPerson:   sharePerPerson,
		Participants:     participantSplits,
	}, nil
}
