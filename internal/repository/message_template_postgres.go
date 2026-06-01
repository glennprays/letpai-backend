package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresMessageTemplateRepository is the postgres adapter for the
// admin-managed message_templates table. `variables` is persisted as
// JSONB; we round-trip it through []string so callers don't have to
// know the storage shape.
type PostgresMessageTemplateRepository struct {
	db *sqlx.DB
}

func NewPostgresMessageTemplateRepository(db *sqlx.DB) ports.MessageTemplateRepository {
	return &PostgresMessageTemplateRepository{db: db}
}

func (r *PostgresMessageTemplateRepository) FindByKey(ctx context.Context, key string) (*entity.MessageTemplate, error) {
	const q = `
		SELECT template_id, key, name, description, body, variables, created_at, updated_at
		FROM message_templates
		WHERE key = $1
	`
	row := r.db.QueryRowContext(ctx, q, key)
	t, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return t, nil
}

func (r *PostgresMessageTemplateRepository) FindAll(ctx context.Context) ([]*entity.MessageTemplate, error) {
	const q = `
		SELECT template_id, key, name, description, body, variables, created_at, updated_at
		FROM message_templates
		ORDER BY key ASC
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	out := make([]*entity.MessageTemplate, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return out, nil
}

func (r *PostgresMessageTemplateRepository) Update(ctx context.Context, t *entity.MessageTemplate) error {
	varsJSON, err := json.Marshal(t.Variables)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	const q = `
		UPDATE message_templates
		SET name = $2, description = $3, body = $4, variables = $5, updated_at = $6
		WHERE template_id = $1
	`
	res, err := r.db.ExecContext(ctx, q, t.TemplateID, t.Name, t.Description, t.Body, string(varsJSON), time.Now())
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

// rowScanner is satisfied by both *sql.Row and *sql.Rows so scanTemplate
// works for both single and multi-row queries.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(s rowScanner) (*entity.MessageTemplate, error) {
	var (
		t        entity.MessageTemplate
		varsJSON []byte
	)
	if err := s.Scan(
		&t.TemplateID,
		&t.Key,
		&t.Name,
		&t.Description,
		&t.Body,
		&varsJSON,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(varsJSON) > 0 {
		var vars []string
		if err := json.Unmarshal(varsJSON, &vars); err == nil {
			t.Variables = vars
		}
	}
	if t.Variables == nil {
		t.Variables = []string{}
	}
	return &t, nil
}
