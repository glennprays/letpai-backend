package ports

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
)

// WhatsAppConfigRepository defines the interface for WhatsApp config operations
type WhatsAppConfigRepository interface {
	// Get retrieves the current WhatsApp config (singleton pattern). Returns
	// (nil, nil) when no row exists yet.
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

	// EnsureSingletonRow inserts a placeholder config row if none
	// exists; idempotent. Used by the QR / status / disconnect flows
	// before they call the *Singleton UPDATEs.
	EnsureSingletonRow(ctx context.Context) error

	// UpdateQRCodeSingleton writes the QR + expiry to whichever
	// non-deleted config row is current. Used when we don't yet know
	// (or don't need to know) the phone number that the gateway JWT
	// is bound to.
	UpdateQRCodeSingleton(ctx context.Context, qrCodeBase64 string, expiresAt time.Time) error

	// UpdateConnectionStatusSingleton mirrors UpdateConnectionStatus
	// but targets the singleton row. Used by Logout and the QR flow.
	UpdateConnectionStatusSingleton(ctx context.Context, status string) error
}
