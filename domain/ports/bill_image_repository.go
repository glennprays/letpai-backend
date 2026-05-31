package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// BillImageRepository defines the interface for bill image data operations.
//
// SECURITY: methods are scoped by session_id only. Callers reached via an
// authenticated HTTP route MUST verify session ownership before touching
// bill images.
type BillImageRepository interface {
	// Create persists a new bill image.
	Create(ctx context.Context, image *entity.BillImage) error

	// FindByID finds a bill image by its ID.
	FindByID(ctx context.Context, billImageID string) (*entity.BillImage, error)

	// FindBySessionID returns all non-deleted bill images for a session,
	// ordered by ordinal.
	FindBySessionID(ctx context.Context, sessionID string) ([]*entity.BillImage, error)

	// Delete soft-deletes a bill image.
	Delete(ctx context.Context, billImageID string) error

	// NextOrdinal returns the next display ordinal for a session
	// (max existing ordinal + 1, or 0 if none exist).
	NextOrdinal(ctx context.Context, sessionID string) (int, error)
}
