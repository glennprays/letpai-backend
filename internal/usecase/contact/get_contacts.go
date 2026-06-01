package contact

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// ContactItem represents a contact item in the list
type ContactItem struct {
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
}

// GetContactsRequest represents the request to get contacts
type GetContactsRequest struct {
	GroupID    *string
	IsFavorite *bool
	Search     *string
	SortBy     string
	SortOrder  string
	Page       int
	Limit      int
}

// GetContactsResponse represents the response for listing contacts
type GetContactsResponse struct {
	Contacts []*ContactItem `json:"contacts"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	Limit    int            `json:"limit"`
}

// GetContactsUseCase handles retrieving contacts
type GetContactsUseCase struct {
	contactRepo ports.ContactRepository
}

// NewGetContactsUseCase creates a new get contacts use case
func NewGetContactsUseCase(
	contactRepo ports.ContactRepository,
) *GetContactsUseCase {
	return &GetContactsUseCase{
		contactRepo: contactRepo,
	}
}

// Execute retrieves contacts with filters and pagination.
// Pagination is clamped here so a malformed query string can't request
// 1,000,000 rows; defaults if the caller passes 0/negative values.
func (uc *GetContactsUseCase) Execute(ctx context.Context, userID string, req *GetContactsRequest) (*GetContactsResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	} else if limit > 200 {
		limit = 200
	}
	opts := &ports.ContactFilterOptions{
		GroupID:    req.GroupID,
		IsFavorite: req.IsFavorite,
		Search:     req.Search,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       page,
		Limit:      limit,
	}

	result, err := uc.contactRepo.FindAll(ctx, userID, opts)
	if err != nil {
		return nil, err
	}

	contacts := make([]*ContactItem, 0, len(result.Contacts))
	for _, c := range result.Contacts {
		var groupID *string
		if c.GroupID != nil {
			gid := c.GroupID.String()
			groupID = &gid
		}

		contacts = append(contacts, &ContactItem{
			ContactID:      c.ContactID.String(),
			Name:           c.Name,
			WhatsAppNumber: c.WhatsAppNumber,
			GroupID:        groupID,
			GroupName:      c.GroupName,
			GroupColor:     c.GroupColor,
			IsFavorite:     c.IsFavorite,
			AvatarURL:      c.AvatarURL,
			Notes:          c.Notes,
			CreatedAt:      c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &GetContactsResponse{
		Contacts: contacts,
		Total:    result.Total,
		Page:     req.Page,
		Limit:    req.Limit,
	}, nil
}
