package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresUserRepository implements UserRepository using PostgreSQL
type PostgresUserRepository struct {
	db *sqlx.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *sqlx.DB) ports.UserRepository {
	return &PostgresUserRepository{db: db}
}

// Create creates a new user
func (r *PostgresUserRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (user_id, whatsapp_number, password_hash, full_name, avatar_url, is_verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.UserID,
		user.WhatsAppNumber,
		user.PasswordHash,
		user.FullName,
		user.AvatarURL,
		user.IsVerified,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a user by ID
func (r *PostgresUserRepository) FindByID(ctx context.Context, userID string) (*entity.User, error) {
	query := `
		SELECT user_id, whatsapp_number, password_hash, full_name, avatar_url, is_verified, created_at, updated_at, deleted_at
		FROM users
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	row := r.db.QueryRowxContext(ctx, query, userID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var user entity.User
	err := row.Scan(
		&user.UserID,
		&user.WhatsAppNumber,
		&user.PasswordHash,
		&user.FullName,
		&user.AvatarURL,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &user, nil
}

// FindByWhatsApp finds a user by WhatsApp number
func (r *PostgresUserRepository) FindByWhatsApp(ctx context.Context, whatsappNumber string) (*entity.User, error) {
	query := `
		SELECT user_id, whatsapp_number, password_hash, full_name, avatar_url, is_verified, created_at, updated_at, deleted_at
		FROM users
		WHERE whatsapp_number = $1 AND deleted_at IS NULL
	`

	row := r.db.QueryRowxContext(ctx, query, whatsappNumber)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var user entity.User
	err := row.Scan(
		&user.UserID,
		&user.WhatsAppNumber,
		&user.PasswordHash,
		&user.FullName,
		&user.AvatarURL,
		&user.IsVerified,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &user, nil
}

// Update updates user information
func (r *PostgresUserRepository) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users
		SET full_name = $2, avatar_url = $3, is_verified = $4, updated_at = $5
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.UserID,
		user.FullName,
		user.AvatarURL,
		user.IsVerified,
		user.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	if rows == 0 {
		return domain.NewError(domain.ErrNotFound, nil)
	}

	return nil
}

// Delete performs a soft delete on a user
func (r *PostgresUserRepository) Delete(ctx context.Context, userID string) error {
	query := `
		UPDATE users
		SET deleted_at = $2, updated_at = $2
		WHERE user_id = $1 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, userID, now)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	if rows == 0 {
		return domain.NewError(domain.ErrNotFound, nil)
	}

	return nil
}

// IsExists checks if a user exists by WhatsApp number
func (r *PostgresUserRepository) IsExists(ctx context.Context, whatsappNumber string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM users
		WHERE whatsapp_number = $1 AND deleted_at IS NULL
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, whatsappNumber).Scan(&count)
	if err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, err)
	}

	return count > 0, nil
}
