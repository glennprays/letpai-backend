package contact

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// BulkOperationsRequest represents the request for bulk operations
type BulkOperationsRequest struct {
	Operation  string   `json:"operation" validate:"required,oneof=add_to_group delete"`
	ContactIDs []string `json:"contact_ids" validate:"required,min=1"`
	GroupID    string   `json:"group_id,omitempty"`
}

// BulkOperationsResponse represents the response after bulk operations
type BulkOperationsResponse struct {
	Message  string `json:"message"`
	Affected int    `json:"affected"`
}

// BulkOperationsUseCase handles bulk contact operations
type BulkOperationsUseCase struct {
	contactRepo ports.ContactRepository
	groupRepo   ports.ContactGroupRepository
}

// NewBulkOperationsUseCase creates a new bulk operations use case
func NewBulkOperationsUseCase(
	contactRepo ports.ContactRepository,
	groupRepo ports.ContactGroupRepository,
) *BulkOperationsUseCase {
	return &BulkOperationsUseCase{
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
	}
}

// Execute performs bulk operations on contacts
func (uc *BulkOperationsUseCase) Execute(ctx context.Context, userID string, req *BulkOperationsRequest) (*BulkOperationsResponse, error) {
	if len(req.ContactIDs) == 0 {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("contact_ids cannot be empty"))
	}

	switch req.Operation {
	case "add_to_group":
		return uc.addToGroup(ctx, userID, req)
	case "delete":
		return uc.bulkDelete(ctx, userID, req)
	default:
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid operation"))
	}
}

func (uc *BulkOperationsUseCase) addToGroup(ctx context.Context, userID string, req *BulkOperationsRequest) (*BulkOperationsResponse, error) {
	if req.GroupID == "" {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("group_id is required for add_to_group operation"))
	}

	// Validate group ID
	_, err := uuid.Parse(req.GroupID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid group ID"))
	}

	// Verify group exists and belongs to user
	_, err = uc.groupRepo.FindByID(ctx, req.GroupID, userID)
	if err != nil {
		return nil, err
	}

	if err := uc.contactRepo.BulkAddToGroup(ctx, req.ContactIDs, req.GroupID, userID); err != nil {
		return nil, err
	}

	return &BulkOperationsResponse{
		Message:  "Contacts added to group successfully",
		Affected: len(req.ContactIDs),
	}, nil
}

func (uc *BulkOperationsUseCase) bulkDelete(ctx context.Context, userID string, req *BulkOperationsRequest) (*BulkOperationsResponse, error) {
	if err := uc.contactRepo.BulkDelete(ctx, req.ContactIDs, userID); err != nil {
		return nil, err
	}

	return &BulkOperationsResponse{
		Message:  "Contacts deleted successfully",
		Affected: len(req.ContactIDs),
	}, nil
}
