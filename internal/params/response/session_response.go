package response

// SessionResponse represents a session
type SessionResponse struct {
	SessionID   string  `json:"session_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
	Currency    string  `json:"currency"`
	SessionDate *string `json:"session_date,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// SessionDetailResponse represents detailed session information
type SessionDetailResponse struct {
	SessionID        string                `json:"session_id"`
	Title            string                `json:"title"`
	Description      string                `json:"description"`
	Status           string                `json:"status"`
	TotalAmount      float64               `json:"total_amount"`
	Currency         string                `json:"currency"`
	SessionDate      *string               `json:"session_date,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
	ParticipantCount int                   `json:"participant_count"`
	BillItemCount    int                   `json:"bill_item_count"`
	PaidCount        int                   `json:"paid_count"`
	Participants     []*ParticipantResponse `json:"participants"`
	Bills            []*BillItemResponse    `json:"bills"`
}

// ParticipantResponse represents a session participant
type ParticipantResponse struct {
	ParticipantID   string   `json:"participant_id"`
	ContactID       *string  `json:"contact_id,omitempty"`
	Name            string   `json:"name"`
	WhatsAppNumber  string   `json:"whatsapp_number"`
	AvatarURL       *string  `json:"avatar_url,omitempty"`
	ShareAmount     float64  `json:"share_amount"`
	PaymentStatus   string   `json:"payment_status"`
	PaymentProofURL *string  `json:"payment_proof_url,omitempty"`
}

// BillItemResponse represents a bill item
type BillItemResponse struct {
	BillItemID  string   `json:"bill_item_id"`
	Description string   `json:"description"`
	Amount      float64  `json:"amount"`
	Category    *string  `json:"category,omitempty"`
}

// SessionListResponse represents a paginated list of sessions
type SessionListResponse struct {
	Sessions []*SessionResponse `json:"sessions"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	Limit    int                `json:"limit"`
}
