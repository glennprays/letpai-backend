package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// SessionFilterOptions represents filter options for listing sessions
type SessionFilterOptions struct {
	Status    *valueobject.SessionStatus
	Search    *string
	SortBy    string // created_at, session_date, title, total_amount
	SortOrder string // asc, desc
	Page      int
	Limit     int
}

// SessionListResult represents the result of listing sessions
type SessionListResult struct {
	Sessions []*entity.Session
	Total    int
}

// SessionRepository defines the interface for session data operations
type SessionRepository interface {
	// Create creates a new session
	Create(ctx context.Context, session *entity.Session) error

	// FindByID finds a session by ID
	FindByID(ctx context.Context, sessionID string, userID string) (*entity.Session, error)

	// FindAll finds sessions for a user with filters and pagination
	FindAll(ctx context.Context, userID string, opts *SessionFilterOptions) (*SessionListResult, error)

	// Update updates a session
	Update(ctx context.Context, session *entity.Session) error

	// Delete performs a soft delete on a session
	Delete(ctx context.Context, sessionID string, userID string) error

	// UpdateTotalAmount updates the total amount of a session.
	// SECURITY: not scoped by user_id. Callers MUST verify session ownership
	// via FindByID(ctx, sessionID, userID) first.
	UpdateTotalAmount(ctx context.Context, sessionID string, totalAmount float64) error

	// UpdateStatus updates the status of a session.
	// SECURITY: not scoped by user_id. Callers MUST verify session ownership
	// via FindByID(ctx, sessionID, userID) first.
	UpdateStatus(ctx context.Context, sessionID string, status valueobject.SessionStatus) error
}
