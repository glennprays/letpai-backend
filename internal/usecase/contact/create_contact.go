package contact

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// CreateContactRequest represents the request to create a contact
type CreateContactRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=100"`
	WhatsAppNumber string  `json:"whatsapp_number" validate:"required"`
	GroupID        *string `json:"group_id,omitempty"`
}

// CreateContactResponse represents the response after creating a contact
type CreateContactResponse struct {
	ContactID      string  `json:"contact_id"`
	Name           string  `json:"name"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	GroupID        *string `json:"group_id,omitempty"`
	GroupName      *string `json:"group_name,omitempty"`
	GroupColor     *string `json:"group_color,omitempty"`
	IsFavorite     bool    `json:"is_favorite"`
}

// CreateContactUseCase handles creating a new contact
type CreateContactUseCase struct {
	contactRepo ports.ContactRepository
	groupRepo   ports.ContactGroupRepository
}

// NewCreateContactUseCase creates a new create contact use case
func NewCreateContactUseCase(
	contactRepo ports.ContactRepository,
	groupRepo ports.ContactGroupRepository,
) *CreateContactUseCase {
	return &CreateContactUseCase{
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
	}
}

// Execute creates a new contact
func (uc *CreateContactUseCase) Execute(ctx context.Context, userID string, req *CreateContactRequest) (*CreateContactResponse, error) {
	// Parse userID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid user ID"))
	}

	// Validate group if provided
	var groupID *uuid.UUID
	if req.GroupID != nil {
		groupUUID, err := uuid.Parse(*req.GroupID)
		if err != nil {
			return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid group ID"))
		}

		// Verify group exists and belongs to user
		_, err = uc.groupRepo.FindByID(ctx, *req.GroupID, userID)
		if err != nil {
			return nil, err
		}
		groupID = &groupUUID
	}

	// Check if contact with same WhatsApp number exists
	exists, err := uc.contactRepo.ExistsByWhatsApp(ctx, userID, req.WhatsAppNumber, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.NewError(domain.ErrConflict, errors.New("contact with this WhatsApp number already exists"))
	}

	// Create contact
	contact := entity.NewContact(userUUID, req.Name, req.WhatsAppNumber)
	contact.AssignToGroup(groupID)

	if err := uc.contactRepo.Create(ctx, contact); err != nil {
		return nil, err
	}

	return &CreateContactResponse{
		ContactID:      contact.ContactID.String(),
		Name:           contact.Name,
		WhatsAppNumber: contact.WhatsAppNumber,
		GroupID:        req.GroupID,
		IsFavorite:     contact.IsFavorite,
	}, nil
}
