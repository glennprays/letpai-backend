package entity

import (
	"time"

	"github.com/google/uuid"
)

// WhatsAppConfig represents WhatsApp gateway connection configuration
type WhatsAppConfig struct {
	ConfigID           uuid.UUID  `json:"config_id"`
	PhoneNumber        string     `json:"phone_number"`
	GatewayToken       string     `json:"-"` // JWT token from gateway, not exposed
	IsConnected        bool       `json:"is_connected"`
	QRCodeBase64       string     `json:"-"` // Cached QR code, not exposed
	QRCodeExpiresAt    *time.Time `json:"qr_code_expires_at,omitempty"`
	ConnectionStatus   string     `json:"connection_status"`
	LastConnectedAt    *time.Time `json:"last_connected_at,omitempty"`
	LastDisconnectedAt *time.Time `json:"last_disconnected_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
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
