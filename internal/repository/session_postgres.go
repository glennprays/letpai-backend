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
	"github.com/glennprays/letpai-backend/pkg/slug"
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

// Create creates a new session.
//
// PublicSlug is generated here (and re-rolled on UNIQUE collision)
// so callers don't have to know about the alphabet, length, or
// retry policy. The collision probability at our scale is ~1e-7
// per insert; 5 tries gives effectively zero probability of giving
// up.
func (r *PostgresSessionRepository) Create(ctx context.Context, session *entity.Session) error {
	const query = `
		INSERT INTO sessions (session_id, public_slug, user_id, title, description, status, total_amount, currency, session_date, bank_name, bank_account_number, bank_account_holder, service_charge_percentage, tax_percentage, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	const maxTries = 5
	for tries := 0; tries < maxTries; tries++ {
		if session.PublicSlug == "" {
			s, err := slug.New()
			if err != nil {
				return domain.NewError(domain.ErrInternalFailure, err)
			}
			session.PublicSlug = s
		}
		_, err := r.db.ExecContext(
			ctx,
			query,
			session.SessionID,
			session.PublicSlug,
			session.UserID,
			session.Title,
			session.Description,
			session.Status,
			session.TotalAmount,
			session.Currency,
			session.SessionDate,
			session.BankName,
			session.BankAccountNumber,
			session.BankAccountHolder,
			session.ServiceChargePercentage,
			session.TaxPercentage,
			session.CreatedAt,
			session.UpdatedAt,
		)
		if err == nil {
			return nil
		}
		if isUniqueViolationOnSlug(err) {
			// Re-roll: clear the slug and loop.
			session.PublicSlug = ""
			continue
		}
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return domain.NewError(domain.ErrInternalFailure, errors.New("public_slug collision after retries"))
}

// isUniqueViolationOnSlug reports whether err is a Postgres UNIQUE
// violation on a *_public_slug index. We branch on this for the slug
// retry loop without rolling back any other unique-violation paths
// the caller might rely on.
func isUniqueViolationOnSlug(err error) bool {
	type sqlState interface{ SQLState() string }
	if ss, ok := err.(sqlState); ok && ss.SQLState() == "23505" {
		msg := err.Error()
		return contains(msg, "public_slug")
	}
	// Fallback for drivers that don't expose SQLState — match the
	// PQ-style error message text.
	msg := err.Error()
	return contains(msg, "23505") && contains(msg, "public_slug")
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// FindByID finds a session by ID.
//
// userID is the authenticated host's id. Pass an empty string to skip
// the ownership filter — used by the public payment-page lookup, which
// scopes by the participant token (the page is only reachable if you
// have the token, so we don't double-gate by host).
func (r *PostgresSessionRepository) FindByID(ctx context.Context, sessionID string, userID string) (*entity.Session, error) {
	query := `
		SELECT session_id, public_slug, user_id, title, description, status, total_amount, currency, session_date, bank_name, bank_account_number, bank_account_holder, service_charge_percentage, tax_percentage, last_notified_at, created_at, updated_at, deleted_at
		FROM sessions
		WHERE session_id = $1 AND deleted_at IS NULL
	`
	args := []any{sessionID}
	if userID != "" {
		query += " AND user_id = $2"
		args = append(args, userID)
	}

	row := r.db.QueryRowxContext(ctx, query, args...)
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
		&session.PublicSlug,
		&session.UserID,
		&session.Title,
		&session.Description,
		&statusStr,
		&session.TotalAmount,
		&session.Currency,
		&session.SessionDate,
		&session.BankName,
		&session.BankAccountNumber,
		&session.BankAccountHolder,
		&session.ServiceChargePercentage,
		&session.TaxPercentage,
		&session.LastNotifiedAt,
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

// FindBySlug looks up a session by its public_slug. Mirrors FindByID's
// userID semantics — empty string skips the ownership filter.
func (r *PostgresSessionRepository) FindBySlug(ctx context.Context, slug string, userID string) (*entity.Session, error) {
	query := `
		SELECT session_id, public_slug, user_id, title, description, status, total_amount, currency, session_date, bank_name, bank_account_number, bank_account_holder, service_charge_percentage, tax_percentage, last_notified_at, created_at, updated_at, deleted_at
		FROM sessions
		WHERE public_slug = $1 AND deleted_at IS NULL
	`
	args := []any{slug}
	if userID != "" {
		query += " AND user_id = $2"
		args = append(args, userID)
	}

	row := r.db.QueryRowxContext(ctx, query, args...)
	var session entity.Session
	var statusStr string
	err := row.Scan(
		&session.SessionID,
		&session.PublicSlug,
		&session.UserID,
		&session.Title,
		&session.Description,
		&statusStr,
		&session.TotalAmount,
		&session.Currency,
		&session.SessionDate,
		&session.BankName,
		&session.BankAccountNumber,
		&session.BankAccountHolder,
		&session.ServiceChargePercentage,
		&session.TaxPercentage,
		&session.LastNotifiedAt,
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

	// Data query.
	//
	// The LEFT JOIN sub-aggregate hydrates the per-session
	// participant_count / paid_count counters the dashboard cards
	// render. Before this, those fields were never populated on the
	// list path, so every card said "0 of 0 paid" regardless of
	// actual settlement status. payment_status='paid' is the value
	// MarkPaidManually writes, so manual marks are included
	// automatically by the FILTER clause.
	dataQuery := `
		SELECT s.session_id, s.public_slug, s.user_id, s.title, s.description, s.status, s.total_amount, s.currency, s.session_date, s.bank_name, s.bank_account_number, s.bank_account_holder, s.service_charge_percentage, s.tax_percentage, s.last_notified_at, s.created_at, s.updated_at, s.deleted_at,
		       COALESCE(sp_agg.participant_count, 0)::int AS participant_count,
		       COALESCE(sp_agg.paid_count, 0)::int        AS paid_count
		FROM sessions s
		LEFT JOIN (
		    SELECT session_id,
		           COUNT(*)::int AS participant_count,
		           COUNT(*) FILTER (WHERE payment_status = 'paid')::int AS paid_count
		    FROM session_participants
		    GROUP BY session_id
		) sp_agg ON sp_agg.session_id = s.session_id
		WHERE ` + strings.ReplaceAll(whereClause, "user_id", "s.user_id") + `
		ORDER BY s.` + opts.SortBy + ` ` + strings.ToUpper(opts.SortOrder) + `
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
		var participantCount, paidCount int
		err = rows.Scan(
			&session.SessionID,
			&session.PublicSlug,
			&session.UserID,
			&session.Title,
			&session.Description,
			&statusStr,
			&session.TotalAmount,
			&session.Currency,
			&session.SessionDate,
			&session.BankName,
			&session.BankAccountNumber,
			&session.BankAccountHolder,
			&session.ServiceChargePercentage,
			&session.TaxPercentage,
			&session.LastNotifiedAt,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.DeletedAt,
			&participantCount,
			&paidCount,
		)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		session.Status = valueobject.SessionStatus(statusStr)
		pc, pd := participantCount, paidCount
		session.ParticipantCount = &pc
		session.PaidCount = &pd
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

// MarkNotified stamps last_notified_at without touching updated_at.
//
// Bumping updated_at here would defeat the whole dirty-for-notify
// gate (the predicate compares updated_at to last_notified_at, so
// touching the former would instantly re-mark the session dirty).
// The triggers in migration 000021 only fire on child tables, not
// on `sessions`, so this UPDATE is genuinely the only writer that
// can leave updated_at untouched.
func (r *PostgresSessionRepository) MarkNotified(ctx context.Context, sessionID string, at time.Time) error {
	const query = `
		UPDATE sessions
		   SET last_notified_at = $2
		 WHERE session_id = $1
		   AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, sessionID, at)
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

// Update updates a session
func (r *PostgresSessionRepository) Update(ctx context.Context, session *entity.Session) error {
	query := `
		UPDATE sessions
		SET title = $2, description = $3, status = $4, total_amount = $5, currency = $6, session_date = $7, bank_name = $8, bank_account_number = $9, bank_account_holder = $10, service_charge_percentage = $11, tax_percentage = $12, updated_at = $13
		WHERE session_id = $1 AND user_id = $14 AND deleted_at IS NULL
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
		session.BankName,
		session.BankAccountNumber,
		session.BankAccountHolder,
		session.ServiceChargePercentage,
		session.TaxPercentage,
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
