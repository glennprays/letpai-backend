package admin

type GetStatusResponse struct {
	IsConnected       bool   `json:"is_connected"`
	GatewayTokenValid bool   `json:"gateway_token_valid"`
	PhoneNumber       string `json:"phone_number"`
	LastConnectedAt   string `json:"last_connected_at,omitempty"`
	QRCodeBase64      string `json:"qr_code,omitempty"`
	QRCodeExpiresAt   string `json:"qr_code_expires_at,omitempty"`
	Message           string `json:"message,omitempty"`
}

type LoginResponse struct {
	AdminID        string `json:"admin_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	Token          string `json:"token"`
}

type QRCodeResponse struct {
	QRCodeBase64 string `json:"qr_code"`
	ExpiresAt    string `json:"qr_code_expires_at"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

type ConfigResponse struct {
	PhoneNumber string `json:"phone_number"`
}
