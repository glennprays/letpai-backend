package contactgroup

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// GetGroupsResponse represents the response for listing contact groups
type GetGroupsResponse struct {
	Groups []*GroupItem `json:"groups"`
}

// GroupItem represents a contact group item
type GroupItem struct {
	GroupID   string `json:"group_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// GetGroupsUseCase handles retrieving all contact groups
type GetGroupsUseCase struct {
	groupRepo ports.ContactGroupRepository
}

// NewGetGroupsUseCase creates a new get groups use case
func NewGetGroupsUseCase(
	groupRepo ports.ContactGroupRepository,
) *GetGroupsUseCase {
	return &GetGroupsUseCase{
		groupRepo: groupRepo,
	}
}

// Execute retrieves all contact groups for a user
func (uc *GetGroupsUseCase) Execute(ctx context.Context, userID string) (*GetGroupsResponse, error) {
	groups, err := uc.groupRepo.FindAll(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]*GroupItem, 0, len(groups))
	for _, g := range groups {
		items = append(items, &GroupItem{
			GroupID:   g.GroupID.String(),
			Name:      g.Name,
			Color:     g.Color,
			SortOrder: g.SortOrder,
		})
	}

	return &GetGroupsResponse{
		Groups: items,
	}, nil
}
