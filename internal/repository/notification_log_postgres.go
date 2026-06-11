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

// PostgresNotificationLogRepository implements NotificationLogRepository using PostgreSQL
type PostgresNotificationLogRepository struct {
	db *sqlx.DB
}

// NewPostgresNotificationLogRepository creates a new PostgreSQL notification log repository
func NewPostgresNotificationLogRepository(db *sqlx.DB) ports.NotificationLogRepository {
	return &PostgresNotificationLogRepository{db: db}
}

// Create creates a new notification log entry
func (r *PostgresNotificationLogRepository) Create(ctx context.Context, log *entity.NotificationLog) error {
	query := `
		INSERT INTO notification_logs
			(log_id, participant_id, notification_type, whatsapp_message_id,
			 message_content, sent_at, status, error_message, phone)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		log.LogID,
		log.ParticipantID,
		log.NotificationType,
		log.WhatsAppMessageID,
		log.MessageContent,
		log.SentAt,
		log.Status,
		log.ErrorMessage,
		log.Phone,
	)

	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// FindByID finds a notification log by ID
func (r *PostgresNotificationLogRepository) FindByID(ctx context.Context, logID string) (*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE log_id = $1
	`

	row := r.db.QueryRowxContext(ctx, query, logID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var log entity.NotificationLog
	err := row.StructScan(&log)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &log, nil
}

// FindByParticipantID finds all notification logs for a participant
func (r *PostgresNotificationLogRepository) FindByParticipantID(ctx context.Context, participantID string) ([]*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE participant_id = $1
		ORDER BY sent_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, participantID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var logs []*entity.NotificationLog
	for rows.Next() {
		var log entity.NotificationLog
		if err := rows.StructScan(&log); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return logs, nil
}

// FindByParticipantIDWithType finds notification logs for a participant filtered by type
func (r *PostgresNotificationLogRepository) FindByParticipantIDWithType(ctx context.Context, participantID string, notificationType string) ([]*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE participant_id = $1 AND notification_type = $2
		ORDER BY sent_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, participantID, notificationType)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var logs []*entity.NotificationLog
	for rows.Next() {
		var log entity.NotificationLog
		if err := rows.StructScan(&log); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return logs, nil
}

// FindLatestByParticipantID finds the latest notification log for a participant
func (r *PostgresNotificationLogRepository) FindLatestByParticipantID(ctx context.Context, participantID string) (*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE participant_id = $1
		ORDER BY sent_at DESC
		LIMIT 1
	`

	row := r.db.QueryRowxContext(ctx, query, participantID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var log entity.NotificationLog
	err := row.StructScan(&log)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &log, nil
}

// FindLatestByParticipantIDWithType finds the latest notification log for a participant filtered by type
func (r *PostgresNotificationLogRepository) FindLatestByParticipantIDWithType(ctx context.Context, participantID string, notificationType string) (*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE participant_id = $1 AND notification_type = $2
		ORDER BY sent_at DESC
		LIMIT 1
	`

	row := r.db.QueryRowxContext(ctx, query, participantID, notificationType)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var log entity.NotificationLog
	err := row.StructScan(&log)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &log, nil
}

// FindLatestPerParticipantBySessionID returns one row per participant
// (the most recent log), keyed by participant_id. The DISTINCT ON +
// ORDER BY (participant_id, sent_at DESC) shape relies on the
// composite index added in migration 000022 — without it Postgres
// would scan every log row for the session. Skips participants with
// no logs (caller renders them as "Not sent yet").
func (r *PostgresNotificationLogRepository) FindLatestPerParticipantBySessionID(ctx context.Context, sessionID string) (map[uuid.UUID]*entity.NotificationLog, error) {
	const q = `
		SELECT DISTINCT ON (nl.participant_id)
		       nl.log_id, nl.participant_id, nl.notification_type,
		       nl.whatsapp_message_id, nl.message_content, nl.sent_at,
		       nl.status, nl.error_message
		  FROM notification_logs nl
		  JOIN session_participants sp ON sp.participant_id = nl.participant_id
		 WHERE sp.session_id = $1
		 ORDER BY nl.participant_id, nl.sent_at DESC
	`
	rows, err := r.db.QueryxContext(ctx, q, sessionID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]*entity.NotificationLog)
	for rows.Next() {
		var log entity.NotificationLog
		if err := rows.StructScan(&log); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		copy := log
		out[log.ParticipantID] = &copy
	}
	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return out, nil
}

// FindBySessionID finds all notification logs for all participants in a session
func (r *PostgresNotificationLogRepository) FindBySessionID(ctx context.Context, sessionID string) ([]*entity.NotificationLog, error) {
	query := `
		SELECT nl.log_id, nl.participant_id, nl.notification_type, nl.whatsapp_message_id, nl.message_content, nl.sent_at, nl.status, nl.error_message
		FROM notification_logs nl
		INNER JOIN session_participants sp ON nl.participant_id = sp.participant_id
		WHERE sp.session_id = $1
		ORDER BY nl.sent_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, sessionID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var logs []*entity.NotificationLog
	for rows.Next() {
		var log entity.NotificationLog
		if err := rows.StructScan(&log); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return logs, nil
}

// FindFailed finds all failed notifications
func (r *PostgresNotificationLogRepository) FindFailed(ctx context.Context) ([]*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE status = 'failed'
		ORDER BY sent_at DESC
	`

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	defer rows.Close()

	var logs []*entity.NotificationLog
	for rows.Next() {
		var log entity.NotificationLog
		if err := rows.StructScan(&log); err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, err)
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return logs, nil
}

// UpdateStatus updates the status of a notification log
func (r *PostgresNotificationLogRepository) UpdateStatus(ctx context.Context, logID string, status string) error {
	query := `
		UPDATE notification_logs
		SET status = $2
		WHERE log_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, logID, status)
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

// UpdateStatusWithMessageID updates the status and WhatsApp message ID of a notification log
func (r *PostgresNotificationLogRepository) UpdateStatusWithMessageID(ctx context.Context, logID string, status string, whatsappMessageID string) error {
	query := `
		UPDATE notification_logs
		SET status = $2, whatsapp_message_id = $3
		WHERE log_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, logID, status, whatsappMessageID)
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

// UpdateStatusWithError updates the status and error message of a notification log
func (r *PostgresNotificationLogRepository) UpdateStatusWithError(ctx context.Context, logID string, status string, errorMessage string) error {
	query := `
		UPDATE notification_logs
		SET status = $2, error_message = $3
		WHERE log_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, logID, status, errorMessage)
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

// UpdateStatusGuarded advances status only if the row is still at
// expectedCurrent (optimistic compare-and-swap). Zero rows affected means
// another webhook delivery already advanced it — that is a successful no-op,
// not an error, which is what makes concurrent/duplicate webhook deliveries
// safe. A real DB error is surfaced so the caller can return 5xx and let the
// gateway retry.
func (r *PostgresNotificationLogRepository) UpdateStatusGuarded(ctx context.Context, logID, newStatus, expectedCurrent string, errorMessage *string) error {
	query := `
		UPDATE notification_logs
		   SET status = $2,
		       error_message = COALESCE($4, error_message),
		       updated_at = CURRENT_TIMESTAMP
		 WHERE log_id = $1 AND status = $3
	`

	if _, err := r.db.ExecContext(ctx, query, logID, newStatus, expectedCurrent, errorMessage); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}

// CountByParticipantIDAndType counts notification logs for a participant filtered by type
func (r *PostgresNotificationLogRepository) CountByParticipantIDAndType(ctx context.Context, participantID string, notificationType string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notification_logs
		WHERE participant_id = $1 AND notification_type = $2
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, participantID, notificationType).Scan(&count)
	if err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}

	return count, nil
}

// CountByParticipantIDAndTypeAfterDate counts notification logs for a participant filtered by type and after a date
func (r *PostgresNotificationLogRepository) CountByParticipantIDAndTypeAfterDate(ctx context.Context, participantID string, notificationType string, afterDate string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notification_logs
		WHERE participant_id = $1 AND notification_type = $2 AND sent_at > $3
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, participantID, notificationType, afterDate).Scan(&count)
	if err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}

	return count, nil
}

// Delete deletes a notification log
func (r *PostgresNotificationLogRepository) Delete(ctx context.Context, logID string) error {
	query := `
		DELETE FROM notification_logs
		WHERE log_id = $1
	`

	result, err := r.db.ExecContext(ctx, query, logID)
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

// FindByWhatsAppMessageID finds a notification log by WhatsApp message ID
func (r *PostgresNotificationLogRepository) FindByWhatsAppMessageID(ctx context.Context, whatsappMessageID string) (*entity.NotificationLog, error) {
	query := `
		SELECT log_id, participant_id, notification_type, whatsapp_message_id, message_content, sent_at, status, error_message
		FROM notification_logs
		WHERE whatsapp_message_id = $1
	`

	row := r.db.QueryRowxContext(ctx, query, whatsappMessageID)
	if row.Err() != nil {
		if errors.Is(row.Err(), sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, row.Err())
	}

	var log entity.NotificationLog
	err := row.StructScan(&log)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.NewError(domain.ErrNotFound, nil)
		}
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	return &log, nil
}

// DeleteByParticipantID deletes all notification logs for a participant
func (r *PostgresNotificationLogRepository) DeleteByParticipantID(ctx context.Context, participantID string) error {
	query := `
		DELETE FROM notification_logs
		WHERE participant_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, participantID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}

	return nil
}
