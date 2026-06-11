package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// outboxColumns is the shared SELECT/RETURNING projection for the worker
// queue methods, including the outbox bookkeeping columns added in 000030.
const outboxColumns = `log_id, participant_id, notification_type, whatsapp_message_id,
	message_content, sent_at, status, error_message, phone,
	attempts, max_attempts, next_attempt_at, locked_at, locked_by, updated_at`

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

// ClaimPending atomically claims up to limit due rows for this worker. It
// flips them to 'sending' and stamps the lock, using FOR UPDATE SKIP LOCKED so
// multiple worker instances never grab the same row. Returns the claimed rows.
func (r *PostgresNotificationLogRepository) ClaimPending(ctx context.Context, workerID string, limit int) ([]*entity.NotificationLog, error) {
	query := `
		UPDATE notification_logs
		   SET status = 'sending', locked_at = CURRENT_TIMESTAMP, locked_by = $1, updated_at = CURRENT_TIMESTAMP
		 WHERE log_id IN (
		     SELECT log_id FROM notification_logs
		      WHERE status IN ('pending','queued','failed')
		        AND next_attempt_at <= CURRENT_TIMESTAMP
		      ORDER BY next_attempt_at
		      LIMIT $2
		      FOR UPDATE SKIP LOCKED
		 )
		 RETURNING ` + outboxColumns

	rows, err := r.db.QueryxContext(ctx, query, workerID, limit)
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
		logCopy := log
		logs = append(logs, &logCopy)
	}
	if err := rows.Err(); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	return logs, nil
}

// MarkSent records a successful send: status 'sent' + the gateway message id,
// clearing the worker lock.
func (r *PostgresNotificationLogRepository) MarkSent(ctx context.Context, logID, whatsappMessageID string) error {
	query := `
		UPDATE notification_logs
		   SET status = 'sent', whatsapp_message_id = $2,
		       locked_at = NULL, locked_by = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE log_id = $1
	`
	if _, err := r.db.ExecContext(ctx, query, logID, whatsappMessageID); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// MarkRetry records a failed attempt that will be retried: status 'failed',
// bumped attempts, the scheduled next_attempt_at, and the error, clearing the
// lock so the row is claimable again once due.
func (r *PostgresNotificationLogRepository) MarkRetry(ctx context.Context, logID string, attempts int, nextAttemptAt time.Time, errMsg string) error {
	query := `
		UPDATE notification_logs
		   SET status = 'failed', attempts = $2, next_attempt_at = $3, error_message = $4,
		       locked_at = NULL, locked_by = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE log_id = $1
	`
	if _, err := r.db.ExecContext(ctx, query, logID, attempts, nextAttemptAt, errMsg); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// MarkDead records terminal failure after retries are exhausted: status
// 'dead', clearing the lock so it is never claimed again.
func (r *PostgresNotificationLogRepository) MarkDead(ctx context.Context, logID string, attempts int, errMsg string) error {
	query := `
		UPDATE notification_logs
		   SET status = 'dead', attempts = $2, error_message = $3,
		       locked_at = NULL, locked_by = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE log_id = $1
	`
	if _, err := r.db.ExecContext(ctx, query, logID, attempts, errMsg); err != nil {
		return domain.NewError(domain.ErrInternalFailure, err)
	}
	return nil
}

// RequeueStuck returns rows that have been stuck in 'sending' since before
// stuckBefore back to 'pending' so a crashed/restarted worker doesn't leave
// them locked forever. Returns the number of rows requeued.
func (r *PostgresNotificationLogRepository) RequeueStuck(ctx context.Context, stuckBefore time.Time) (int64, error) {
	query := `
		UPDATE notification_logs
		   SET status = 'pending', locked_at = NULL, locked_by = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE status = 'sending' AND locked_at < $1
	`
	res, err := r.db.ExecContext(ctx, query, stuckBefore)
	if err != nil {
		return 0, domain.NewError(domain.ErrInternalFailure, err)
	}
	n, _ := res.RowsAffected()
	return n, nil
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
