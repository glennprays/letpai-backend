package contact

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// DeleteContactResponse represents the response after deleting a contact
type DeleteContactResponse struct {
	Message string `json:"message"`
}

// DeleteContactUseCase handles deleting a contact
type DeleteContactUseCase struct {
	contactRepo ports.ContactRepository
}

// NewDeleteContactUseCase creates a new delete contact use case
func NewDeleteContactUseCase(
	contactRepo ports.ContactRepository,
) *DeleteContactUseCase {
	return &DeleteContactUseCase{
		contactRepo: contactRepo,
	}
}

// Execute deletes a contact
func (uc *DeleteContactUseCase) Execute(ctx context.Context, userID, contactID string) (*DeleteContactResponse, error) {
	if err := uc.contactRepo.Delete(ctx, contactID, userID); err != nil {
		return nil, err
	}

	return &DeleteContactResponse{
		Message: "Contact deleted successfully",
	}, nil
}
