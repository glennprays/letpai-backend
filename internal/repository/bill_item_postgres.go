package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresBillItemRepository implements BillItemRepository using PostgreSQL
type PostgresBillItemRepository struct {
	db *sqlx.DB
}

// NewPostgresBillItemRepository creates a new PostgreSQL bill item repository
func NewPostgresBillItemRepository(db *sqlx.DB) ports.BillItemRepository {
	return &PostgresBillItemRepository{db: db}
}

// Create creates a new bill item
func (r *PostgresBillItemRepository) Create(ctx context.Context, item *entity.BillItem) error {
	query := `
		INSERT INTO bill_items (bill_item_id, session_id, description, amount, category, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		item.BillItemID,
		item.SessionID,
		item.Description,
		item.Amount,
		item.Category,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a bill item by ID
func (r *PostgresBillItemRepository) FindByID(ctx context.Context, billItemID string) (*entity.BillItem, error) {
	query := `
		SELECT bill_item_id, session_id, description, amount, category, created_at, updated_at
		FROM bill_items
		WHERE bill_item_id = $1
	`

	var item entity.BillItem
	err := r.db.GetContext(ctx, &item, query, billItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &item, nil
}

// FindBySessionID finds all bill items for a session
func (r *PostgresBillItemRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*entity.BillItem, error) {
	query := `
		SELECT bill_item_id, session_id, description, amount, category, created_at, updated_at
		FROM bill_items
		WHERE session_id = $1
		ORDER BY created_at ASC
	`

	var items []*entity.BillItem
	err := r.db.SelectContext(ctx, &items, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.BillItem{}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return items, nil
}

// Update updates a bill item
func (r *PostgresBillItemRepository) Update(ctx context.Context, item *entity.BillItem) error {
	query := `
		UPDATE bill_items
		SET description = $2, amount = $3, category = $4, updated_at = $5
		WHERE bill_item_id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		item.BillItemID,
		item.Description,
		item.Amount,
		item.Category,
		item.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	if rows == 0 {
		return domain.NewError(domain.ErrNotFound, nil)
	}

	return nil
}

// Delete deletes a bill item
func (r *PostgresBillItemRepository) Delete(ctx context.Context, billItemID string) error {
	query := `
		DELETE FROM bill_items
		WHERE bill_item_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, billItemID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	if rows == 0 {
		return domain.NewError(domain.ErrNotFound, nil)
	}

	return nil
}

// DeleteBySessionID deletes all bill items for a session
func (r *PostgresBillItemRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	query := `
		DELETE FROM bill_items
		WHERE session_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// SumBySessionID calculates the total amount of all bill items for a session
func (r *PostgresBillItemRepository) SumBySessionID(ctx context.Context, sessionID string) (float64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM bill_items
		WHERE session_id = $1
	`

	var total float64
	err := r.db.GetContext(ctx, &total, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}

	return total, nil
}
