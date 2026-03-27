package contactgroup

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateGroupRequest represents the request to update a contact group
type UpdateGroupRequest struct {
	Name      *string `json:"name" validate:"omitempty,min=1,max=50"`
	Color     *string `json:"color" validate:"omitempty,min=1,max=20"`
	SortOrder *int    `json:"sort_order"`
}

// UpdateGroupResponse represents the response after updating a contact group
type UpdateGroupResponse struct {
	GroupID   string `json:"group_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// UpdateGroupUseCase handles updating a contact group
type UpdateGroupUseCase struct {
	groupRepo ports.ContactGroupRepository
}

// NewUpdateGroupUseCase creates a new update group use case
func NewUpdateGroupUseCase(
	groupRepo ports.ContactGroupRepository,
) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{
		groupRepo: groupRepo,
	}
}

// Execute updates a contact group
func (uc *UpdateGroupUseCase) Execute(ctx context.Context, userID, groupID string, req *UpdateGroupRequest) (*UpdateGroupResponse, error) {
	// Fetch existing group
	group, err := uc.groupRepo.FindByID(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}

	// Check if name is being updated and if it conflicts
	if req.Name != nil && *req.Name != group.Name {
		exists, err := uc.groupRepo.ExistsByName(ctx, userID, *req.Name, &groupID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, domain.NewError(domain.ErrConflict, errors.New("group with this name already exists"))
		}
		group.Update(*req.Name, group.Color)
	}

	// Update color if provided
	if req.Color != nil {
		group.Update(group.Name, *req.Color)
	}

	// Update sort order if provided
	if req.SortOrder != nil {
		group.UpdateSortOrder(*req.SortOrder)
	}

	if err := uc.groupRepo.Update(ctx, group); err != nil {
		return nil, err
	}

	return &UpdateGroupResponse{
		GroupID:   group.GroupID.String(),
		Name:      group.Name,
		Color:     group.Color,
		SortOrder: group.SortOrder,
	}, nil
}
