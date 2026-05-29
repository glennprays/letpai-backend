package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

var ErrBillItemNotBelongToSession = errors.New("bill item does not belong to this session")

// DeleteBillItemResponse represents response after deleting a bill item
type DeleteBillItemResponse struct {
	Message  string  `json:"message"`
	NewTotal float64 `json:"new_total"`
}

// DeleteBillItemUseCase handles deleting a bill item.
//
// After a successful delete the use case re-runs CalculateSplits so
// the remaining bill amounts are redistributed across participants.
// Without this, every participant's share_amount would silently
// reflect a session total that no longer matches reality.
type DeleteBillItemUseCase struct {
	billItemRepo ports.BillItemRepository
	sessionRepo  ports.SessionRepository
	calcSplits   *CalculateSplitsUseCase
}

// NewDeleteBillItemUseCase creates a new delete bill item use case
func NewDeleteBillItemUseCase(
	billItemRepo ports.BillItemRepository,
	sessionRepo ports.SessionRepository,
	calcSplits *CalculateSplitsUseCase,
) *DeleteBillItemUseCase {
	return &DeleteBillItemUseCase{
		billItemRepo: billItemRepo,
		sessionRepo:  sessionRepo,
		calcSplits:   calcSplits,
	}
}

// Execute deletes a bill item and recalculates session total
func (uc *DeleteBillItemUseCase) Execute(ctx context.Context, userID, sessionID, billItemID string) (*DeleteBillItemResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Get bill item
	billItem, err := uc.billItemRepo.FindByID(ctx, billItemID)
	if err != nil {
		return nil, err
	}

	// Verify bill item belongs to the session
	if billItem.SessionID.String() != sessionID {
		return nil, domain.NewError(domain.ErrBadRequest, ErrBillItemNotBelongToSession)
	}

	// Delete bill item
	if err := uc.billItemRepo.Delete(ctx, billItemID); err != nil {
		return nil, err
	}

	// Recalculate session total
	items, err := uc.billItemRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	newTotal := 0.0
	for _, item := range items {
		newTotal += item.Amount
	}

	session.SetTotal(newTotal)

	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	// Best-effort rebalance. CalculateSplits errors out when there
	// are 0 participants -- treat that as "nothing to rebalance"
	// rather than failing the delete that's already committed.
	_, _ = uc.calcSplits.Execute(ctx, userID, sessionID)

	return &DeleteBillItemResponse{
		Message:  "Bill item deleted successfully",
		NewTotal: newTotal,
	}, nil
}
