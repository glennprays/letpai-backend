package request

// CreateContactRequest represents the request to create a contact
type CreateContactRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=100"`
	WhatsAppNumber string  `json:"whatsapp_number" validate:"required"`
	GroupID        *string `json:"group_id,omitempty"`
}

// UpdateContactRequest represents the request to update a contact
type UpdateContactRequest struct {
	Name           *string `json:"name" validate:"omitempty,min=1,max=100"`
	WhatsAppNumber *string `json:"whatsapp_number" validate:"omitempty"`
	GroupID        *string `json:"group_id,omitempty"`
	IsFavorite     *bool   `json:"is_favorite,omitempty"`
}

// BulkContactsRequest represents the request for bulk contact operations
type BulkContactsRequest struct {
	Operation  string   `json:"operation" validate:"required,oneof=add_to_group delete"`
	ContactIDs []string `json:"contact_ids" validate:"required,min=1"`
	GroupID    string   `json:"group_id,omitempty"`
}

// GetContactsQuery represents query parameters for listing contacts
type GetContactsQuery struct {
	GroupID    *string `query:"group_id"`
	IsFavorite *bool   `query:"is_favorite"`
	Search     *string `query:"search"`
	SortBy     string  `query:"sort_by" validate:"omitempty,oneof=name created_at group_name whatsappnumber"`
	SortOrder  string  `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Page       int     `query:"page" validate:"omitempty,min=1"`
	Limit      int     `query:"limit" validate:"omitempty,min=1,max=100"`
}
