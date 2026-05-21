package request

// ForgotPasswordRequest represents the body of POST /auth/forgot-password.
type ForgotPasswordRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,min=10,max=20"`
}
