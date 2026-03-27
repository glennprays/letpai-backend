package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// AddBillItemRequest represents the request to add a bill item
type AddBillItemRequest struct {
	Description string  `json:"description" validate:"required,min=1,max=200"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Category    *string `json:"category,omitempty"`
}

// BillItemItem represents a bill item in the response
type BillItemItem struct {
	BillItemID  string  `json:"bill_item_id"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Category    *string `json:"category,omitempty"`
}

// AddBillItemResponse represents the response after adding a bill item
type AddBillItemResponse struct {
	Message     string        `json:"message"`
	BillItem    *BillItemItem `json:"bill_item"`
	TotalAmount float64       `json:"total_amount"`
}

// AddBillItemUseCase handles adding a bill item to a session
type AddBillItemUseCase struct {
	sessionRepo  ports.SessionRepository
	billItemRepo ports.BillItemRepository
}

// NewAddBillItemUseCase creates a new add bill item use case
func NewAddBillItemUseCase(
	sessionRepo ports.SessionRepository,
	billItemRepo ports.BillItemRepository,
) *AddBillItemUseCase {
	return &AddBillItemUseCase{
		sessionRepo:  sessionRepo,
		billItemRepo: billItemRepo,
	}
}

// Execute adds a bill item to a session
func (uc *AddBillItemUseCase) Execute(ctx context.Context, userID, sessionID string, req *AddBillItemRequest) (*AddBillItemResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow adding bills to active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot add bills to a completed or cancelled session"))
	}

	sessionUUID := uuid.MustParse(sessionID)

	// Create bill item
	billItem := entity.NewBillItem(sessionUUID, req.Description, req.Amount, req.Category)

	if err := uc.billItemRepo.Create(ctx, billItem); err != nil {
		return nil, err
	}

	// Update session total
	session.AddToTotal(req.Amount)
	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	return &AddBillItemResponse{
		Message: "Bill item added successfully",
		BillItem: &BillItemItem{
			BillItemID:  billItem.BillItemID.String(),
			Description: billItem.Description,
			Amount:      billItem.Amount,
			Category:    billItem.Category,
		},
		TotalAmount: session.TotalAmount,
	}, nil
}
