package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/errors"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresOTPRepository implements OTPRepository using PostgreSQL
type PostgresOTPRepository struct {
	db *sqlx.DB
}

// NewPostgresOTPRepository creates a new PostgreSQL OTP repository
func NewPostgresOTPRepository(db *sqlx.DB) ports.OTPRepository {
	return &PostgresOTPRepository{db: db}
}

// Create creates a new OTP verification record
func (r *PostgresOTPRepository) Create(ctx context.Context, otp *entity.OTPVerification) error {
	query := `
		INSERT INTO otp_verifications (otp_id, user_id, whatsapp_number, otp_code, expires_at, is_used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		otp.OTPID,
		otp.UserID,
		otp.WhatsAppNumber,
		otp.OTPCode,
		otp.ExpiresAt,
		otp.IsUsed,
		otp.CreatedAt,
	)

	if err != nil {
		return errors.NewError(errors.ErrInternalFailure, err)
	}

	return nil
}

// FindValidByPhone finds a valid (not expired, not used) OTP by phone number
func (r *PostgresOTPRepository) FindValidByPhone(ctx context.Context, whatsappNumber string) (*entity.OTPVerification, error) {
	query := `
		SELECT otp_id, user_id, whatsapp_number, otp_code, expires_at, is_used, created_at, used_at
		FROM otp_verifications
		WHERE whatsapp_number = $1 AND is_used = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var otp entity.OTPVerification
	err := r.db.GetContext(ctx, &otp, query, whatsappNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.NewError(errors.ErrNotFound, nil)
		}
		return nil, errors.NewError(errors.ErrInternalFailure, err)
	}

	return &otp, nil
}

// FindByID finds an OTP by ID
func (r *PostgresOTPRepository) FindByID(ctx context.Context, otpID string) (*entity.OTPVerification, error) {
	query := `
		SELECT otp_id, user_id, whatsapp_number, otp_code, expires_at, is_used, created_at, used_at
		FROM otp_verifications
		WHERE otp_id = $1
	`

	var otp entity.OTPVerification
	err := r.db.GetContext(ctx, &otp, query, otpID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.NewError(errors.ErrNotFound, nil)
		}
		return nil, errors.NewError(errors.ErrInternalFailure, err)
	}

	return &otp, nil
}

// MarkAsUsed marks an OTP as used
func (r *PostgresOTPRepository) MarkAsUsed(ctx context.Context, otpID string) error {
	query := `
		UPDATE otp_verifications
		SET is_used = TRUE, used_at = $2
		WHERE otp_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, otpID, time.Now())
	if err != nil {
		return errors.NewError(errors.ErrInternalFailure, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewError(errors.ErrInternalFailure, err)
	}

	if rows == 0 {
		return errors.NewError(errors.ErrNotFound, nil)
	}

	return nil
}

// InvalidatePreviousOTPs marks all previous OTPs for a phone number as used/invalid
func (r *PostgresOTPRepository) InvalidatePreviousOTPs(ctx context.Context, whatsappNumber string) error {
	query := `
		UPDATE otp_verifications
		SET is_used = TRUE, used_at = $2
		WHERE whatsapp_number = $1 AND is_used = FALSE
	`

	_, err := r.db.ExecContext(ctx, query, whatsappNumber, time.Now())
	if err != nil {
		return errors.NewError(errors.ErrInternalFailure, err)
	}

	return nil
}

// CleanupExpired deletes expired OTP records older than specified duration
func (r *PostgresOTPRepository) CleanupExpired(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM otp_verifications
		WHERE expires_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, errors.NewError(errors.ErrInternalFailure, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewError(errors.ErrInternalFailure, err)
	}

	return rows, nil
}
