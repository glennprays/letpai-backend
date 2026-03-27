package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// BillItemRepository defines the interface for bill item data operations
type BillItemRepository interface {
	// Create creates a new bill item
	Create(ctx context.Context, item *entity.BillItem) error

	// FindByID finds a bill item by ID
	FindByID(ctx context.Context, billItemID string) (*entity.BillItem, error)

	// FindBySessionID finds all bill items for a session
	FindBySessionID(ctx context.Context, sessionID string) ([]*entity.BillItem, error)

	// Update updates a bill item
	Update(ctx context.Context, item *entity.BillItem) error

	// Delete deletes a bill item
	Delete(ctx context.Context, billItemID string) error

	// DeleteBySessionID deletes all bill items for a session
	DeleteBySessionID(ctx context.Context, sessionID string) error

	// SumBySessionID calculates the total amount of all bill items for a session
	SumBySessionID(ctx context.Context, sessionID string) (float64, error)
}
