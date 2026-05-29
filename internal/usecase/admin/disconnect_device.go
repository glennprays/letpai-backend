package admin

import (
	"context"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/internal/service"
)

// DisconnectDeviceResult tells the FE the new connection state so the
// status badge can update inline. status is one of:
// connected | disconnected | qr_pending | failed (see entity.WhatsAppConfig).
type DisconnectDeviceResult struct {
	Status  string `json:"connection_status"`
	Message string `json:"message"`
}

// DisconnectDeviceUseCase wraps WhatsAppService.Logout so admins can
// drop the gateway pairing from the admin panel without bouncing
// through the gateway dashboard.
//
// This is intentionally a thin wrapper — the service already handles
// the local row flip (connection_status = disconnected, last_disconnected_at).
type DisconnectDeviceUseCase struct {
	whatsappSvc *service.WhatsAppService
}

// NewDisconnectDeviceUseCase wires the use case.
func NewDisconnectDeviceUseCase(whatsappSvc *service.WhatsAppService) *DisconnectDeviceUseCase {
	return &DisconnectDeviceUseCase{whatsappSvc: whatsappSvc}
}

// Execute disconnects the paired device.
func (uc *DisconnectDeviceUseCase) Execute(ctx context.Context) (*DisconnectDeviceResult, error) {
	if err := uc.whatsappSvc.Logout(ctx); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to disconnect device: %w", err))
	}
	return &DisconnectDeviceResult{
		Status:  "disconnected",
		Message: "Device disconnected from gateway",
	}, nil
}
