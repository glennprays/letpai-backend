package request

import "time"

// CreateSessionRequest represents the request to create a session
type CreateSessionRequest struct {
	Title       string     `json:"title" validate:"required,min=1,max=200"`
	Description string     `json:"description" validate:"max=1000"`
	Currency    string     `json:"currency" validate:"required,len=3"`
	SessionDate *time.Time `json:"session_date,omitempty"`
}

// UpdateSessionRequest represents the request to update a session
type UpdateSessionRequest struct {
	Title       *string    `json:"title" validate:"omitempty,min=1,max=200"`
	Description *string    `json:"description" validate:"omitempty,max=1000"`
	Currency    *string    `json:"currency" validate:"omitempty,len=3"`
	SessionDate *time.Time `json:"session_date,omitempty"`
}

// GetSessionsQuery represents query parameters for listing sessions
type GetSessionsQuery struct {
	Status    *string `query:"status" validate:"omitempty,oneof=active completed cancelled"`
	Search    *string `query:"search"`
	SortBy    string  `query:"sort_by" validate:"omitempty,oneof=created_at sessiondate title total_amount"`
	SortOrder string  `query:"sort_order" validate:"omitempty,oneof=asc desc"`
	Page      int     `query:"page" validate:"omitempty,min=1"`
	Limit     int     `query:"limit" validate:"omitempty,min=1,max=100"`
}

// AddParticipantsRequest represents the request to add participants
type AddParticipantsRequest struct {
	Participants []AddParticipantRequest `json:"participants" validate:"required,min=1"`
}

// AddParticipantRequest represents a single participant to add
type AddParticipantRequest struct {
	ContactID      *string `json:"contact_id,omitempty"`
	CustomName     string  `json:"custom_name,omitempty"`
	CustomWhatsApp string  `json:"custom_whatsapp,omitempty"`
}

// UpdateParticipantRequest represents the request to update a participant
type UpdateParticipantRequest struct {
	CustomName     *string `json:"custom_name,omitempty"`
	CustomWhatsApp *string `json:"custom_whatsapp,omitempty"`
}

// AddBillItemRequest represents the request to add a bill item
type AddBillItemRequest struct {
	Description string  `json:"description" validate:"required,min=1,max=200"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Category    *string `json:"category,omitempty"`
}
