package contact

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// GetContactByIDResponse represents the response for getting a contact by ID
type GetContactByIDResponse struct {
	ContactID      string  `json:"contact_id"`
	Name           string  `json:"name"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	GroupID        *string `json:"group_id,omitempty"`
	GroupName      *string `json:"group_name,omitempty"`
	GroupColor     *string `json:"group_color,omitempty"`
	IsFavorite     bool    `json:"is_favorite"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// GetContactByIDUseCase handles retrieving a contact by ID
type GetContactByIDUseCase struct {
	contactRepo ports.ContactRepository
}

// NewGetContactByIDUseCase creates a new get contact by ID use case
func NewGetContactByIDUseCase(
	contactRepo ports.ContactRepository,
) *GetContactByIDUseCase {
	return &GetContactByIDUseCase{
		contactRepo: contactRepo,
	}
}

// Execute retrieves a contact by ID
func (uc *GetContactByIDUseCase) Execute(ctx context.Context, userID, contactID string) (*GetContactByIDResponse, error) {
	contact, err := uc.contactRepo.FindByID(ctx, contactID, userID)
	if err != nil {
		return nil, err
	}

	var groupID *string
	if contact.GroupID != nil {
		gid := contact.GroupID.String()
		groupID = &gid
	}

	return &GetContactByIDResponse{
		ContactID:      contact.ContactID.String(),
		Name:           contact.Name,
		WhatsAppNumber: contact.WhatsAppNumber,
		GroupID:        groupID,
		GroupName:      contact.GroupName,
		GroupColor:     contact.GroupColor,
		IsFavorite:     contact.IsFavorite,
		AvatarURL:      contact.AvatarURL,
		Notes:          contact.Notes,
		CreatedAt:      contact.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      contact.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
