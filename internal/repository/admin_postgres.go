package repository

import (
	"context"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

type PostgresAdminRepository struct {
	db *sqlx.DB
}

// NewPostgresAdminRepository creates a new admin repository instance
func NewPostgresAdminRepository(db *sqlx.DB) ports.AdminRepository {
	return &PostgresAdminRepository{
		db: db,
	}
}

// FindByWhatsAppNumber finds an admin by their WhatsApp number
func (r *PostgresAdminRepository) FindByWhatsAppNumber(ctx context.Context, whatsappNumber string) (*entity.Admin, error) {
	const query = `
		SELECT admin_id, whatsapp_number, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM admins
		WHERE whatsapp_number = $1 AND deleted_at IS NULL
	`

	var admin entity.Admin
	err := r.db.GetContext(ctx, &admin, query, whatsappNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin by whatsapp: %w", err))
	}
	return &admin, nil
}

// FindByID finds an admin by ID
func (r *PostgresAdminRepository) FindByID(ctx context.Context, adminID string) (*entity.Admin, error) {
	const query = `
		SELECT admin_id, whatsapp_number, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM admins
		WHERE admin_id = $1 AND deleted_at IS NULL
	`

	var admin entity.Admin
	err := r.db.GetContext(ctx, &admin, query, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin by id: %w", err))
	}
	return &admin, nil
}

// Create creates a new admin
func (r *PostgresAdminRepository) Create(ctx context.Context, admin *entity.Admin) error {
	const query = `
		INSERT INTO admins (admin_id, whatsapp_number, password_hash, full_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		admin.AdminID, admin.WhatsAppNumber, admin.PasswordHash,
		admin.FullName, admin.Role, admin.IsActive,
		admin.CreatedAt, admin.UpdatedAt)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to create admin: %w", err))
	}
	return nil
}

// Update updates an existing admin
func (r *PostgresAdminRepository) Update(ctx context.Context, admin *entity.Admin) error {
	const query = `
		UPDATE admins
		SET full_name = $1,
		    updated_at = $2,
		    password_hash = COALESCE($3, password_hash),
		    is_active = COALESCE($4, is_active)
		WHERE admin_id = $5
	`

	_, err := r.db.ExecContext(ctx, query,
		admin.FullName, admin.UpdatedAt, admin.PasswordHash,
		admin.IsActive, admin.AdminID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update admin: %w", err))
	}
	return nil
}

// SoftDelete marks an admin as deleted
func (r *PostgresAdminRepository) SoftDelete(ctx context.Context, adminID string) error {
	const query = `
		UPDATE admins
		SET deleted_at = CURRENT_TIMESTAMP,
		    is_active = FALSE,
		    updated_at = CURRENT_TIMESTAMP
		WHERE admin_id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, adminID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to soft delete admin: %w", err))
	}
	return nil
}

// List returns all admins (super_admin only)
func (r *PostgresAdminRepository) List(ctx context.Context) ([]*entity.Admin, error) {
	const query = `
		SELECT admin_id, whatsapp_number, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM admins
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	var admins []*entity.Admin
	err := r.db.SelectContext(ctx, &admins, query)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to list admins: %w", err))
	}
	return admins, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *PostgresAdminRepository) UpdateLastLogin(ctx context.Context, adminID string) error {
	const query = `
		UPDATE admins
		SET last_login_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE admin_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, adminID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update last login: %w", err))
	}
	return nil
}

// CheckExistsByWhatsAppNumber checks if an admin with given WhatsApp exists
func (r *PostgresAdminRepository) CheckExistsByWhatsAppNumber(ctx context.Context, whatsappNumber string) (bool, error) {
	const query = `
		SELECT COUNT(*) as count
		FROM admins
		WHERE whatsapp_number = $1 AND deleted_at IS NULL
	`

	var count int
	err := r.db.GetContext(ctx, &count, query, whatsappNumber)
	if err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to check admin exists: %w", err))
	}
	return count > 0, nil
}
