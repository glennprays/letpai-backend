package ports

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// SessionBankAccountRepository is the persistence port for the
// per-session bank-accounts list.
type SessionBankAccountRepository interface {
	// FindBySessionID returns every live account for a session,
	// ordered by ordinal ascending. Soft-deleted rows are excluded.
	FindBySessionID(ctx context.Context, sessionID string) ([]*entity.SessionBankAccount, error)

	// BulkReplace overwrites the entire list for a session in one
	// transaction: soft-deletes every existing live row, then inserts
	// the provided accounts with fresh ordinals starting at 0.
	// Idempotent and FE-friendly — the form just PUTs the full list
	// every save and the server doesn't have to diff.
	BulkReplace(ctx context.Context, sessionID string, accounts []*entity.SessionBankAccount) error
}
