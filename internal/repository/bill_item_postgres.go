package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgresBillItemRepository implements BillItemRepository using PostgreSQL.
//
// All Create/Update calls open a transaction so that the bill_items row
// and its associated bill_item_participants rows are written atomically.
// Reads load the join rows in a single follow-up query and attach them
// to the in-memory entity (BillItem.ParticipantIDs).
type PostgresBillItemRepository struct {
	db *sqlx.DB
}

// NewPostgresBillItemRepository creates a new PostgreSQL bill item repository
func NewPostgresBillItemRepository(db *sqlx.DB) ports.BillItemRepository {
	return &PostgresBillItemRepository{db: db}
}

// Create creates a new bill item along with its participant assignments.
func (r *PostgresBillItemRepository) Create(ctx context.Context, item *entity.BillItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const insertItem = `
		INSERT INTO bill_items (bill_item_id, session_id, description, amount, category, includes_service_charge, includes_tax, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	if _, err := tx.ExecContext(
		ctx,
		insertItem,
		item.BillItemID,
		item.SessionID,
		item.Description,
		item.Amount,
		item.Category,
		item.IncludesServiceCharge,
		item.IncludesTax,
		item.CreatedAt,
		item.UpdatedAt,
	); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	if err := insertJoinRows(ctx, tx, item.BillItemID, item.ParticipantIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// FindByID finds a bill item by ID and attaches its participant IDs.
func (r *PostgresBillItemRepository) FindByID(ctx context.Context, billItemID string) (*entity.BillItem, error) {
	const query = `
		SELECT bill_item_id, session_id, description, amount, category, includes_service_charge, includes_tax, created_at, updated_at
		FROM bill_items
		WHERE bill_item_id = $1
	`
	row := r.db.QueryRowxContext(ctx, query, billItemID)

	var item entity.BillItem
	if err := row.Scan(
		&item.BillItemID,
		&item.SessionID,
		&item.Description,
		&item.Amount,
		&item.Category,
		&item.IncludesServiceCharge,
		&item.IncludesTax,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	ids, err := r.participantIDsForBills(ctx, []uuid.UUID{item.BillItemID})
	if err != nil {
		return nil, err
	}
	item.ParticipantIDs = ids[item.BillItemID]

	return &item, nil
}

// FindBySessionID finds all bill items for a session, hydrating each one's
// participant IDs in a single follow-up query.
func (r *PostgresBillItemRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*entity.BillItem, error) {
	const query = `
		SELECT bill_item_id, session_id, description, amount, category, includes_service_charge, includes_tax, created_at, updated_at
		FROM bill_items
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.BillItem{}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	items := make([]*entity.BillItem, 0)
	billIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var item entity.BillItem
		if err := rows.Scan(
			&item.BillItemID,
			&item.SessionID,
			&item.Description,
			&item.Amount,
			&item.Category,
			&item.IncludesServiceCharge,
			&item.IncludesTax,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		items = append(items, &item)
		billIDs = append(billIDs, item.BillItemID)
	}
	if err = rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	if len(items) == 0 {
		return items, nil
	}

	joined, err := r.participantIDsForBills(ctx, billIDs)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		it.ParticipantIDs = joined[it.BillItemID]
	}

	return items, nil
}

// Update updates a bill item and rewrites its participant assignments
// atomically. An empty ParticipantIDs slice clears all rows (i.e. resets
// the bill to "everyone in the session").
func (r *PostgresBillItemRepository) Update(ctx context.Context, item *entity.BillItem) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const updateItem = `
		UPDATE bill_items
		SET description = $2, amount = $3, category = $4, includes_service_charge = $5, includes_tax = $6, updated_at = $7
		WHERE bill_item_id = $1
	`
	res, err := tx.ExecContext(
		ctx,
		updateItem,
		item.BillItemID,
		item.Description,
		item.Amount,
		item.Category,
		item.IncludesServiceCharge,
		item.IncludesTax,
		item.UpdatedAt,
	)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	if rows == 0 {
		return domain.NewError(domain.ErrNotFound, nil)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM bill_item_participants WHERE bill_item_id = $1`, item.BillItemID); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	if err := insertJoinRows(ctx, tx, item.BillItemID, item.ParticipantIDs); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// Delete deletes a bill item. Join rows are cascade-deleted by the FK.
func (r *PostgresBillItemRepository) Delete(ctx context.Context, billItemID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM bill_items WHERE bill_item_id = $1`, billItemID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	if rows == 0 {
		return domain.NewError(domain.ErrNotFound, nil)
	}
	return nil
}

// DeleteBySessionID deletes every bill item for a session. Join rows
// cascade-delete via the FK on bill_item_participants.
func (r *PostgresBillItemRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM bill_items WHERE session_id = $1`, sessionID); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// SumBySessionID returns the total amount of every bill item in a session.
func (r *PostgresBillItemRepository) SumBySessionID(ctx context.Context, sessionID string) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM bill_items WHERE session_id = $1`,
		sessionID,
	).Scan(&total)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}
	return total, nil
}

// participantIDsForBills loads the participant_ids for a set of bill items
// and returns them grouped by bill_item_id. Empty result for a given id
// means "applies to everyone".
func (r *PostgresBillItemRepository) participantIDsForBills(ctx context.Context, billIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	out := make(map[uuid.UUID][]uuid.UUID, len(billIDs))
	if len(billIDs) == 0 {
		return out, nil
	}

	q, args, err := sqlx.In(
		`SELECT bill_item_id, participant_id FROM bill_item_participants WHERE bill_item_id IN (?)`,
		billIDs,
	)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	q = r.db.Rebind(q)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	for rows.Next() {
		var billID, pid uuid.UUID
		if err := rows.Scan(&billID, &pid); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		out[billID] = append(out[billID], pid)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return out, nil
}

// insertJoinRows writes the bill_item_participants rows for a given bill.
// A nil/empty slice is a no-op (legacy "everyone" semantics).
func insertJoinRows(ctx context.Context, tx *sqlx.Tx, billItemID uuid.UUID, participantIDs []uuid.UUID) error {
	if len(participantIDs) == 0 {
		return nil
	}
	const stmt = `INSERT INTO bill_item_participants (bill_item_id, participant_id) VALUES ($1, $2)`
	for _, pid := range participantIDs {
		if _, err := tx.ExecContext(ctx, stmt, billItemID, pid); err != nil {
			return domain.NewError(domain.ErrInternalFailure, err)
		}
	}
	return nil
}
