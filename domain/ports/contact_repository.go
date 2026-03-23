package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// ContactFilterOptions represents filter options for listing contacts
type ContactFilterOptions struct {
	GroupID    *string
	IsFavorite *bool
	Search     *string
	SortBy     string // name, created_at, group_name
	SortOrder  string // asc, desc
	Page       int
	Limit      int
}

// ContactListResult represents the result of listing contacts
type ContactListResult struct {
	Contacts []*entity.Contact
	Total    int
}

// ContactRepository defines the interface for contact data operations
type ContactRepository interface {
	// Create creates a new contact
	Create(ctx context.Context, contact *entity.Contact) error

	// FindByID finds a contact by ID
	FindByID(ctx context.Context, contactID string, userID string) (*entity.Contact, error)

	// FindAll finds contacts for a user with filters and pagination
	FindAll(ctx context.Context, userID string, opts *ContactFilterOptions) (*ContactListResult, error)

	// Update updates a contact
	Update(ctx context.Context, contact *entity.Contact) error

	// Delete performs a soft delete on a contact
	Delete(ctx context.Context, contactID string, userID string) error

	// BulkDelete performs soft delete on multiple contacts
	BulkDelete(ctx context.Context, contactIDs []string, userID string) error

	// BulkAddToGroup adds multiple contacts to a group
	BulkAddToGroup(ctx context.Context, contactIDs []string, groupID string, userID string) error

	// ExistsByWhatsApp checks if a contact with the WhatsApp number exists for the user
	ExistsByWhatsApp(ctx context.Context, userID string, whatsappNumber string, excludeID *string) (bool, error)
}
