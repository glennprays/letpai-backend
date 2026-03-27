package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateBillItemRequest represents a request to update a bill item
type UpdateBillItemRequest struct {
	Description string  `json:"description" validate:"omitempty,min=1,max=500"`
	Amount      float64 `json:"amount" validate:"omitempty,gt=0"`
	Category    string  `json:"category,omitempty"`
}

// UpdateBillItemResponse represents response after updating a bill item
type UpdateBillItemResponse struct {
	BillItemID  string  `json:"bill_item_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    *string `json:"category,omitempty"`
	UpdatedAt   string  `json:"updated_at"`
}

// UpdateBillItemUseCase handles updating a bill item
type UpdateBillItemUseCase struct {
	billItemRepo ports.BillItemRepository
	sessionRepo  ports.SessionRepository
}

// NewUpdateBillItemUseCase creates a new update bill item use case
func NewUpdateBillItemUseCase(
	billItemRepo ports.BillItemRepository,
	sessionRepo ports.SessionRepository,
) *UpdateBillItemUseCase {
	return &UpdateBillItemUseCase{
		billItemRepo: billItemRepo,
		sessionRepo:  sessionRepo,
	}
}

// Execute updates a bill item and recalculates session total
func (uc *UpdateBillItemUseCase) Execute(ctx context.Context, userID, sessionID, billItemID string, req *UpdateBillItemRequest) (*UpdateBillItemResponse, error) {
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
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("bill item does not belong to this session"))
	}

	// Update fields if provided
	if req.Description != "" {
		billItem.Description = req.Description
	}
	if req.Amount > 0 {
		billItem.Amount = req.Amount
	}
	if req.Category != "" {
		billItem.Category = &req.Category
	}

	if err := uc.billItemRepo.Update(ctx, billItem); err != nil {
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

	return &UpdateBillItemResponse{
		BillItemID:  billItem.BillItemID.String(),
		Description: billItem.Description,
		Amount:      billItem.Amount,
		Category:    billItem.Category,
		UpdatedAt:   billItem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
