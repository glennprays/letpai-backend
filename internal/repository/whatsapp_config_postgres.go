package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/jmoiron/sqlx"
)

type PostgresWhatsAppConfigRepository struct {
	db *sqlx.DB
}

// NewPostgresWhatsAppConfigRepository creates a new config repository instance
func NewPostgresWhatsAppConfigRepository(db *sqlx.DB) ports.WhatsAppConfigRepository {
	return &PostgresWhatsAppConfigRepository{
		db: db,
	}
}

// Get retrieves the current WhatsApp config (singleton pattern). Returns
// (nil, nil) when no row exists yet — callers expect that to mean
// "gateway not configured" rather than an internal error.
//
// Missing columns from the SELECT (gateway_token, is_connected,
// qr_code_base64, qr_code_expires_at) were the reason the
// GetStatusUseCase always reported `gateway_token_valid=false` and
// `is_connected=false` even after a successful pairing — the writes
// were correct, the read just never sourced them.
func (r *PostgresWhatsAppConfigRepository) Get(ctx context.Context) (*entity.WhatsAppConfig, error) {
	const query = `
		SELECT config_id, phone_number, gateway_token, is_connected, qr_code_base64,
		       qr_code_expires_at, connection_status, last_connected_at,
		       last_disconnected_at, created_at, updated_at
		FROM whatsapp_configs
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var config entity.WhatsAppConfig
	err := r.db.GetContext(ctx, &config, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get whatsapp config: %w", err))
	}
	return &config, nil
}

// CreateOrUpdate creates a new config or updates existing one
func (r *PostgresWhatsAppConfigRepository) CreateOrUpdate(ctx context.Context, config *entity.WhatsAppConfig) error {
	const query = `
		INSERT INTO whatsapp_configs (config_id, phone_number, connection_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (phone_number)
		DO UPDATE SET
			connection_status = EXCLUDED.connection_status,
			updated_at = CURRENT_TIMESTAMP,
			gateway_token = COALESCE(EXCLUDED.gateway_token, whatsapp_configs.gateway_token),
			qr_code_base64 = COALESCE(EXCLUDED.qr_code_base64, whatsapp_configs.qr_code_base64),
			qr_code_expires_at = COALESCE(EXCLUDED.qr_code_expires_at, whatsapp_configs.qr_code_expires_at)
		WHERE whatsapp_configs.phone_number = excluded.phone_number
		RETURNING *
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ConfigID, config.PhoneNumber, config.ConnectionStatus,
		config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to create/update whatsapp config: %w", err))
	}
	return nil
}

// UpdateToken updates the gateway token
func (r *PostgresWhatsAppConfigRepository) UpdateToken(ctx context.Context, phone_number, token string) error {
	const query = `
		UPDATE whatsapp_configs
		SET gateway_token = $1,
		    updated_at = CURRENT_TIMESTAMP
		WHERE phone_number = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, token, phone_number)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update gateway token: %w", err))
	}
	return nil
}

// UpdateConnectionStatus updates the connection status
func (r *PostgresWhatsAppConfigRepository) UpdateConnectionStatus(ctx context.Context, phone_number, status string) error {
	const query = `
		UPDATE whatsapp_configs
		SET connection_status = $1,
		    is_connected = CASE WHEN $1 = 'connected' THEN TRUE ELSE FALSE END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE phone_number = $2 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, status, phone_number)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update connection status: %w", err))
	}
	return nil
}

// UpdateQRCode updates the cached QR code
func (r *PostgresWhatsAppConfigRepository) UpdateQRCode(ctx context.Context, phone_number, qrCodeBase64 string, expiresAt time.Time) error {
	const query = `
		UPDATE whatsapp_configs
		SET qr_code_base64 = $1,
		    qr_code_expires_at = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE phone_number = $3 AND deleted_at IS NULL
	`

	_, err := r.db.ExecContext(ctx, query, qrCodeBase64, expiresAt, phone_number)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update qr code: %w", err))
	}
	return nil
}

// Delete removes the config
func (r *PostgresWhatsAppConfigRepository) Delete(ctx context.Context, configID string) error {
	const query = `
		DELETE FROM whatsapp_configs
		WHERE config_id = $1
	`

	_, err := r.db.ExecContext(ctx, query, configID)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to delete whatsapp config: %w", err))
	}
	return nil
}

// EnsureSingletonRow inserts a placeholder config row when none exists.
// The phone_number column is UNIQUE NOT NULL, so an empty string acts
// as a stable singleton key; ON CONFLICT makes the call idempotent.
//
// Callers in the QR / status / disconnect paths invoke this once before
// the singleton UPDATEs so those UPDATEs always have a target. The
// row's phone_number is populated later — either via the operator
// pasting a gateway token (existing UpdateToken path) or, eventually,
// auto-discovered from an inbound webhook.
func (r *PostgresWhatsAppConfigRepository) EnsureSingletonRow(ctx context.Context) error {
	const query = `
		INSERT INTO whatsapp_configs (config_id, phone_number, connection_status, created_at, updated_at)
		VALUES (gen_random_uuid(), '', 'disconnected', NOW(), NOW())
		ON CONFLICT (phone_number) DO NOTHING
	`
	if _, err := r.db.ExecContext(ctx, query); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to ensure singleton config row: %w", err))
	}
	return nil
}

// UpdateQRCodeSingleton writes the QR + expiry to the most recently
// updated, non-deleted config row. Uses a CTE because Postgres doesn't
// accept ORDER BY / LIMIT directly on a bare UPDATE. No-op when no
// row exists — callers should invoke EnsureSingletonRow first.
func (r *PostgresWhatsAppConfigRepository) UpdateQRCodeSingleton(ctx context.Context, qrCodeBase64 string, expiresAt time.Time) error {
	const query = `
		WITH target AS (
			SELECT config_id FROM whatsapp_configs
			WHERE deleted_at IS NULL
			ORDER BY updated_at DESC
			LIMIT 1
		)
		UPDATE whatsapp_configs
		SET qr_code_base64 = $1,
		    qr_code_expires_at = $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE config_id IN (SELECT config_id FROM target)
	`
	if _, err := r.db.ExecContext(ctx, query, qrCodeBase64, expiresAt); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update qr code (singleton): %w", err))
	}
	return nil
}

// UpdateConnectionStatusSingleton mirrors UpdateConnectionStatus's
// effects (connection_status, is_connected, last_(dis)connected_at)
// but targets the singleton row rather than matching by phone. Used
// by GetQRCode -> "qr_pending" and Logout / failed-poll -> "disconnected"
// when we don't have a phone label to match against.
func (r *PostgresWhatsAppConfigRepository) UpdateConnectionStatusSingleton(ctx context.Context, status string) error {
	const query = `
		WITH target AS (
			SELECT config_id FROM whatsapp_configs
			WHERE deleted_at IS NULL
			ORDER BY updated_at DESC
			LIMIT 1
		)
		UPDATE whatsapp_configs
		SET connection_status = $1,
		    is_connected = CASE WHEN $1 = 'connected' THEN TRUE ELSE FALSE END,
		    last_connected_at = CASE WHEN $1 = 'connected' THEN NOW() ELSE last_connected_at END,
		    last_disconnected_at = CASE WHEN $1 = 'disconnected' THEN NOW() ELSE last_disconnected_at END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE config_id IN (SELECT config_id FROM target)
	`
	if _, err := r.db.ExecContext(ctx, query, status); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update connection status (singleton): %w", err))
	}
	return nil
}
