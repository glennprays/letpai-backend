package contactgroup

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// DeleteGroupResponse represents the response after deleting a contact group
type DeleteGroupResponse struct {
	Message string `json:"message"`
}

// DeleteGroupUseCase handles deleting a contact group
type DeleteGroupUseCase struct {
	groupRepo ports.ContactGroupRepository
}

// NewDeleteGroupUseCase creates a new delete group use case
func NewDeleteGroupUseCase(
	groupRepo ports.ContactGroupRepository,
) *DeleteGroupUseCase {
	return &DeleteGroupUseCase{
		groupRepo: groupRepo,
	}
}

// Execute deletes a contact group
func (uc *DeleteGroupUseCase) Execute(ctx context.Context, userID, groupID string) (*DeleteGroupResponse, error) {
	if err := uc.groupRepo.Delete(ctx, groupID, userID); err != nil {
		return nil, err
	}

	return &DeleteGroupResponse{
		Message: "Group deleted successfully",
	}, nil
}
