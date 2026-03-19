package request

// LoginRequest represents the request to login
type LoginRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
	Password       string `json:"password" validate:"required"`
}
