package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"strconv"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/jmoiron/sqlx"
)

// PostgresSessionRepository implements SessionRepository using PostgreSQL
type PostgresSessionRepository struct {
	db *sqlx.DB
}

// NewPostgresSessionRepository creates a new PostgreSQL session repository
func NewPostgresSessionRepository(db *sqlx.DB) ports.SessionRepository {
	return &PostgresSessionRepository{db: db}
}

// Create creates a new session
func (r *PostgresSessionRepository) Create(ctx context.Context, session *entity.Session) error {
	query := `
		INSERT INTO sessions (session_id, user_id, title, description, status, total_amount, currency, session_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		session.SessionID,
		session.UserID,
		session.Title,
		session.Description,
		session.Status,
		session.TotalAmount,
		session.Currency,
		session.SessionDate,
		session.CreatedAt,
		session.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a session by ID
func (r *PostgresSessionRepository) FindByID(ctx context.Context, sessionID string, userID string) (*entity.Session, error) {
	query := `
		SELECT session_id, user_id, title, description, status, total_amount, currency, session_date, created_at, updated_at, deleted_at
		FROM sessions
		WHERE session_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	row := r.db.QueryRowxContext(ctx, query, sessionID, userID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var session entity.Session
	var statusStr string
	err := row.Scan(
		&session.SessionID,
		&session.UserID,
		&session.Title,
		&session.Description,
		&statusStr,
		&session.TotalAmount,
		&session.Currency,
		&session.SessionDate,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	session.Status = valueobject.SessionStatus(statusStr)

	return &session, nil
}

// FindAll finds sessions for a user with filters and pagination
func (r *PostgresSessionRepository) FindAll(ctx context.Context, userID string, opts *ports.SessionFilterOptions) (*ports.SessionListResult, error) {
	if opts == nil {
		opts = &ports.SessionFilterOptions{
			Page:      1,
			Limit:     20,
			SortBy:    "created_at",
			SortOrder: "desc",
		}
	}

	// Validate pagination
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.Limit < 1 || opts.Limit > 100 {
		opts.Limit = 20
	}
	offset := (opts.Page - 1) * opts.Limit

	// Build WHERE clause
	whereConditions := []string{"user_id = $1", "deleted_at IS NULL"}
	args := []interface{}{userID}
	argIndex := 2

	if opts.Status != nil {
		whereConditions = append(whereConditions, "status = $"+strconv.Itoa(argIndex))
		args = append(args, string(*opts.Status))
		argIndex++
	}

	if opts.Search != nil && *opts.Search != "" {
		whereConditions = append(whereConditions, "(title ILIKE $"+strconv.Itoa(argIndex)+" OR description ILIKE $"+strconv.Itoa(argIndex+1)+")")
		searchPattern := "%" + *opts.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Validate sort
	validSortBy := map[string]bool{
		"created_at":   true,
		"session_date": true,
		"title":        true,
		"total_amount": true,
	}
	if !validSortBy[opts.SortBy] {
		opts.SortBy = "created_at"
	}

	validSortOrder := map[string]bool{
		"asc":  true,
		"desc": true,
	}
	if !validSortOrder[opts.SortOrder] {
		opts.SortOrder = "desc"
	}

	// Count query
	countQuery := `
		SELECT COUNT(*)
		FROM sessions
		WHERE ` + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Data query
	dataQuery := `
		SELECT session_id, user_id, title, description, status, total_amount, currency, session_date, created_at, updated_at, deleted_at
		FROM sessions
		WHERE ` + whereClause + `
		ORDER BY ` + opts.SortBy + ` ` + strings.ToUpper(opts.SortOrder) + `
		LIMIT $` + strconv.Itoa(argIndex) + ` OFFSET $` + strconv.Itoa(argIndex+1)
	args = append(args, opts.Limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &ports.SessionListResult{
				Sessions: []*entity.Session{},
				Total:    total,
			}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var sessions []*entity.Session
	for rows.Next() {
		var session entity.Session
		var statusStr string
		err = rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.Title,
			&session.Description,
			&statusStr,
			&session.TotalAmount,
			&session.Currency,
			&session.SessionDate,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.DeletedAt,
		)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		session.Status = valueobject.SessionStatus(statusStr)
		sessions = append(sessions, &session)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	if len(sessions) == 0 {
		return &ports.SessionListResult{
			Sessions: []*entity.Session{},
			Total:    total,
		}, nil
	}

	return &ports.SessionListResult{
		Sessions: sessions,
		Total:    total,
	}, nil
}

// Update updates a session
func (r *PostgresSessionRepository) Update(ctx context.Context, session *entity.Session) error {
	query := `
		UPDATE sessions
		SET title = $2, description = $3, status = $4, total_amount = $5, currency = $6, session_date = $7, updated_at = $8
		WHERE session_id = $1 AND user_id = $9 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		session.SessionID,
		session.Title,
		session.Description,
		session.Status,
		session.TotalAmount,
		session.Currency,
		session.SessionDate,
		session.UpdatedAt,
		session.UserID,
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

// Delete performs a soft delete on a session
func (r *PostgresSessionRepository) Delete(ctx context.Context, sessionID string, userID string) error {
	query := `
		UPDATE sessions
		SET deleted_at = $2, updated_at = $2
		WHERE session_id = $1 AND user_id = $3 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, sessionID, now, userID)
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

// UpdateTotalAmount updates the total amount of a session
func (r *PostgresSessionRepository) UpdateTotalAmount(ctx context.Context, sessionID string, totalAmount float64) error {
	query := `
		UPDATE sessions
		SET total_amount = $2, updated_at = NOW()
		WHERE session_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, totalAmount)
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

// UpdateStatus updates the status of a session
func (r *PostgresSessionRepository) UpdateStatus(ctx context.Context, sessionID string, status valueobject.SessionStatus) error {
	query := `
		UPDATE sessions
		SET status = $2, updated_at = NOW()
		WHERE session_id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, sessionID, status)
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
