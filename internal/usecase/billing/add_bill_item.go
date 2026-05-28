package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// AddBillItemRequest represents the request to add a bill item.
//
// ParticipantIDs is optional. An empty/missing list means "split equally
// across every participant in the session"; a non-empty list scopes the
// bill to those participants only.
type AddBillItemRequest struct {
	Description    string   `json:"description" validate:"required,min=1,max=200"`
	Amount         float64  `json:"amount" validate:"required,gt=0"`
	Category       *string  `json:"category,omitempty"`
	ParticipantIDs []string `json:"participant_ids,omitempty" validate:"omitempty,dive,uuid"`
}

// BillItemItem represents a bill item in the response
type BillItemItem struct {
	BillItemID     string   `json:"bill_item_id"`
	Description    string   `json:"description"`
	Amount         float64  `json:"amount"`
	Category       *string  `json:"category,omitempty"`
	ParticipantIDs []string `json:"participant_ids"`
}

// AddBillItemResponse represents the response after adding a bill item
type AddBillItemResponse struct {
	Message     string        `json:"message"`
	BillItem    *BillItemItem `json:"bill_item"`
	TotalAmount float64       `json:"total_amount"`
}

// AddBillItemUseCase handles adding a bill item to a session
type AddBillItemUseCase struct {
	sessionRepo     ports.SessionRepository
	billItemRepo    ports.BillItemRepository
	participantRepo ports.ParticipantRepository
}

// NewAddBillItemUseCase creates a new add bill item use case
func NewAddBillItemUseCase(
	sessionRepo ports.SessionRepository,
	billItemRepo ports.BillItemRepository,
	participantRepo ports.ParticipantRepository,
) *AddBillItemUseCase {
	return &AddBillItemUseCase{
		sessionRepo:     sessionRepo,
		billItemRepo:    billItemRepo,
		participantRepo: participantRepo,
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

	// Resolve and validate participant_ids (if provided). They must all
	// belong to this session so we don't accidentally bill someone else.
	participantUUIDs, err := uc.resolveParticipants(ctx, sessionID, req.ParticipantIDs)
	if err != nil {
		return nil, err
	}

	// Create bill item
	billItem := entity.NewBillItem(sessionUUID, req.Description, req.Amount, req.Category, participantUUIDs)

	if err := uc.billItemRepo.Create(ctx, billItem); err != nil {
		return nil, err
	}

	// Update session total
	session.AddToTotal(req.Amount)
	if err := uc.sessionRepo.Update(ctx, session); err != nil {
		return nil, err
	}

	pidStrs := make([]string, 0, len(billItem.ParticipantIDs))
	for _, id := range billItem.ParticipantIDs {
		pidStrs = append(pidStrs, id.String())
	}

	return &AddBillItemResponse{
		Message: "Bill item added successfully",
		BillItem: &BillItemItem{
			BillItemID:     billItem.BillItemID.String(),
			Description:    billItem.Description,
			Amount:         billItem.Amount,
			Category:       billItem.Category,
			ParticipantIDs: pidStrs,
		},
		TotalAmount: session.TotalAmount,
	}, nil
}

// resolveParticipants parses incoming participant_id strings into UUIDs
// and verifies that each one belongs to the given session. Returns nil
// when the input is empty (legacy "everyone" semantics).
func (uc *AddBillItemUseCase) resolveParticipants(ctx context.Context, sessionID string, ids []string) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	participants, err := uc.participantRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	valid := make(map[string]bool, len(participants))
	for _, p := range participants {
		valid[p.ParticipantID.String()] = true
	}

	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if !valid[id] {
			return nil, domain.NewError(domain.ErrBadRequest, errors.New("participant_id does not belong to this session"))
		}
		u, err := uuid.Parse(id)
		if err != nil {
			return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid participant_id"))
		}
		out = append(out, u)
	}
	return out, nil
}
