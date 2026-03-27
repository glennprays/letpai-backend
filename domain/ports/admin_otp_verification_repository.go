package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// AdminOTPVerificationRepository defines the interface for admin OTP verification operations
type AdminOTPVerificationRepository interface {
	// Create creates a new admin OTP verification record
	Create(ctx context.Context, verification *entity.AdminOTPVerification) error

	// FindByOTPCode finds an admin OTP verification by OTP code
	FindByOTPCode(ctx context.Context, otpCode string) (*entity.AdminOTPVerification, error)

	// FindByAdminID finds all OTP verifications for an admin
	FindByAdminID(ctx context.Context, adminID string) ([]*entity.AdminOTPVerification, error)

	// MarkAsVerified marks an OTP as verified
	MarkAsVerified(ctx context.Context, verificationID string) error

	// InvalidatePreviousVerifications marks all previous OTPs as invalid/expired
	InvalidatePreviousVerifications(ctx context.Context, adminID string) error

	// CleanupExpired deletes expired OTP records older than specified duration
	CleanupExpired(ctx context.Context, olderThan interface{}) (int64, error)
}
