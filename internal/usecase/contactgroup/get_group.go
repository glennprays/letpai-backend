package contactgroup

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// GetGroupItem represents a single contact group (swagger ContactGroup schema)
type GetGroupItem struct {
	GroupID   string `json:"group_id"`
	GroupName string `json:"group_name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
	CreatedAt string `json:"created_at"`
}

// GetGroupUseCase handles retrieving a single contact group
type GetGroupUseCase struct {
	groupRepo ports.ContactGroupRepository
}

// NewGetGroupUseCase creates a new get group use case
func NewGetGroupUseCase(
	groupRepo ports.ContactGroupRepository,
) *GetGroupUseCase {
	return &GetGroupUseCase{
		groupRepo: groupRepo,
	}
}

// Execute retrieves a single contact group owned by the user
func (uc *GetGroupUseCase) Execute(ctx context.Context, groupID, userID string) (*GetGroupItem, error) {
	group, err := uc.groupRepo.FindByID(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}

	return &GetGroupItem{
		GroupID:   group.GroupID.String(),
		GroupName: group.Name,
		Color:     group.Color,
		SortOrder: group.SortOrder,
		CreatedAt: group.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
