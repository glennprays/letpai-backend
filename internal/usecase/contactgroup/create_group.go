package contactgroup

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// CreateGroupRequest represents the request to create a contact group
type CreateGroupRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=50"`
	Color     string `json:"color" validate:"required,min=1,max=20"`
	SortOrder int    `json:"sort_order"`
}

// CreateGroupResponse represents the response after creating a contact group
type CreateGroupResponse struct {
	GroupID   uuid.UUID `json:"group_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
}

// CreateGroupUseCase handles creating a new contact group
type CreateGroupUseCase struct {
	groupRepo ports.ContactGroupRepository
}

// NewCreateGroupUseCase creates a new create group use case
func NewCreateGroupUseCase(
	groupRepo ports.ContactGroupRepository,
) *CreateGroupUseCase {
	return &CreateGroupUseCase{
		groupRepo: groupRepo,
	}
}

// Execute creates a new contact group
func (uc *CreateGroupUseCase) Execute(ctx context.Context, userID string, req *CreateGroupRequest) (*CreateGroupResponse, error) {
	// Parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid user ID"))
	}

	// Check if group with same name exists
	exists, err := uc.groupRepo.ExistsByName(ctx, userID, req.Name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.NewError(domain.ErrConflict, errors.New("group with this name already exists"))
	}

	// Create group
	group := entity.NewContactGroup(userUUID, req.Name, req.Color, req.SortOrder)

	if err := uc.groupRepo.Create(ctx, group); err != nil {
		return nil, err
	}

	return &CreateGroupResponse{
		GroupID:   group.GroupID,
		Name:      group.Name,
		Color:     group.Color,
		SortOrder: group.SortOrder,
	}, nil
}
