package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

// PostgresContactRepository implements ContactRepository using PostgreSQL
type PostgresContactRepository struct {
	db *sqlx.DB
}

// NewPostgresContactRepository creates a new PostgreSQL contact repository
func NewPostgresContactRepository(db *sqlx.DB) ports.ContactRepository {
	return &PostgresContactRepository{db: db}
}

// Create creates a new contact
func (r *PostgresContactRepository) Create(ctx context.Context, contact *entity.Contact) error {
	query := `
		INSERT INTO contacts (contact_id, user_id, name, whatsapp_number, group_id, is_favorite, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		contact.ContactID,
		contact.UserID,
		contact.Name,
		contact.WhatsAppNumber,
		contact.GroupID,
		contact.IsFavorite,
		contact.CreatedAt,
		contact.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a contact by ID
func (r *PostgresContactRepository) FindByID(ctx context.Context, contactID string, userID string) (*entity.Contact, error) {
	query := `
		SELECT c.contact_id, c.user_id, c.name, c.whatsapp_number, c.group_id, c.is_favorite,
		       c.created_at, c.updated_at, c.deleted_at,
		       g.name as group_name, g.color as group_color
		FROM contacts c
		LEFT JOIN contact_groups g ON c.group_id = g.group_id AND g.deleted_at IS NULL
		WHERE c.contact_id = $1 AND c.user_id = $2 AND c.deleted_at IS NULL
	`

	row := r.db.QueryRowxContext(ctx, query, contactID, userID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var contact entity.Contact
	err := row.Scan(
		&contact.ContactID,
		&contact.UserID,
		&contact.Name,
		&contact.WhatsAppNumber,
		&contact.GroupID,
		&contact.IsFavorite,
		&contact.CreatedAt,
		&contact.UpdatedAt,
		&contact.DeletedAt,
		&contact.GroupName,
		&contact.GroupColor,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &contact, nil
}

// FindAll finds contacts for a user with filters and pagination
func (r *PostgresContactRepository) FindAll(ctx context.Context, userID string, opts *ports.ContactFilterOptions) (*ports.ContactListResult, error) {
	if opts == nil {
		opts = &ports.ContactFilterOptions{
			Page:    1,
			Limit:   20,
			SortBy:  "created_at",
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
	whereConditions := []string{"c.user_id = $1", "c.deleted_at IS NULL"}
	args := []interface{}{userID}
	argIndex := 2

	if opts.GroupID != nil {
		whereConditions = append(whereConditions, "c.group_id = $"+string(rune('0'+argIndex)))
		args = append(args, *opts.GroupID)
		argIndex++
	}

	if opts.IsFavorite != nil {
		whereConditions = append(whereConditions, "c.is_favorite = $"+string(rune('0'+argIndex)))
		args = append(args, *opts.IsFavorite)
		argIndex++
	}

	if opts.Search != nil && *opts.Search != "" {
		whereConditions = append(whereConditions, "(c.name ILIKE $"+string(rune('0'+argIndex))+" OR c.whatsapp_number ILIKE $"+string(rune('0'+argIndex+1))+")")
		searchPattern := "%" + *opts.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Validate sort
	validSortBy := map[string]bool{
		"name":        true,
		"created_at":  true,
		"group_name":  true,
		"whatsapp_number": true,
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
		FROM contacts c
		WHERE ` + whereClause
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Data query
	dataQuery := `
		SELECT c.contact_id, c.user_id, c.name, c.whatsapp_number, c.group_id, c.is_favorite,
		       c.created_at, c.updated_at, c.deleted_at,
		       g.name as group_name, g.color as group_color
		FROM contacts c
		LEFT JOIN contact_groups g ON c.group_id = g.group_id AND g.deleted_at IS NULL
		WHERE ` + whereClause + `
		ORDER BY c.` + opts.SortBy + ` ` + strings.ToUpper(opts.SortOrder) + `
		LIMIT $` + string(rune('0'+argIndex)) + ` OFFSET $` + string(rune('0'+argIndex+1))
	args = append(args, opts.Limit, offset)

	rows, err := r.db.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &ports.ContactListResult{
				Contacts: []*entity.Contact{},
				Total:    total,
			}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var contacts []*entity.Contact
	for rows.Next() {
		var contact entity.Contact
		err = rows.Scan(
			&contact.ContactID,
			&contact.UserID,
			&contact.Name,
			&contact.WhatsAppNumber,
			&contact.GroupID,
			&contact.IsFavorite,
			&contact.CreatedAt,
			&contact.UpdatedAt,
			&contact.DeletedAt,
			&contact.GroupName,
			&contact.GroupColor,
		)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		contacts = append(contacts, &contact)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	if len(contacts) == 0 {
		return &ports.ContactListResult{
			Contacts: []*entity.Contact{},
			Total:    total,
		}, nil
	}

	return &ports.ContactListResult{
		Contacts: contacts,
		Total:    total,
	}, nil
}

// Update updates a contact
func (r *PostgresContactRepository) Update(ctx context.Context, contact *entity.Contact) error {
	query := `
		UPDATE contacts
		SET name = $2, whatsapp_number = $3, group_id = $4, is_favorite = $5, updated_at = $6
		WHERE contact_id = $1 AND user_id = $7 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		contact.ContactID,
		contact.Name,
		contact.WhatsAppNumber,
		contact.GroupID,
		contact.IsFavorite,
		contact.UpdatedAt,
		contact.UserID,
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

// Delete performs a soft delete on a contact
func (r *PostgresContactRepository) Delete(ctx context.Context, contactID string, userID string) error {
	query := `
		UPDATE contacts
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE contact_id = $1 AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, contactID, userID)
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

// BulkDelete performs soft delete on multiple contacts
func (r *PostgresContactRepository) BulkDelete(ctx context.Context, contactIDs []string, userID string) error {
	if len(contactIDs) == 0 {
		return nil
	}

	query := `
		UPDATE contacts
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE contact_id = ANY($1) AND user_id = $2 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, contactIDs, userID)
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

// BulkAddToGroup adds multiple contacts to a group
func (r *PostgresContactRepository) BulkAddToGroup(ctx context.Context, contactIDs []string, groupID string, userID string) error {
	if len(contactIDs) == 0 {
		return nil
	}

	query := `
		UPDATE contacts
		SET group_id = $1, updated_at = NOW()
		WHERE contact_id = ANY($2) AND user_id = $3 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, groupID, contactIDs, userID)
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

// ExistsByWhatsApp checks if a contact with the WhatsApp number exists for the user
func (r *PostgresContactRepository) ExistsByWhatsApp(ctx context.Context, userID string, whatsappNumber string, excludeID *string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM contacts
		WHERE user_id = $1 AND whatsapp_number = $2 AND deleted_at IS NULL
	`
	args := []interface{}{userID, whatsappNumber}

	if excludeID != nil {
		query += " AND contact_id != $3"
		args = append(args, *excludeID)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, err)
	}

	return count > 0, nil
}
