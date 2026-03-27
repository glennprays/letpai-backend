package ports

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// WhatsAppConfigRepository defines the interface for WhatsApp config operations
type WhatsAppConfigRepository interface {
	// Get retrieves the current WhatsApp config (singleton pattern)
	Get(ctx context.Context) (*entity.WhatsAppConfig, error)

	// CreateOrUpdate creates a new config or updates existing one for the phone number
	CreateOrUpdate(ctx context.Context, config *entity.WhatsAppConfig) error

	// UpdateToken updates the gateway token
	UpdateToken(ctx context.Context, phone_number, token string) error

	// UpdateConnectionStatus updates the connection status
	UpdateConnectionStatus(ctx context.Context, phone_number, status string) error

	// UpdateQRCode updates the cached QR code
	UpdateQRCode(ctx context.Context, phone_number, qrCodeBase64 string, expiresAt time.Time) error

	// Delete removes the config
	Delete(ctx context.Context, configID string) error
}
