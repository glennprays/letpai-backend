package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/jmoiron/sqlx"
)

// PostgresParticipantRepository implements ParticipantRepository using PostgreSQL
type PostgresParticipantRepository struct {
	db *sqlx.DB
}

// NewPostgresParticipantRepository creates a new PostgreSQL participant repository
func NewPostgresParticipantRepository(db *sqlx.DB) ports.ParticipantRepository {
	return &PostgresParticipantRepository{db: db}
}

// Create creates a new participant
func (r *PostgresParticipantRepository) Create(ctx context.Context, participant *entity.SessionParticipant) error {
	query := `
		INSERT INTO session_participants (participant_id, session_id, contact_id, custom_name, custom_whatsapp, share_amount, payment_status, payment_proof_url, rejection_count, rejection_reason, notification_count, last_notification_at, joined_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		participant.ParticipantID,
		participant.SessionID,
		participant.ContactID,
		participant.CustomName,
		participant.CustomWhatsApp,
		participant.ShareAmount,
		participant.PaymentStatus,
		participant.PaymentProofURL,
		participant.RejectionCount,
		participant.RejectionReason,
		participant.NotificationCount,
		participant.LastNotificationAt,
		participant.JoinedAt,
		participant.UpdatedAt,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a participant by ID
func (r *PostgresParticipantRepository) FindByID(ctx context.Context, participantID string) (*entity.SessionParticipant, error) {
	query := `
		SELECT participant_id, session_id, contact_id, custom_name, custom_whatsapp, share_amount, payment_status, payment_proof_url, rejection_count, rejection_reason, notification_count, last_notification_at, joined_at, updated_at
		FROM session_participants
		WHERE participant_id = $1
	`

	row := r.db.QueryRowxContext(ctx, query, participantID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var participant entity.SessionParticipant
	var paymentStatusStr string
	err := row.Scan(
		&participant.ParticipantID,
		&participant.SessionID,
		&participant.ContactID,
		&participant.CustomName,
		&participant.CustomWhatsApp,
		&participant.ShareAmount,
		&paymentStatusStr,
		&participant.PaymentProofURL,
		&participant.RejectionCount,
		&participant.RejectionReason,
		&participant.NotificationCount,
		&participant.LastNotificationAt,
		&participant.JoinedAt,
		&participant.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	participant.PaymentStatus = valueobject.PaymentStatus(paymentStatusStr)

	return &participant, nil
}

// FindBySessionID finds all participants for a session
func (r *PostgresParticipantRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*entity.SessionParticipant, error) {
	query := `
		SELECT participant_id, session_id, contact_id, custom_name, custom_whatsapp, share_amount, payment_status, payment_proof_url, rejection_count, rejection_reason, notification_count, last_notification_at, joined_at, updated_at
		FROM session_participants
		WHERE session_id = $1
		ORDER BY joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.SessionParticipant{}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var participants []*entity.SessionParticipant
	for rows.Next() {
		var participant entity.SessionParticipant
		var paymentStatusStr string
		err = rows.Scan(
			&participant.ParticipantID,
			&participant.SessionID,
			&participant.ContactID,
			&participant.CustomName,
			&participant.CustomWhatsApp,
			&participant.ShareAmount,
			&paymentStatusStr,
			&participant.PaymentProofURL,
			&participant.RejectionCount,
			&participant.RejectionReason,
			&participant.NotificationCount,
			&participant.LastNotificationAt,
			&participant.JoinedAt,
			&participant.UpdatedAt,
		)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		participant.PaymentStatus = valueobject.PaymentStatus(paymentStatusStr)
		participants = append(participants, &participant)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	if len(participants) == 0 {
		return []*entity.SessionParticipant{}, nil
	}

	return participants, nil
}

// FindBySessionIDWithContactInfo finds all participants for a session with
// contact info joined in. Returns contact name + whatsapp + avatar so callers
// don't need a separate per-participant lookup.
func (r *PostgresParticipantRepository) FindBySessionIDWithContactInfo(ctx context.Context, sessionID string) ([]*entity.SessionParticipant, error) {
	query := `
		SELECT sp.participant_id, sp.session_id, sp.contact_id, sp.custom_name, sp.custom_whatsapp,
		       sp.share_amount, sp.payment_status, sp.payment_proof_url, sp.rejection_count, sp.rejection_reason,
		       sp.notification_count, sp.last_notification_at, sp.joined_at, sp.updated_at,
		       c.avatar_url as contact_avatar_url,
		       c.name as contact_name,
		       c.whatsapp_number as contact_whatsapp
		FROM session_participants sp
		LEFT JOIN contacts c ON sp.contact_id = c.contact_id AND c.deleted_at IS NULL
		WHERE sp.session_id = $1
		ORDER BY sp.joined_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []*entity.SessionParticipant{}, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var participants []*entity.SessionParticipant
	for rows.Next() {
		var participant entity.SessionParticipant
		var paymentStatusStr string
		err = rows.Scan(
			&participant.ParticipantID,
			&participant.SessionID,
			&participant.ContactID,
			&participant.CustomName,
			&participant.CustomWhatsApp,
			&participant.ShareAmount,
			&paymentStatusStr,
			&participant.PaymentProofURL,
			&participant.RejectionCount,
			&participant.RejectionReason,
			&participant.NotificationCount,
			&participant.LastNotificationAt,
			&participant.JoinedAt,
			&participant.UpdatedAt,
			&participant.ContactAvatarURL,
			&participant.ContactName,
			&participant.ContactWhatsApp,
		)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		participant.PaymentStatus = valueobject.PaymentStatus(paymentStatusStr)
		participants = append(participants, &participant)
	}

	if err = rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	if len(participants) == 0 {
		return []*entity.SessionParticipant{}, nil
	}

	return participants, nil
}

// Update updates a participant
func (r *PostgresParticipantRepository) Update(ctx context.Context, participant *entity.SessionParticipant) error {
	query := `
		UPDATE session_participants
		SET custom_name = $2, custom_whatsapp = $3, share_amount = $4, payment_status = $5, payment_proof_url = $6, rejection_count = $7, rejection_reason = $8, notification_count = $9, last_notification_at = $10, updated_at = $11
		WHERE participant_id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		participant.ParticipantID,
		participant.CustomName,
		participant.CustomWhatsApp,
		participant.ShareAmount,
		participant.PaymentStatus,
		participant.PaymentProofURL,
		participant.RejectionCount,
		participant.RejectionReason,
		participant.NotificationCount,
		participant.LastNotificationAt,
		participant.UpdatedAt,
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

// Delete deletes a participant
func (r *PostgresParticipantRepository) Delete(ctx context.Context, participantID string) error {
	query := `
		DELETE FROM session_participants
		WHERE participant_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, participantID)
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

// DeleteBySessionID deletes all participants for a session
func (r *PostgresParticipantRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	query := `
		DELETE FROM session_participants
		WHERE session_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, sessionID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// UpdateShareAmount updates the share amount for a participant
func (r *PostgresParticipantRepository) UpdateShareAmount(ctx context.Context, participantID string, amount float64) error {
	query := `
		UPDATE session_participants
		SET share_amount = $2, updated_at = NOW()
		WHERE participant_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, participantID, amount)
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

// UpdatePaymentStatus updates the payment status for a participant
func (r *PostgresParticipantRepository) UpdatePaymentStatus(ctx context.Context, participantID string, status string) error {
	query := `
		UPDATE session_participants
		SET payment_status = $2, updated_at = NOW()
		WHERE participant_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, participantID, status)
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

// MarkSubmittedWithProof atomically transitions a participant from 'pending'
// to 'submitted' with the given proof URL. The WHERE clause requires status =
// 'pending', so two concurrent submitters race once in the database and only
// the first one's UPDATE returns rowsAffected == 1.
func (r *PostgresParticipantRepository) MarkSubmittedWithProof(ctx context.Context, participantID, proofURL string) (bool, error) {
	const query = `
		UPDATE session_participants
		SET payment_status = 'submitted', payment_proof_url = $2, updated_at = NOW()
		WHERE participant_id = $1 AND payment_status = 'pending'
	`
	result, err := r.db.ExecContext(ctx, query, participantID, proofURL)
	if err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, domain.NewError(domain.ErrInternalFailure, err)
	}
	return rows == 1, nil
}

// BulkUpdateShareAmounts updates share amounts for multiple participants
func (r *PostgresParticipantRepository) BulkUpdateShareAmounts(ctx context.Context, updates map[string]float64) error {
	if len(updates) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	defer tx.Rollback()

	query := `
		UPDATE session_participants
		SET share_amount = $2, updated_at = NOW()
		WHERE participant_id = $1
	`

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	defer stmt.Close()

	for participantID, amount := range updates {
		if _, err := stmt.ExecContext(ctx, participantID, amount); err != nil {
			return domain.NewError(domain.ErrInternalFailure, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// CountBySessionIDAndStatus counts participants by payment status for a session
func (r *PostgresParticipantRepository) CountBySessionIDAndStatus(ctx context.Context, sessionID string, status string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM session_participants
		WHERE session_id = $1 AND payment_status = $2
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, sessionID, status).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}

	return count, nil
}
