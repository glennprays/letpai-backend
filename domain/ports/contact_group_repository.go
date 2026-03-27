package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// ContactGroupRepository defines the interface for contact group data operations
type ContactGroupRepository interface {
	// Create creates a new contact group
	Create(ctx context.Context, group *entity.ContactGroup) error

	// FindByID finds a contact group by ID
	FindByID(ctx context.Context, groupID string, userID string) (*entity.ContactGroup, error)

	// FindAll finds all contact groups for a user
	FindAll(ctx context.Context, userID string) ([]*entity.ContactGroup, error)

	// Update updates a contact group
	Update(ctx context.Context, group *entity.ContactGroup) error

	// Delete performs a soft delete on a contact group
	Delete(ctx context.Context, groupID string, userID string) error

	// ExistsByName checks if a group with the same name exists for the user
	ExistsByName(ctx context.Context, userID string, name string, excludeID *string) (bool, error)
}
