package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// AdminRepository defines the interface for admin data operations
type AdminRepository interface {
	// FindByWhatsAppNumber finds an admin by their WhatsApp number
	FindByWhatsAppNumber(ctx context.Context, whatsappNumber string) (*entity.Admin, error)

	// FindByID finds an admin by ID
	FindByID(ctx context.Context, adminID string) (*entity.Admin, error)

	// Create creates a new admin
	Create(ctx context.Context, admin *entity.Admin) error

	// Update updates an existing admin
	Update(ctx context.Context, admin *entity.Admin) error

	// SoftDelete marks an admin as deleted
	SoftDelete(ctx context.Context, adminID string) error

	// List returns all admins (super_admin only)
	List(ctx context.Context) ([]*entity.Admin, error)

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, adminID string) error

	// CheckExistsByWhatsAppNumber checks if an admin with given WhatsApp exists
	CheckExistsByWhatsAppNumber(ctx context.Context, whatsappNumber string) (bool, error)

	// CountActiveSuperAdmins returns the number of non-deleted, active
	// super_admin rows — used as a safety check before demoting/deleting one.
	CountActiveSuperAdmins(ctx context.Context) (int, error)

	// HasUsableSuperAdmin returns true once at least one active, non-deleted
	// super_admin row carries a real bcrypt password (i.e. not the seed
	// placeholder). Drives the first-boot wizard gate at
	// GET /admin/auth/needs-setup and POST /admin/auth/bootstrap.
	HasUsableSuperAdmin(ctx context.Context) (bool, error)

	// UpsertBootstrapSuperAdmin atomically claims the bootstrap slot. It
	// re-checks HasUsableSuperAdmin inside a SERIALIZABLE transaction so
	// two concurrent /admin/auth/bootstrap calls can't both succeed: only
	// the first commit lands. If a placeholder seed row exists, it's
	// updated in place; otherwise a fresh row is inserted.
	UpsertBootstrapSuperAdmin(ctx context.Context, admin *entity.Admin) error
}
