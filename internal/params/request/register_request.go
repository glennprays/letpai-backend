package request

// RegisterRequest represents the request to register a new user
type RegisterRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,phone"`
	Password       string `json:"password" validate:"required,min=8"`
}
