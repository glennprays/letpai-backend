package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// OTPService handles OTP generation and validation
type OTPService struct {
	defaultExpiry time.Duration
}

// NewOTPService creates a new OTP service
func NewOTPService(expiryMinutes int) *OTPService {
	return &OTPService{
		defaultExpiry: time.Duration(expiryMinutes) * time.Minute,
	}
}

// Generate generates a 6-digit OTP code
func (s *OTPService) Generate() (string, error) {
	// Generate 6 random digits
	otp := make([]byte, 6)
	for i := range otp {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("failed to generate random digit: %w", err)
		}
		otp[i] = byte(n.Int64() + '0')
	}
	return string(otp), nil
}

// GetExpiry returns the default expiry duration
func (s *OTPService) GetExpiry() time.Duration {
	return s.defaultExpiry
}

// FormatWhatsAppMessage formats the OTP message for WhatsApp
func (s *OTPService) FormatWhatsAppMessage(otpCode string) string {
	return fmt.Sprintf("*Letpai*\n\nYour verification code is: *%s*\n\nThis code will expire in %d minutes.\n\nIf you didn't request this code, please ignore this message.",
		otpCode,
		int(s.defaultExpiry.Minutes()),
	)
}

// ValidateCode validates if a string is a valid 6-digit OTP code
func (s *OTPService) ValidateCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
