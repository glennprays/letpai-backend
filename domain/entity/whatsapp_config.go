package entity

import (
	"time"

	"github.com/google/uuid"
)

// WhatsAppConfig represents WhatsApp gateway connection configuration.
//
// db tags are explicit because sqlx's default mapper (strings.ToLower)
// would turn PhoneNumber into "phonenumber", GatewayToken into
// "gatewaytoken", etc. — none of which match the snake_case columns.
// Without these tags GetContext silently leaves every field at its
// zero value, which is why the Status badge could never flip from
// "Not paired" even when the row was correctly written by the writes
// (which use positional $1, $2 placeholders and aren't affected).
type WhatsAppConfig struct {
	ConfigID           uuid.UUID  `json:"config_id"                     db:"config_id"`
	PhoneNumber        string     `json:"phone_number"                  db:"phone_number"`
	GatewayToken       string     `json:"-"                             db:"gateway_token"` // JWT token from gateway, not exposed
	IsConnected        bool       `json:"is_connected"                  db:"is_connected"`
	QRCodeBase64       string     `json:"-"                             db:"qr_code_base64"` // Cached QR code, not exposed
	QRCodeExpiresAt    *time.Time `json:"qr_code_expires_at,omitempty"  db:"qr_code_expires_at"`
	ConnectionStatus   string     `json:"connection_status"             db:"connection_status"`
	LastConnectedAt    *time.Time `json:"last_connected_at,omitempty"   db:"last_connected_at"`
	LastDisconnectedAt *time.Time `json:"last_disconnected_at,omitempty" db:"last_disconnected_at"`
	CreatedAt          time.Time  `json:"created_at"                    db:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"                    db:"updated_at"`
}

const (
	ConnectionStatusConnected    = "connected"
	ConnectionStatusDisconnected = "disconnected"
	ConnectionStatusQRPending    = "qr_pending"
	ConnectionStatusFailed       = "failed"
)

// NewWhatsAppConfig creates a new config instance
func NewWhatsAppConfig(phoneNumber string) *WhatsAppConfig {
	now := time.Now()
	return &WhatsAppConfig{
		ConfigID:         uuid.New(),
		PhoneNumber:      phoneNumber,
		ConnectionStatus: ConnectionStatusDisconnected,
		IsConnected:      false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// UpdateToken updates the gateway token
func (wc *WhatsAppConfig) UpdateToken(token string) {
	wc.GatewayToken = token
	wc.UpdatedAt = time.Now()
}

// UpdateConnectionStatus updates the connection status and related timestamps
func (wc *WhatsAppConfig) UpdateConnectionStatus(status string) {
	wc.ConnectionStatus = status
	now := time.Now()
	wc.UpdatedAt = now

	if status == ConnectionStatusConnected {
		wc.IsConnected = true
		wc.LastConnectedAt = &now
		wc.LastDisconnectedAt = nil
		wc.QRCodeExpiresAt = nil
	} else if status == ConnectionStatusDisconnected {
		wc.IsConnected = false
		wc.LastDisconnectedAt = &now
	}
}
