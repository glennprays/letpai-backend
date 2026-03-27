package admin

type LoginRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	OTPCode        string `json:"otp_code" validate:"required,len=6,max=6"`
}

type LoginResponse struct {
	AdminID        string `json:"admin_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	Token          string `json:"token"`
}

type VerifyOTPRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	OTPCode        string `json:"otp_code" validate:"required,len=6,max=6"`
}

type VerifyOTPResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type GetProfileResponse struct {
	AdminID        string  `json:"admin_id"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	FullName       string  `json:"full_name"`
	Role           string  `json:"role"`
	IsVerified     bool    `json:"is_verified"`
	LastLoginAt    *string `json:"last_login_at,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type UpdateProfileRequest struct {
	FullName string `json:"full_name" validate:"omitempty,max=100"`
}

type UpdateProfileResponse struct {
	AdminID   string `json:"admin_id"`
	FullName  string `json:"full_name"`
	UpdatedAt string `json:"updated_at"`
}

type SetupPasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=100"`
}

type SetupPasswordResponse struct {
	Message string `json:"message"`
}
