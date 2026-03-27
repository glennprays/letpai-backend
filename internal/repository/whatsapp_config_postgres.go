package repository

import (
	"context"
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

// Get retrieves the current WhatsApp config (singleton pattern)
func (r *PostgresWhatsAppConfigRepository) Get(ctx context.Context) (*entity.WhatsAppConfig, error) {
	const query = `
		SELECT config_id, phone_number, connection_status, last_connected_at,
		       last_disconnected_at, created_at, updated_at
		FROM whatsapp_configs
		WHERE deleted_at IS NULL
		ORDER BY updated_at DESC
		LIMIT 1
	`

	var config entity.WhatsAppConfig
	err := r.db.GetContext(ctx, &config, query)
	if err != nil {
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
