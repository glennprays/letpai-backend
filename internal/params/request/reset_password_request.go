package request

// ResetPasswordRequest represents the body of POST /auth/reset-password.
type ResetPasswordRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,min=10,max=20"`
	OTPCode        string `json:"otp_code" validate:"required,len=6"`
	NewPassword    string `json:"new_password" validate:"required,min=8"`
}
