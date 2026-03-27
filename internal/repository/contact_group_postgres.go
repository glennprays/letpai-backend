package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresContactGroupRepository implements ContactGroupRepository using PostgreSQL
type PostgresContactGroupRepository struct {
	db *sqlx.DB
}

// NewPostgresContactGroupRepository creates a new PostgreSQL contact group repository
func NewPostgresContactGroupRepository(db *sqlx.DB) ports.ContactGroupRepository {
	return &PostgresContactGroupRepository{db: db}
}

// Create creates a new contact group
func (r *PostgresContactGroupRepository) Create(ctx context.Context, group *entity.ContactGroup) error {
	query := `
		INSERT INTO contact_groups (group_id, user_id, name, color, sort_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		group.GroupID,
		group.UserID,
		group.Name,
		group.Color,
		group.SortOrder,
		group.CreatedAt,
		group.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a contact group by ID
func (r *PostgresContactGroupRepository) FindByID(ctx context.Context, groupID string, userID string) (*entity.ContactGroup, error) {
	query := `
		SELECT group_id, user_id, name, color, sort_order, created_at, updated_at, deleted_at
		FROM contact_groups
		WHERE group_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	row := r.db.QueryRowxContext(ctx, query, groupID, userID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var group entity.ContactGroup
	err := row.Scan(
		&group.GroupID,
		&group.UserID,
		&group.Name,
		&group.Color,
		&group.SortOrder,
		&group.CreatedAt,
		&group.UpdatedAt,
		&group.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &group, nil
}

// FindAll finds all contact groups for a user
func (r *PostgresContactGroupRepository) FindAll(ctx context.Context, userID string) ([]*entity.ContactGroup, error) {
	query := `
		SELECT group_id, user_id, name, color, sort_order, created_at, updated_at, deleted_at
		FROM contact_groups
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.ContactGroup{}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var groups []*entity.ContactGroup
	for rows.Next() {
		var group entity.ContactGroup
		err = rows.Scan(
			&group.GroupID,
			&group.UserID,
			&group.Name,
			&group.Color,
			&group.SortOrder,
			&group.CreatedAt,
			&group.UpdatedAt,
			&group.DeletedAt,
		)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		groups = append(groups, &group)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	if len(groups) == 0 {
		return []*entity.ContactGroup{}, nil
	}

	return groups, nil
}

// Update updates a contact group
func (r *PostgresContactGroupRepository) Update(ctx context.Context, group *entity.ContactGroup) error {
	query := `
		UPDATE contact_groups
		SET name = $2, color = $3, sort_order = $4, updated_at = $5
		WHERE group_id = $1 AND user_id = $6 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		group.GroupID,
		group.Name,
		group.Color,
		group.SortOrder,
		group.UpdatedAt,
		group.UserID,
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

// Delete performs a soft delete on a contact group
func (r *PostgresContactGroupRepository) Delete(ctx context.Context, groupID string, userID string) error {
	query := `
		UPDATE contact_groups
		SET deleted_at = $2, updated_at = $2
		WHERE group_id = $1 AND user_id = $3 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, groupID, now, userID)
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

// ExistsByName checks if a group with the same name exists for the user
func (r *PostgresContactGroupRepository) ExistsByName(ctx context.Context, userID string, name string, excludeID *string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM contact_groups
		WHERE user_id = $1 AND name = $2 AND deleted_at IS NULL
	`
	args := []interface{}{userID, name}

	if excludeID != nil {
		query += " AND group_id != $3"
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, err)
	}

	return count > 0, nil
}
