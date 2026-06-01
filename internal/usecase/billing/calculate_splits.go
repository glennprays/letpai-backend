package billing

import (
	"context"
	"errors"
	"math"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// FeeBreakdownItem represents a participant's fee breakdown
type FeeBreakdownItem struct {
	ParticipantID      string  `json:"participant_id"`
	ItemsTotal         float64 `json:"items_total"`
	ServiceChargeShare float64 `json:"service_charge_share"`
	TaxShare           float64 `json:"tax_share"`
	Total              float64 `json:"total"`
}

// CalculateSplitsResponse represents the response after calculating splits
type CalculateSplitsResponse struct {
	Message          string              `json:"message"`
	TotalAmount      float64             `json:"total_amount"`
	ParticipantCount int                 `json:"participant_count"`
	SharePerPerson   float64             `json:"share_per_person"`
	Participants     []*ParticipantSplit `json:"participants"`
	FeeBreakdown     []*FeeBreakdownItem `json:"fee_breakdown,omitempty"`
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
// When the session has fee_config (service_charge_percentage / tax_percentage
// > 0), the calculation is fee-aware:
//   - Each participant's items_total = sum of their bill item shares
//   - service_charge_share = sum of shares on items where includes_service_charge=true
//     × session.service_charge_percentage / 100
//   - tax_share = sum of shares on items where includes_tax=true
//     × session.tax_percentage / 100
//   - share_amount = items_total + service_charge_share + tax_share
//
// When no fee_config exists (or both are zero), the calculation is the
// simple per-bill split as before.
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

	hasFees := session.ServiceChargePercentage > 0 || session.TaxPercentage > 0

	// Per-participant accumulators
	type accum struct {
		itemsTotal         float64
		serviceChargeBase  float64 // sum of shares on items where includes_service_charge=true
		taxBase            float64 // sum of shares on items where includes_tax=true
	}
	accumulators := make(map[string]*accum, len(orderedIDs))
	for _, id := range orderedIDs {
		accumulators[id] = &accum{}
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
				shareIDs = append(shareIDs, orderedIDs...)
			}
		}

		// Floor the per-person share to whole units (IDR is whole-currency).
		// Push the rounding remainder onto the last participant so the bill
		// total reconciles exactly.
		perPerson := math.Floor(bill.Amount / float64(len(shareIDs)))
		distributed := perPerson * float64(len(shareIDs))
		remainder := bill.Amount - distributed

		for i, id := range shareIDs {
			share := perPerson
			if i == len(shareIDs)-1 {
				share += remainder
			}
			accumulators[id].itemsTotal += share
			if hasFees {
				if bill.IncludesServiceCharge {
					accumulators[id].serviceChargeBase += share
				}
				if bill.IncludesTax {
					accumulators[id].taxBase += share
				}
			}
		}

		billTotal += bill.Amount
	}

	updates := make(map[string]float64, len(orderedIDs))
	participantSplits := make([]*ParticipantSplit, 0, len(orderedIDs))
	feeBreakdowns := make([]*FeeBreakdownItem, 0, len(orderedIDs))

	for _, p := range participants {
		idStr := p.ParticipantID.String()
		a := accumulators[idStr]

		var scShare, taxShare, total float64
		if hasFees {
			scShare = math.Round(a.serviceChargeBase * session.ServiceChargePercentage / 100)
			taxShare = math.Round(a.taxBase * session.TaxPercentage / 100)
			total = math.Round(a.itemsTotal) + scShare + taxShare
		} else {
			total = math.Round(a.itemsTotal)
		}

		p.SetShareAmount(total)
		updates[idStr] = total
		participantSplits = append(participantSplits, &ParticipantSplit{
			ParticipantID: idStr,
			ShareAmount:   total,
			PaymentStatus: p.PaymentStatus.String(),
		})
		feeBreakdowns = append(feeBreakdowns, &FeeBreakdownItem{
			ParticipantID:      idStr,
			ItemsTotal:         math.Round(a.itemsTotal),
			ServiceChargeShare: scShare,
			TaxShare:           taxShare,
			Total:              total,
		})
	}

	if err := uc.participantRepo.BulkUpdateShareAmounts(ctx, updates); err != nil {
		return nil, err
	}

	// Update session total_amount to include fees so the frontend
	// shows a consistent grand total everywhere (dashboard, session
	// detail, participant pages).
	if hasFees {
		totalSc := 0.0
		totalTax := 0.0
		for _, fb := range feeBreakdowns {
			totalSc += fb.ServiceChargeShare
			totalTax += fb.TaxShare
		}
		grandTotal := billTotal + totalSc + totalTax
		session.SetTotal(grandTotal)
		if err := uc.sessionRepo.Update(ctx, session); err != nil {
			return nil, err
		}
	}

	avg := 0.0
	if len(orderedIDs) > 0 {
		avg = math.Round((billTotal/float64(len(orderedIDs)))*100) / 100
	}

	resp := &CalculateSplitsResponse{
		Message:          "Splits calculated successfully",
		TotalAmount:      billTotal,
		ParticipantCount: len(orderedIDs),
		SharePerPerson:   avg,
		Participants:     participantSplits,
	}
	if hasFees {
		resp.FeeBreakdown = feeBreakdowns
	}
	return resp, nil
}
