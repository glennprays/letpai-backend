package response

// ContactGroupResponse represents a contact group
type ContactGroupResponse struct {
	GroupID   string `json:"group_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
}

// ContactResponse represents a contact
type ContactResponse struct {
	ContactID      string  `json:"contact_id"`
	Name           string  `json:"name"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	GroupID        *string `json:"group_id,omitempty"`
	GroupName      *string `json:"group_name,omitempty"`
	GroupColor     *string `json:"group_color,omitempty"`
	IsFavorite     bool    `json:"is_favorite"`
	CreatedAt      string  `json:"created_at"`
}

// ContactListResponse represents a paginated list of contacts
type ContactListResponse struct {
	Contacts []*ContactResponse `json:"contacts"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	Limit    int                `json:"limit"`
}
