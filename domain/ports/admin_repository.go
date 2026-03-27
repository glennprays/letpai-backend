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
}
