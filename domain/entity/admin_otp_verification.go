package entity

import (
	"time"

	"github.com/google/uuid"
)

// AdminOTPVerification represents an admin OTP verification record
type AdminOTPVerification struct {
	VerificationID uuid.UUID  `json:"verification_id"`
	AdminID        uuid.UUID  `json:"admin_id"`
	OTPCode        string     `json:"-"`
	ExpiresAt      time.Time  `json:"expires_at"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// NewAdminOTPVerification creates a new admin OTP verification instance
func NewAdminOTPVerification(adminID uuid.UUID, otpCode string, expiryDuration time.Duration) *AdminOTPVerification {
	now := time.Now()
	return &AdminOTPVerification{
		VerificationID: uuid.New(),
		AdminID:        adminID,
		OTPCode:        otpCode,
		ExpiresAt:      now.Add(expiryDuration),
		CreatedAt:      now,
	}
}

// IsExpired checks if the OTP has expired
func (o *AdminOTPVerification) IsExpired() bool {
	return time.Now().After(o.ExpiresAt)
}

// IsVerified checks if the OTP has been verified
func (o *AdminOTPVerification) IsVerified() bool {
	return o.VerifiedAt != nil
}

// IsValid checks if the OTP is valid (not expired and not verified)
func (o *AdminOTPVerification) IsValid() bool {
	return !o.IsExpired() && !o.IsVerified()
}

// MarkVerified marks the OTP as verified
func (o *AdminOTPVerification) MarkVerified() {
	now := time.Now()
	o.VerifiedAt = &now
}
