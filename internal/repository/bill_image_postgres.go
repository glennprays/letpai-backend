package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresBillImageRepository implements BillImageRepository using PostgreSQL.
type PostgresBillImageRepository struct {
	db *sqlx.DB
}

// NewPostgresBillImageRepository creates a new PostgreSQL bill image repository.
func NewPostgresBillImageRepository(db *sqlx.DB) ports.BillImageRepository {
	return &PostgresBillImageRepository{db: db}
}

// Create persists a new bill image.
func (r *PostgresBillImageRepository) Create(ctx context.Context, img *entity.BillImage) error {
	const query = `
		INSERT INTO bill_images (bill_image_id, session_id, image_url, thumbnail_url, file_name, file_format, file_size, ordinal, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(
		ctx,
		query,
		img.BillImageID,
		img.SessionID,
		img.ImageURL,
		img.ThumbnailURL,
		img.FileName,
		img.FileFormat,
		img.FileSize,
		img.Ordinal,
		img.CreatedAt,
		img.UpdatedAt,
	)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// FindByID finds a bill image by its ID.
func (r *PostgresBillImageRepository) FindByID(ctx context.Context, billImageID string) (*entity.BillImage, error) {
	const query = `
		SELECT bill_image_id, session_id, image_url, thumbnail_url, file_name, file_format, file_size, ordinal, created_at, updated_at
		FROM bill_images
		WHERE bill_image_id = $1 AND deleted_at IS NULL
	`
	var img entity.BillImage
	err := r.db.QueryRowxContext(ctx, query, billImageID).Scan(
		&img.BillImageID,
		&img.SessionID,
		&img.ImageURL,
		&img.ThumbnailURL,
		&img.FileName,
		&img.FileFormat,
		&img.FileSize,
		&img.Ordinal,
		&img.CreatedAt,
		&img.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return &img, nil
}

// FindBySessionID returns all non-deleted bill images for a session, ordered by ordinal.
func (r *PostgresBillImageRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*entity.BillImage, error) {
	const query = `
		SELECT bill_image_id, session_id, image_url, thumbnail_url, file_name, file_format, file_size, ordinal, created_at, updated_at
		FROM bill_images
		WHERE session_id = $1 AND deleted_at IS NULL
		ORDER BY ordinal ASC, created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.BillImage{}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var images []*entity.BillImage
	for rows.Next() {
		var img entity.BillImage
		if err := rows.Scan(
			&img.BillImageID,
			&img.SessionID,
			&img.ImageURL,
			&img.ThumbnailURL,
			&img.FileName,
			&img.FileFormat,
			&img.FileSize,
			&img.Ordinal,
			&img.CreatedAt,
			&img.UpdatedAt,
		); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		images = append(images, &img)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	if images == nil {
		images = []*entity.BillImage{}
	}
	return images, nil
}

// Delete soft-deletes a bill image.
func (r *PostgresBillImageRepository) Delete(ctx context.Context, billImageID string) error {
	const query = `
		UPDATE bill_images
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE bill_image_id = $1 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, billImageID)
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

// NextOrdinal returns the next display ordinal for a session.
func (r *PostgresBillImageRepository) NextOrdinal(ctx context.Context, sessionID string) (int, error) {
	var maxOrd *int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT MAX(ordinal) FROM bill_images WHERE session_id = $1 AND deleted_at IS NULL`,
		sessionID,
	).Scan(&maxOrd)
	if err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}
	if maxOrd == nil {
		return 0, nil
	}
	return *maxOrd + 1, nil
}
