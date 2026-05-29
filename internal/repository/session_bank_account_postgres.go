package repository

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresSessionBankAccountRepository is the postgres adapter for
// session_bank_accounts.
type PostgresSessionBankAccountRepository struct {
	db *sqlx.DB
}

func NewPostgresSessionBankAccountRepository(db *sqlx.DB) ports.SessionBankAccountRepository {
	return &PostgresSessionBankAccountRepository{db: db}
}

func (r *PostgresSessionBankAccountRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*entity.SessionBankAccount, error) {
	const q = `
		SELECT account_id, session_id, ordinal, bank_name, account_number, account_holder, created_at, updated_at, deleted_at
		  FROM session_bank_accounts
		 WHERE session_id = $1
		   AND deleted_at IS NULL
		 ORDER BY ordinal ASC
	`
	rows, err := r.db.QueryxContext(ctx, q, sessionID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	out := make([]*entity.SessionBankAccount, 0)
	for rows.Next() {
		var a entity.SessionBankAccount
		if err := rows.StructScan(&a); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		copy := a
		out = append(out, &copy)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return out, nil
}

// BulkReplace soft-deletes every live row for the session and inserts
// the provided accounts in one transaction. The new rows receive
// fresh ordinals starting at 0 — caller order wins. Empty accounts
// (nothing typed in any field) are dropped silently so the FE can
// PUT a sparse list without server-side cleanup. If the resulting
// list is empty, the session ends up with no bank rows at all (the
// public payment page renders "no transfer details set").
func (r *PostgresSessionBankAccountRepository) BulkReplace(ctx context.Context, sessionID string, accounts []*entity.SessionBankAccount) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()

	// Soft-delete the existing live rows first. Using soft-delete
	// (not DELETE) so a future history view can show "host had two
	// accounts, then dropped to one" without losing the older rows.
	const softDelete = `
		UPDATE session_bank_accounts
		   SET deleted_at = $2,
		       updated_at = $2
		 WHERE session_id = $1
		   AND deleted_at IS NULL
	`
	if _, err := tx.ExecContext(ctx, softDelete, sessionID, now); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	const insert = `
		INSERT INTO session_bank_accounts (account_id, session_id, ordinal, bank_name, account_number, account_holder, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
	`
	sessionUUID, perr := uuid.Parse(sessionID)
	if perr != nil {
		return domain.NewError(domain.ErrBadRequest, perr)
	}

	ordinal := 0
	for _, a := range accounts {
		a.Normalize()
		if a.IsEmpty() {
			continue
		}
		if _, err := tx.ExecContext(
			ctx,
			insert,
			uuid.New(),
			sessionUUID,
			ordinal,
			a.BankName,
			a.AccountNumber,
			a.AccountHolder,
			now,
		); err != nil {
			return domain.NewError(domain.ErrInternalFailure, err)
		}
		ordinal++
	}

	if err := tx.Commit(); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}
