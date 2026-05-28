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

// Execute calculates per-bill splits, accumulates per-participant shares,
// and writes them back to the session participants.
//
// SEMANTICS: each bill is divided among its assigned participants (empty
// assignment ⇒ everyone in the session). Per-participant shares are
// rounded to 2 decimals; any rounding remainder is added to the last
// participant in the bill's iteration order so the bill total matches
// the original amount exactly.
//
// CONCURRENCY: NOT transactional. Concurrent AddBillItem / DeleteBillItem
// / AddParticipants / RemoveParticipant calls between the read and the
// write can leave the split stale until the next call. Hosts should
// call this once their edits are done.
func (uc *CalculateSplitsUseCase) Execute(ctx context.Context, userID, sessionID string) (*CalculateSplitsResponse, error) {
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot calculate splits for a completed or cancelled session"))
	}

	participants, err := uc.participantRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(participants) == 0 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("no participants in session"))
	}

	orderedIDs := make([]string, 0, len(participants))
	participantByID := make(map[string]bool, len(participants))
	for _, p := range participants {
		id := p.ParticipantID.String()
		orderedIDs = append(orderedIDs, id)
		participantByID[id] = true
	}

	bills, err := uc.billItemRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	accumulated := make(map[string]float64, len(orderedIDs))
	for _, id := range orderedIDs {
		accumulated[id] = 0
	}

	billTotal := 0.0
	for _, bill := range bills {
		shareIDs := make([]string, 0)
		if len(bill.ParticipantIDs) == 0 {
			shareIDs = append(shareIDs, orderedIDs...)
		} else {
			for _, pid := range bill.ParticipantIDs {
				idStr := pid.String()
				if participantByID[idStr] {
					shareIDs = append(shareIDs, idStr)
				}
			}
			if len(shareIDs) == 0 {
				// All originally-assigned participants have been removed;
				// fall back to "everyone" so the bill still contributes.
				shareIDs = append(shareIDs, orderedIDs...)
			}
		}

		perPerson := math.Round((bill.Amount/float64(len(shareIDs)))*100) / 100
		distributed := perPerson * float64(len(shareIDs))
		remainder := math.Round((bill.Amount-distributed)*100) / 100

		for _, id := range shareIDs {
			accumulated[id] += perPerson
		}
		if remainder != 0 {
			accumulated[shareIDs[len(shareIDs)-1]] += remainder
		}

		billTotal += bill.Amount
	}

	updates := make(map[string]float64, len(orderedIDs))
	participantSplits := make([]*ParticipantSplit, 0, len(orderedIDs))
	for _, p := range participants {
		idStr := p.ParticipantID.String()
		amount := math.Round(accumulated[idStr]*100) / 100
		p.SetShareAmount(amount)
		updates[idStr] = amount
		participantSplits = append(participantSplits, &ParticipantSplit{
			ParticipantID: idStr,
			ShareAmount:   amount,
			PaymentStatus: p.PaymentStatus.String(),
		})
	}

	if err := uc.participantRepo.BulkUpdateShareAmounts(ctx, updates); err != nil {
		return nil, err
	}

	avg := 0.0
	if len(orderedIDs) > 0 {
		avg = math.Round((billTotal/float64(len(orderedIDs)))*100) / 100
	}

	return &CalculateSplitsResponse{
		Message:          "Splits calculated successfully",
		TotalAmount:      billTotal,
		ParticipantCount: len(orderedIDs),
		SharePerPerson:   avg,
		Participants:     participantSplits,
	}, nil
}
