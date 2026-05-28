package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// UpdateBillItemRequest represents a request to update a bill item.
//
// ParticipantIDs is a nullable pointer:
//   - nil (absent from JSON)  → keep existing assignments untouched
//   - non-nil, empty slice    → clear assignments (apply to everyone)
//   - non-nil, populated      → replace assignments with the given list
type UpdateBillItemRequest struct {
	Description    string    `json:"description" validate:"omitempty,min=1,max=500"`
	Amount         float64   `json:"amount" validate:"omitempty,gt=0"`
	Category       string    `json:"category,omitempty"`
	ParticipantIDs *[]string `json:"participant_ids,omitempty" validate:"omitempty,dive,uuid"`
}

// UpdateBillItemResponse represents response after updating a bill item
type UpdateBillItemResponse struct {
	BillItemID     string   `json:"bill_item_id"`
	Description    string   `json:"description"`
	Amount         float64  `json:"amount"`
	Category       *string  `json:"category,omitempty"`
	ParticipantIDs []string `json:"participant_ids"`
	UpdatedAt      string   `json:"updated_at"`
}

// UpdateBillItemUseCase handles updating a bill item
type UpdateBillItemUseCase struct {
	billItemRepo    ports.BillItemRepository
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
}

// NewUpdateBillItemUseCase creates a new update bill item use case
func NewUpdateBillItemUseCase(
	billItemRepo ports.BillItemRepository,
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
) *UpdateBillItemUseCase {
	return &UpdateBillItemUseCase{
		billItemRepo:    billItemRepo,
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
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

	// ParticipantIDs is intentionally a pointer: nil means "untouched",
	// non-nil empty means "reset to everyone".
	if req.ParticipantIDs != nil {
		resolved, err := uc.resolveParticipants(ctx, sessionID, *req.ParticipantIDs)
		if err != nil {
			return nil, err
		}
		billItem.ParticipantIDs = resolved
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

	pidStrs := make([]string, 0, len(billItem.ParticipantIDs))
	for _, id := range billItem.ParticipantIDs {
		pidStrs = append(pidStrs, id.String())
	}

	return &UpdateBillItemResponse{
		BillItemID:     billItem.BillItemID.String(),
		Description:    billItem.Description,
		Amount:         billItem.Amount,
		Category:       billItem.Category,
		ParticipantIDs: pidStrs,
		UpdatedAt:      billItem.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

// resolveParticipants validates the incoming list of participant_ids
// against the session's actual participants. Empty input is allowed and
// represents the legacy "everyone" assignment.
func (uc *UpdateBillItemUseCase) resolveParticipants(ctx context.Context, sessionID string, ids []string) ([]uuid.UUID, error) {
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
