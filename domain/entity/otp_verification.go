package entity

import (
	"time"

	"github.com/google/uuid"
)

// OTPVerification represents an OTP verification record
type OTPVerification struct {
	OTPID          uuid.UUID  `json:"otp_id"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	WhatsAppNumber string     `json:"whatsapp_number"`
	OTPCode        string     `json:"-"` // Never expose OTP in JSON
	ExpiresAt      time.Time  `json:"expires_at"`
	IsUsed         bool       `json:"is_used"`
	CreatedAt      time.Time  `json:"created_at"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
}

// NewOTPVerification creates a new OTP verification instance
func NewOTPVerification(whatsappNumber, otpCode string, expiryDuration time.Duration) *OTPVerification {
	now := time.Now()
	return &OTPVerification{
		OTPID:          uuid.New(),
		WhatsAppNumber: whatsappNumber,
		OTPCode:        otpCode,
		ExpiresAt:      now.Add(expiryDuration),
		IsUsed:         false,
		CreatedAt:      now,
	}
}

// NewOTPVerificationWithUser creates a new OTP verification instance with user ID
func NewOTPVerificationWithUser(userID uuid.UUID, whatsappNumber, otpCode string, expiryDuration time.Duration) *OTPVerification {
	otp := NewOTPVerification(whatsappNumber, otpCode, expiryDuration)
	otp.UserID = &userID
	return otp
}

// IsExpired checks if the OTP has expired
func (o *OTPVerification) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// MarkUsed marks the OTP as used
func (o *OTPVerification) MarkUsed() {
	now := time.Now()
	o.IsUsed = true
	o.UsedAt = &now
}

// IsValid checks if the OTP is valid (not expired and not used)
func (o *OTPVerification) IsValid() bool {
	return !o.IsExpired() && !o.IsUsed
}
