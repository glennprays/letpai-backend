package admin

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// GetStatusResponse represents WhatsApp gateway status
type GetStatusResponse struct {
	IsConnected       bool   `json:"is_connected"`
	GatewayTokenValid bool   `json:"gateway_token_valid"`
	PhoneNumber       string `json:"phone_number"`
	LastConnectedAt   string `json:"last_connected_at,omitempty"`
	Message           string `json:"message,omitempty"`
}

// GetStatusUseCase handles getting WhatsApp gateway status
//
// As of this revision the use case also polls the gateway through
// WhatsAppService.GetLoginStatus so the badge on /admin/whatsapp can
// flip from "qr_pending" to "connected" without anyone else having
// to refresh the underlying row. Gateway failures are swallowed —
// /admin/status must never 500 just because the gateway is down.
type GetStatusUseCase struct {
	whatsappConfigRepo ports.WhatsAppConfigRepository
	whatsappSvc        *service.WhatsAppService
}

// NewGetStatusUseCase creates a new get status use case
func NewGetStatusUseCase(
	whatsappConfigRepo ports.WhatsAppConfigRepository,
	whatsappSvc *service.WhatsAppService,
) *GetStatusUseCase {
	return &GetStatusUseCase{
		whatsappConfigRepo: whatsappConfigRepo,
		whatsappSvc:        whatsappSvc,
	}
}

// Execute retrieves WhatsApp gateway status
func (uc *GetStatusUseCase) Execute(ctx context.Context) (*GetStatusResponse, error) {
	config, err := uc.whatsappConfigRepo.Get(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Live poll the gateway so the local row reflects reality. We
	// only poll when we have something to ask about — no phone, no
	// gateway token, no point. GetLoginStatus persists the new
	// connection_status itself, so we re-read after.
	if config != nil && config.PhoneNumber != "" && config.GatewayToken != "" {
		if _, _, pollErr := uc.whatsappSvc.GetLoginStatus(ctx); pollErr == nil {
			refreshed, refreshErr := uc.whatsappConfigRepo.Get(ctx)
			if refreshErr == nil && refreshed != nil {
				config = refreshed
			}
		}
	}

	response := &GetStatusResponse{
		IsConnected:       false,
		GatewayTokenValid: false,
		PhoneNumber:       "",
		Message:           "WhatsApp not connected",
	}

	if config != nil {
		response.PhoneNumber = config.PhoneNumber
		response.IsConnected = config.ConnectionStatus == "connected"
		response.GatewayTokenValid = config.GatewayToken != ""
		response.LastConnectedAt = config.LastConnectedAt.Format(time.RFC3339)

		if config.ConnectionStatus == "connected" {
			response.Message = "WhatsApp connected"
		} else if config.ConnectionStatus == "disconnected" {
			response.Message = "WhatsApp disconnected"
		} else {
			response.Message = "WhatsApp not paired"
		}
	}

	return response, nil
}
