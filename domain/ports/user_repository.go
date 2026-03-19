package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *entity.User) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, userID string) (*entity.User, error)

	// FindByWhatsApp finds a user by WhatsApp number
	FindByWhatsApp(ctx context.Context, whatsappNumber string) (*entity.User, error)

	// Update updates user information
	Update(ctx context.Context, user *entity.User) error

	// Delete performs a soft delete on a user
	Delete(ctx context.Context, userID string) error

	// IsExists checks if a user exists by WhatsApp number
	IsExists(ctx context.Context, whatsappNumber string) (bool, error)
}
