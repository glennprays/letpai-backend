package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// GetQRCodeRequest is intentionally empty.
//
// WAGA's JWT (configured via WHATSAPP_API_KEY) is phone-scoped at
// registration time, so the gateway derives the phone from the bearer
// token on every subsequent SDK call. Asking the operator to retype
// it on /admin/whatsapp was app-layer ceremony with no functional
// effect — dropped to match the actual protocol.
type GetQRCodeRequest struct{}

// GetQRCodeResponse represents QR code response
type GetQRCodeResponse struct {
	QRCode    string `json:"qr_code"`
	ExpiresAt string `json:"qr_code_expires_at"`
}

// GetQRCodeUseCase handles generating QR code for WhatsApp pairing
type GetQRCodeUseCase struct {
	whatsappSvc *service.WhatsAppService
}

// NewGetQRCodeUseCase creates a new get QR code use case
func NewGetQRCodeUseCase(whatsappSvc *service.WhatsAppService) *GetQRCodeUseCase {
	return &GetQRCodeUseCase{
		whatsappSvc: whatsappSvc,
	}
}

// Execute generates QR code for WhatsApp pairing
func (uc *GetQRCodeUseCase) Execute(ctx context.Context, _ *GetQRCodeRequest) (*GetQRCodeResponse, error) {
	qrCode, expiresAt, err := uc.whatsappSvc.GetQRCode(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to generate QR code: %w", err))
	}

	return &GetQRCodeResponse{
		QRCode:    qrCode,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

// LogoutUseCase handles admin logout
type LogoutUseCase struct {
	adminRepo ports.AdminRepository
}

// NewLogoutUseCase creates a new logout use case
func NewLogoutUseCase(adminRepo ports.AdminRepository) *LogoutUseCase {
	return &LogoutUseCase{
		adminRepo: adminRepo,
	}
}

// Execute handles admin logout
func (uc *LogoutUseCase) Execute(ctx context.Context, adminID string) error {
	// TODO: Implement session invalidation if needed
	// For now, we just acknowledge the logout
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	admin.UpdateLastLogin()
	if err := uc.adminRepo.Update(ctx, admin); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update admin: %w", err))
	}

	return nil
}

// UpdateConfigRequest represents update config request
type UpdateConfigRequest struct {
	PhoneNumber string `json:"phone_number" validate:"omitempty,len=13,max=20"`
	Token       string `json:"token" validate:"omitempty"`
}

// UpdateConfigUseCase handles updating WhatsApp configuration
type UpdateConfigUseCase struct {
	whatsappConfigRepo ports.WhatsAppConfigRepository
}

// NewUpdateConfigUseCase creates a new update config use case
func NewUpdateConfigUseCase(whatsappConfigRepo ports.WhatsAppConfigRepository) *UpdateConfigUseCase {
	return &UpdateConfigUseCase{
		whatsappConfigRepo: whatsappConfigRepo,
	}
}

// Execute updates WhatsApp configuration
func (uc *UpdateConfigUseCase) Execute(ctx context.Context, req *UpdateConfigRequest) error {
	if req.PhoneNumber == "" && req.Token == "" {
		return domain.NewError(domain.ErrBadRequest, errors.New("at least one field is required"))
	}

	config, err := uc.whatsappConfigRepo.Get(ctx)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	if config == nil {
		return domain.NewError(domain.ErrNotFound, errors.New("no WhatsApp configuration found"))
	}

	// Update token if provided
	if req.Token != "" {
		if err := uc.whatsappConfigRepo.UpdateToken(ctx, config.PhoneNumber, req.Token); err != nil {
			return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update token: %w", err))
		}
	}

	return nil
}
