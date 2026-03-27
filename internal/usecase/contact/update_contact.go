package contact

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// UpdateContactRequest represents the request to update a contact
type UpdateContactRequest struct {
	Name           *string `json:"name" validate:"omitempty,min=1,max=100"`
	WhatsAppNumber *string `json:"whatsapp_number" validate:"omitempty"`
	GroupID        *string `json:"group_id,omitempty"`
	IsFavorite     *bool   `json:"is_favorite,omitempty"`
}

// UpdateContactResponse represents the response after updating a contact
type UpdateContactResponse struct {
	ContactID      string  `json:"contact_id"`
	Name           string  `json:"name"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	GroupID        *string `json:"group_id,omitempty"`
	GroupName      *string `json:"group_name,omitempty"`
	GroupColor     *string `json:"group_color,omitempty"`
	IsFavorite     bool    `json:"is_favorite"`
	UpdatedAt      string  `json:"updated_at"`
}

// UpdateContactUseCase handles updating a contact
type UpdateContactUseCase struct {
	contactRepo ports.ContactRepository
	groupRepo   ports.ContactGroupRepository
}

// NewUpdateContactUseCase creates a new update contact use case
func NewUpdateContactUseCase(
	contactRepo ports.ContactRepository,
	groupRepo ports.ContactGroupRepository,
) *UpdateContactUseCase {
	return &UpdateContactUseCase{
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
	}
}

// Execute updates a contact
func (uc *UpdateContactUseCase) Execute(ctx context.Context, userID, contactID string, req *UpdateContactRequest) (*UpdateContactResponse, error) {
	// Fetch existing contact
	contact, err := uc.contactRepo.FindByID(ctx, contactID, userID)
	if err != nil {
		return nil, err
	}

	// Validate group if provided
	if req.GroupID != nil {
		if *req.GroupID == "" {
			// Remove from group
			contact.AssignToGroup(nil)
		} else {
			groupUUID, err := uuid.Parse(*req.GroupID)
			if err != nil {
				return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid group ID"))
			}

			// Verify group exists and belongs to user
			_, err = uc.groupRepo.FindByID(ctx, *req.GroupID, userID)
			if err != nil {
				return nil, err
			}
			contact.AssignToGroup(&groupUUID)
		}
	}

	// Update name if provided
	if req.Name != nil {
		contact.Update(*req.Name, "")
	}

	// Update WhatsApp number if provided
	if req.WhatsAppNumber != nil {
		// Check if WhatsApp number conflicts with another contact
		exists, err := uc.contactRepo.ExistsByWhatsApp(ctx, userID, *req.WhatsAppNumber, &contactID)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, domain.NewError(domain.ErrConflict, errors.New("contact with this WhatsApp number already exists"))
		}
		contact.Update("", *req.WhatsAppNumber)
	}

	// Update favorite status if provided
	if req.IsFavorite != nil {
		contact.SetFavorite(*req.IsFavorite)
	}

	if err := uc.contactRepo.Update(ctx, contact); err != nil {
		return nil, err
	}

	var groupID *string
	if contact.GroupID != nil {
		gid := contact.GroupID.String()
		groupID = &gid
	}

	return &UpdateContactResponse{
		ContactID:      contact.ContactID.String(),
		Name:           contact.Name,
		WhatsAppNumber: contact.WhatsAppNumber,
		GroupID:        groupID,
		GroupName:      contact.GroupName,
		GroupColor:     contact.GroupColor,
		IsFavorite:     contact.IsFavorite,
		UpdatedAt:      contact.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
