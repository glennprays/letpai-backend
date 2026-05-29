package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
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

// FindByWhatsAppNumber finds an admin by their WhatsApp number.
// Discriminates sql.ErrNoRows → ErrNotFound so callers get 404, not 500.
//
// password_hash is included in the SELECT because the password Login
// use case needs it for bcrypt verification. The hash never leaves the
// process via JSON (Admin.PasswordHash carries `json:"-"`).
func (r *PostgresAdminRepository) FindByWhatsAppNumber(ctx context.Context, whatsappNumber string) (*entity.Admin, error) {
	const query = `
		SELECT admin_id, whatsapp_number, password_hash, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM admins
		WHERE whatsapp_number = $1 AND deleted_at IS NULL
	`

	var admin entity.Admin
	err := r.db.GetContext(ctx, &admin, query, whatsappNumber)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin by whatsapp: %w", err))
	}
	return &admin, nil
}

// FindByID finds an admin by ID.
// Discriminates sql.ErrNoRows → ErrNotFound so callers get 404, not 500.
// Includes password_hash (see FindByWhatsAppNumber for the rationale).
func (r *PostgresAdminRepository) FindByID(ctx context.Context, adminID string) (*entity.Admin, error) {
	const query = `
		SELECT admin_id, whatsapp_number, password_hash, full_name, role, is_active, last_login_at, created_at, updated_at
		FROM admins
		WHERE admin_id = $1 AND deleted_at IS NULL
	`

	var admin entity.Admin
	err := r.db.GetContext(ctx, &admin, query, adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
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

// CountActiveSuperAdmins returns the number of non-deleted, active super_admin
// rows — used as a safety check before demoting/deleting one.
func (r *PostgresAdminRepository) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	const query = `
		SELECT COUNT(*)
		FROM admins
		WHERE role = 'super_admin' AND is_active = TRUE AND deleted_at IS NULL
	`

	var count int
	if err := r.db.GetContext(ctx, &count, query); err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to count super admins: %w", err))
	}
	return count, nil
}

// hasUsableSuperAdminSQL is shared by HasUsableSuperAdmin and the
// re-check inside UpsertBootstrapSuperAdmin so the placeholder rule
// lives in exactly one place. The literal here must stay in sync with
// migrations/000013_seed_super_admin.up.sql.
const hasUsableSuperAdminSQL = `
	SELECT EXISTS (
		SELECT 1 FROM admins
		WHERE role = 'super_admin'
		  AND is_active = TRUE
		  AND deleted_at IS NULL
		  AND password_hash IS NOT NULL
		  AND password_hash <> ''
		  AND password_hash NOT LIKE '$2a$10$xxxxxxxxxx%'
	)
`

// HasUsableSuperAdmin returns true once at least one active, non-deleted
// super_admin row carries a real bcrypt password (i.e. not the seed
// placeholder). Drives the first-boot wizard gate.
func (r *PostgresAdminRepository) HasUsableSuperAdmin(ctx context.Context) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, hasUsableSuperAdminSQL); err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to check bootstrap state: %w", err))
	}
	return exists, nil
}

// UpsertBootstrapSuperAdmin atomically claims the bootstrap slot.
//
// Approach:
//  1. Open a SERIALIZABLE transaction.
//  2. Re-check the gate inside the transaction. If a usable super admin
//     already exists, refuse with ErrConflict (which the handler maps
//     to a 409). A second concurrent setup request that loses the race
//     will hit this branch — or a Postgres serialization error, which
//     bubbles up as ErrInternalFailure and the FE shows it as
//     "another admin completed setup; please refresh".
//  3. Try to UPDATE an existing placeholder row in place. The WHERE
//     clause matches only rows that look like the seed placeholder
//     (empty / NULL hash or the xxxxx literal).
//  4. If no placeholder row exists (the seed migration was skipped or
//     someone wiped the table), INSERT a fresh row.
func (r *PostgresAdminRepository) UpsertBootstrapSuperAdmin(ctx context.Context, admin *entity.Admin) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to begin tx: %w", err))
	}
	defer func() { _ = tx.Rollback() }()

	var alreadyUsable bool
	if err := tx.GetContext(ctx, &alreadyUsable, hasUsableSuperAdminSQL); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to recheck bootstrap state: %w", err))
	}
	if alreadyUsable {
		return domain.NewError(domain.ErrConflict, errors.New("bootstrap not allowed: a super admin already exists"))
	}

	const updatePlaceholder = `
		UPDATE admins
		SET whatsapp_number = $1,
		    password_hash   = $2,
		    full_name       = $3,
		    is_active       = TRUE,
		    updated_at      = NOW()
		WHERE role = 'super_admin'
		  AND deleted_at IS NULL
		  AND (
		      password_hash IS NULL
		      OR password_hash = ''
		      OR password_hash LIKE '$2a$10$xxxxxxxxxx%'
		  )
		RETURNING admin_id
	`

	var updatedID string
	err = tx.GetContext(ctx, &updatedID, updatePlaceholder,
		admin.WhatsAppNumber, admin.PasswordHash, admin.FullName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update placeholder admin: %w", err))
	}

	if errors.Is(err, sql.ErrNoRows) {
		// No placeholder row matched — insert a fresh super admin.
		const insertFresh = `
			INSERT INTO admins (admin_id, whatsapp_number, password_hash, full_name, role, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'super_admin', TRUE, $5, $6)
		`
		if _, err := tx.ExecContext(ctx, insertFresh,
			admin.AdminID, admin.WhatsAppNumber, admin.PasswordHash, admin.FullName,
			admin.CreatedAt, admin.UpdatedAt); err != nil {
			return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to insert bootstrap admin: %w", err))
		}
	} else {
		// The caller's admin.AdminID is the freshly-minted UUID it
		// expected to use; align it with the row we actually updated
		// so the handler can return the correct admin_id.
		parsed, parseErr := uuid.Parse(updatedID)
		if parseErr == nil {
			admin.AdminID = parsed
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to commit bootstrap tx: %w", err))
	}
	return nil
}
