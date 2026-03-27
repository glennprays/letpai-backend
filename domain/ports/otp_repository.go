package ports

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// OTPRepository defines the interface for OTP data operations
type OTPRepository interface {
	// Create creates a new OTP verification record
	Create(ctx context.Context, otp *entity.OTPVerification) error

	// FindValidByPhone finds a valid (not expired, not used) OTP by phone number
	FindValidByPhone(ctx context.Context, whatsappNumber string) (*entity.OTPVerification, error)

	// FindByID finds an OTP by ID
	FindByID(ctx context.Context, otpID string) (*entity.OTPVerification, error)

	// MarkAsUsed marks an OTP as used
	MarkAsUsed(ctx context.Context, otpID string) error

	// InvalidatePreviousOTPs marks all previous OTPs for a phone number as used/invalid
	InvalidatePreviousOTPs(ctx context.Context, whatsappNumber string) error

	// CleanupExpired deletes expired OTP records older than specified duration
	CleanupExpired(ctx context.Context, olderThan time.Duration) (int64, error)
}
