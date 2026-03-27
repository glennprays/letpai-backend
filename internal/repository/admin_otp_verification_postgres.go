package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

type PostgresAdminOTPVerificationRepository struct {
	db *sqlx.DB
}

// NewPostgresAdminOTPVerificationRepository creates a new admin OTP verification repository instance
func NewPostgresAdminOTPVerificationRepository(db *sqlx.DB) ports.AdminOTPVerificationRepository {
	return &PostgresAdminOTPVerificationRepository{
		db: db,
	}
}

// Create creates a new admin OTP verification record
func (r *PostgresAdminOTPVerificationRepository) Create(ctx context.Context, verification *entity.AdminOTPVerification) error {
	const query = `
		INSERT INTO admin_otp_verifications (verification_id, admin_id, otp_code, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		verification.VerificationID,
		verification.AdminID,
		verification.OTPCode,
		verification.ExpiresAt,
		verification.CreatedAt,
	)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to create admin OTP verification: %w", err))
	}
	return nil
}

// FindByOTPCode finds an admin OTP verification by OTP code
func (r *PostgresAdminOTPVerificationRepository) FindByOTPCode(ctx context.Context, otpCode string) (*entity.AdminOTPVerification, error) {
	const query = `
		SELECT verification_id, admin_id, otp_code, expires_at, verified_at, created_at
		FROM admin_otp_verifications
		WHERE otp_code = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var verification entity.AdminOTPVerification
	err := r.db.GetContext(ctx, &verification, query, otpCode)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin OTP by code: %w", err))
	}
	return &verification, nil
}

// FindByAdminID finds all OTP verifications for an admin
func (r *PostgresAdminOTPVerificationRepository) FindByAdminID(ctx context.Context, adminID string) ([]*entity.AdminOTPVerification, error) {
	const query = `
		SELECT verification_id, admin_id, otp_code, expires_at, verified_at, created_at
		FROM admin_otp_verifications
		WHERE admin_id = $1
		ORDER BY created_at DESC
	`

	var verifications []*entity.AdminOTPVerification
	err := r.db.SelectContext(ctx, &verifications, query, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin OTPs by admin ID: %w", err))
	}
	return verifications, nil
}

// MarkAsVerified marks an OTP as verified
func (r *PostgresAdminOTPVerificationRepository) MarkAsVerified(ctx context.Context, verificationID string) error {
	const query = `
		UPDATE admin_otp_verifications
		SET verified_at = CURRENT_TIMESTAMP
		WHERE verification_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, verificationID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to mark OTP as verified: %w", err))
	}
	return nil
}

// InvalidatePreviousVerifications marks all previous OTPs as invalid
func (r *PostgresAdminOTPVerificationRepository) InvalidatePreviousVerifications(ctx context.Context, adminID string) error {
	const query = `
		UPDATE admin_otp_verifications
		SET verified_at = CURRENT_TIMESTAMP
		WHERE admin_id = $1 AND verified_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, adminID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to invalidate previous verifications: %w", err))
	}
	return nil
}

// CleanupExpired deletes expired OTP records older than specified duration
func (r *PostgresAdminOTPVerificationRepository) CleanupExpired(ctx context.Context, olderThan interface{}) (int64, error) {
	const query = `
		DELETE FROM admin_otp_verifications
		WHERE expires_at < $1
	`

	cutoff := time.Now().Add(-24 * time.Hour) // Delete OTPs older than 24 hours
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to cleanup expired OTPs: %w", err))
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get rows affected: %w", err))
	}

	return count, nil
}
