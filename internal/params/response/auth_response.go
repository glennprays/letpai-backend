package response

import "time"

// RegisterResponse represents the response after user registration
type RegisterResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	UserID    string    `json:"user_id,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token"`
	User    *User  `json:"user"`
}

// VerifyOTPResponse represents the response after OTP verification
type VerifyOTPResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token"`
	User    *User  `json:"user"`
}

// LogoutResponse represents the response after logout
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// UpdateProfileResponse represents the response after a successful profile update
type UpdateProfileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user"`
}

// ForgotPasswordResponse represents the response after initiating a password reset.
type ForgotPasswordResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// User represents user data in responses
type User struct {
	UserID         string `json:"user_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
}
