package admin

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
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
type GetStatusUseCase struct {
	whatsappConfigRepo ports.WhatsAppConfigRepository
}

// NewGetStatusUseCase creates a new get status use case
func NewGetStatusUseCase(whatsappConfigRepo ports.WhatsAppConfigRepository) *GetStatusUseCase {
	return &GetStatusUseCase{
		whatsappConfigRepo: whatsappConfigRepo,
	}
}

// Execute retrieves WhatsApp gateway status
func (uc *GetStatusUseCase) Execute(ctx context.Context) (*GetStatusResponse, error) {
	config, err := uc.whatsappConfigRepo.Get(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
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
