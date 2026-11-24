package admin

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
)

// GetStatusResponse represents WhatsApp gateway status.
//
// IsConnected / GatewayTokenValid report the live gateway view. The
// token_* fields describe the locally stored JWT: has_token says a
// sealed token exists in whatsapp_configs (i.e. the admin registered
// a phone), token_phone/token_expires_at are display-only details
// parsed from the JWT without verifying its signature.
type GetStatusResponse struct {
	IsConnected       bool   `json:"is_connected"`
	GatewayTokenValid bool   `json:"gateway_token_valid"`
	PhoneNumber       string `json:"phone_number"`
	HasToken          bool   `json:"has_token"`
	TokenPhone        string `json:"token_phone,omitempty"`
	TokenExpiresAt    string `json:"token_expires_at,omitempty"`
	LastConnectedAt   string `json:"last_connected_at,omitempty"`
	Message           string `json:"message,omitempty"`
}

// GetStatusUseCase polls the gateway for the live pairing state and
// joins it with the locally stored token metadata. The gateway is the
// only source of truth for "am I paired"; whatsapp_configs only
// answers "did we ever register a phone here".
type GetStatusUseCase struct {
	whatsappSvc *service.WhatsAppService
	configRepo  ports.WhatsAppConfigRepository
}

func NewGetStatusUseCase(whatsappSvc *service.WhatsAppService, configRepo ports.WhatsAppConfigRepository) *GetStatusUseCase {
	return &GetStatusUseCase{
		whatsappSvc: whatsappSvc,
		configRepo:  configRepo,
	}
}

// Execute polls the gateway and returns the live state.
func (uc *GetStatusUseCase) Execute(ctx context.Context) (*GetStatusResponse, error) {
	resp := &GetStatusResponse{}

	// Stored-token metadata first, so it's present even when the
	// gateway itself is unreachable.
	row, rowErr := uc.configRepo.Get(ctx)
	if rowErr == nil && row != nil && row.GatewayToken != "" {
		resp.HasToken = true
		resp.TokenPhone = row.PhoneNumber
		if exp, ok := service.ParseJWTExpiry(row.GatewayToken); ok {
			resp.TokenExpiresAt = exp.Format("2006-01-02T15:04:05Z")
		}
	}

	authenticated, msg, err := uc.whatsappSvc.GetLoginStatus(ctx)
	if err != nil {
		// Distinguish "our JWT is dead" from "can't reach the gateway":
		// 401 means the stored token was rejected → admin should
		// re-register; anything else is infrastructure.
		if errors.Is(err, waga.ErrUnauthorized) {
			resp.IsConnected = false
			resp.GatewayTokenValid = false
			resp.Message = "Gateway rejected the token. Re-register the phone number."
			return resp, nil
		}
		// Reaching the gateway failed entirely. Report "not connected,
		// token not valid" rather than 500'ing — the FE can render
		// a useful "gateway unreachable" state.
		resp.IsConnected = false
		resp.GatewayTokenValid = false
		resp.Message = "Gateway unreachable. Check WHATSAPP_GATEWAY_URL and WHATSAPP_API_KEY."
		return resp, nil
	}

	resp.IsConnected = authenticated
	resp.GatewayTokenValid = true
	resp.Message = msg
	if authenticated {
		resp.Message = "WhatsApp connected"
		resp.PhoneNumber = resp.TokenPhone
	} else if resp.Message == "" {
		resp.Message = "WhatsApp not paired"
	}
	return resp, nil
}
