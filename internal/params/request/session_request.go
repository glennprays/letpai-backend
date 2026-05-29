package request

import "time"

// CreateSessionRequest represents the request to create a session.
// Bank fields are optional; the host can add them later via Update.
type CreateSessionRequest struct {
	Title             string     `json:"title" validate:"required,min=1,max=200"`
	Description       string     `json:"description" validate:"max=1000"`
	Currency          string     `json:"currency" validate:"required,len=3"`
	SessionDate       *time.Time `json:"session_date,omitempty"`
	BankName          *string    `json:"bank_name,omitempty" validate:"omitempty,max=80"`
	BankAccountNumber *string    `json:"bank_account_number,omitempty" validate:"omitempty,max=40"`
	BankAccountHolder *string    `json:"bank_account_holder,omitempty" validate:"omitempty,max=80"`
}

// UpdateSessionRequest represents the request to update a session.
//
// Bank fields use pointer semantics so the FE can distinguish "leave
// alone" (omitted) from "explicit clear" (empty string).
type UpdateSessionRequest struct {
	Title             *string    `json:"title" validate:"omitempty,min=1,max=200"`
	Description       *string    `json:"description" validate:"omitempty,max=1000"`
	Currency          *string    `json:"currency" validate:"omitempty,len=3"`
	SessionDate       *time.Time `json:"session_date,omitempty"`
	BankName          *string    `json:"bank_name,omitempty" validate:"omitempty,max=80"`
	BankAccountNumber *string    `json:"bank_account_number,omitempty" validate:"omitempty,max=40"`
	BankAccountHolder *string    `json:"bank_account_holder,omitempty" validate:"omitempty,max=80"`
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

// UpdateParticipantRequest represents the request to update a participant's
// custom name / WhatsApp number. Payment-status transitions (submit / approve /
// reject) go through the dedicated /payments/* endpoints instead — see
// PaymentHandler.SubmitPayment / ApprovePayment / RejectPayment.
type UpdateParticipantRequest struct {
	CustomName     *string `json:"custom_name,omitempty"`
	CustomWhatsApp *string `json:"custom_whatsapp,omitempty"`
}

// AddBillItemRequest represents the request to add a bill item.
//
// ParticipantIDs is optional. Empty/missing → "split equally across every
// participant in the session". Non-empty → scope the bill to those
// participants. This field used to be missing here, so the FE's
// per-participant assignment was silently dropped at the handler
// boundary and every bill was treated as everyone's.
type AddBillItemRequest struct {
	Description    string   `json:"description" validate:"required,min=1,max=200"`
	Amount         float64  `json:"amount" validate:"required,gt=0"`
	Category       *string  `json:"category,omitempty"`
	ParticipantIDs []string `json:"participant_ids,omitempty" validate:"omitempty,dive,uuid"`
}

// UpdateBillItemRequest represents a request to update a bill item.
//
// ParticipantIDs is a pointer so we can distinguish between
// "absent (don't touch assignments)" and "explicit empty array
// (reset to everyone)". Same drop-at-boundary bug applied here.
type UpdateBillItemRequest struct {
	Description    string    `json:"description" validate:"omitempty,min=1,max=500"`
	Amount         float64   `json:"amount" validate:"omitempty,gt=0"`
	Category       string    `json:"category,omitempty"`
	ParticipantIDs *[]string `json:"participant_ids,omitempty" validate:"omitempty,dive,uuid"`
}
