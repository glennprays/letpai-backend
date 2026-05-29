package admin

import (
	"context"

	"github.com/glennprays/letpai-backend/internal/service"
)

// GetStatusResponse represents WhatsApp gateway status.
//
// Trimmed to the values we can authoritatively report from the gateway
// SDK alone: whether the device is currently paired (IsConnected) and
// whether the operator-configured token successfully reached the
// gateway (GatewayTokenValid -- false implies "couldn't even talk to
// the gateway", surfaced via the call failing rather than the field).
type GetStatusResponse struct {
	IsConnected       bool   `json:"is_connected"`
	GatewayTokenValid bool   `json:"gateway_token_valid"`
	PhoneNumber       string `json:"phone_number"`
	LastConnectedAt   string `json:"last_connected_at,omitempty"`
	Message           string `json:"message,omitempty"`
}

// GetStatusUseCase handles getting WhatsApp gateway status.
//
// As of this revision the use case is a thin wrapper around
// WhatsAppService.GetLoginStatus -- the local whatsapp_configs table
// is no longer consulted. The gateway is the only source of truth
// for "am I paired". When the gateway is unreachable / 4xx / 5xx,
// the SDK error bubbles up and the FE surfaces it.
type GetStatusUseCase struct {
	whatsappSvc *service.WhatsAppService
}

// NewGetStatusUseCase creates a new get status use case
func NewGetStatusUseCase(whatsappSvc *service.WhatsAppService) *GetStatusUseCase {
	return &GetStatusUseCase{whatsappSvc: whatsappSvc}
}

// Execute polls the gateway and returns the live state.
func (uc *GetStatusUseCase) Execute(ctx context.Context) (*GetStatusResponse, error) {
	authenticated, msg, err := uc.whatsappSvc.GetLoginStatus(ctx)
	if err != nil {
		// Reaching the gateway failed entirely. Report "not connected,
		// token not valid" rather than 500'ing — the FE can render
		// a useful "gateway unreachable" state.
		return &GetStatusResponse{
			IsConnected:       false,
			GatewayTokenValid: false,
			Message:           "Gateway unreachable. Check WHATSAPP_GATEWAY_URL and WHATSAPP_API_KEY.",
		}, nil
	}

	resp := &GetStatusResponse{
		IsConnected:       authenticated,
		GatewayTokenValid: true,
		Message:           msg,
	}
	if authenticated {
		resp.Message = "WhatsApp connected"
	} else if resp.Message == "" {
		resp.Message = "WhatsApp not paired"
	}
	return resp, nil
}
