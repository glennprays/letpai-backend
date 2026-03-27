package request

// VerifyOTPRequest represents the request to verify OTP
type VerifyOTPRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
	OTPCode        string `json:"otp_code" validate:"required,len=6"`
}
