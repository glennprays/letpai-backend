package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

type GetStatusUseCase struct {
	configRepo ports.WhatsAppConfigRepository
}

func NewGetStatusUseCase(configRepo ports.WhatsAppConfigRepository) *GetStatusUseCase {
	return &GetStatusUseCase{
		configRepo: configRepo,
	}
}

type GetStatusResponse struct {
	IsConnected       bool   `json:"is_connected"`
	GatewayTokenValid bool  `json:"gateway_token_valid"`
	PhoneNumber      string `json:"phone_number"`
	LastConnectedAt  *string `json:"last_connected_at,omitempty"`
	QRCodeBase64     string `json:"qr_code,omitempty"`
}

type InitiateLoginUseCase struct {
	configRepo   ports.WhatsAppConfigRepository
	whatsappSvc  *service.WhatsAppService
}

func NewInitiateLoginUseCase(configRepo ports.WhatsAppConfigRepository, whatsappSvc *service.WhatsAppService) *InitiateLoginUseCase {
	return &InitiateLoginUseCase{
		configRepo:   configRepo,
		whatsappSvc: whatsappSvc,
	}
}

type GetQRCodeUseCase struct {
	configRepo ports.WhatsAppConfigRepository
	whatsappSvc *service.WhatsAppService
}

func NewGetQRCodeUseCase(configRepo ports.WhatsAppConfigRepository, whatsappSvc *service.WhatsAppService) *GetQRCodeUseCase {
	return &GetQRCodeUseCase{
		configRepo:   configRepo,
		whatsappSvc: whatsappSvc,
	}
}

type LoginResponse struct {
	GatewayToken string `json:"gateway_token"`
	Message     string `json:"message"`
}

type QRCodeResponse struct {
	QRCodeBase64 string `json:"qr_code"`
	ExpiresAt     string `json:"expires_at"`
}

type LogoutResponse struct {
	Message string `json:"message"`
}

func (uc *GetStatusUseCase) Execute(ctx context.Context) (*GetStatusResponse, error) {
	config, err := uc.configRepo.Get(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	gatewayTokenValid := config != nil && config.GatewayToken != ""

	return &GetStatusResponse{
		IsConnected:       config != nil && config.IsConnected,
		GatewayTokenValid: gatewayTokenValid,
		PhoneNumber:      config.PhoneNumber,
		LastConnectedAt: formatTimePtr(config.LastConnectedAt),
		QRCodeBase64:     formatQRBase64Ptr(config.QRCodeBase64),
	}, nil
}

func (uc *InitiateLoginUseCase) Execute(ctx context.Context) (*LoginResponse, error) {
	qrResp, err := uc.whatsappSvc.GetQRCode(ctx, "json")
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	err = uc.configRepo.UpdateQRCode(ctx, "", qrResp.QRCode, time.Now().Add(time.Duration(qrResp.ExpiresIn)*time.Second)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update QR code: %w", err))
	}

	return &LoginResponse{
		GatewayToken: qrResp.QRCode,
		Message:     "QR login initiated",
	}, nil
}

func (uc *GetQRCodeUseCase) Execute(ctx context.Context) (*QRCodeResponse, error) {
	config, err := uc.configRepo.Get(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	qrResp, err := uc.whatsappSvc.GetQRCode(ctx, "json")
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	err = uc.configRepo.UpdateQRCode(ctx, "", qrResp.QRCode, time.Now().Add(time.Duration(qrResp.ExpiresIn)*time.Second)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update QR code: %w", err))
	}

	return &QRCodeResponse{
		QRCodeBase64: qrResp.QRCode,
		ExpiresAt:     time.Now().Add(time.Duration(qrResp.ExpiresIn)*time.Second).Format(time.RFC3339),
	}, nil
}

func (uc *InitiateLoginUseCase) Execute(ctx context.Context) (*LogoutResponse, error) {
	_, err := uc.whatsappSvc.Logout(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to logout from gateway: %w", err))
	}

	err = uc.configRepo.UpdateConnectionStatus(ctx, "", entity.ConnectionStatusDisconnected)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update status on logout: %w", err))
	}

	return &LogoutResponse{
		Message: "WhatsApp disconnected",
	}, nil
}

type UpdateConfigUseCase struct {
	configRepo ports.WhatsAppConfigRepository
	whatsappSvc *service.WhatsAppService
}

func NewUpdateConfigUseCase(configRepo ports.WhatsAppConfigRepository, whatsappSvc *service.WhatsAppService) *UpdateConfigUseCase {
	return &UpdateConfigUseCase{
		configRepo:   configRepo,
		whatsappSvc: whatsappSvc,
	}
}

type UpdateConfigRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,len=13"`
}

type UpdateConfigResponse struct {
	Message string `json:"message"`
}

func (uc *UpdateConfigUseCase) Execute(ctx context.Context, req *UpdateConfigRequest) (*UpdateConfigResponse, error) {
	_, err := uc.whatsappSvc.Logout(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to disconnect old phone: %w", err))
	}

	newConfig := entity.NewWhatsAppConfig(req.PhoneNumber)
	newConfig.UpdateConnectionStatus(entity.ConnectionStatusQRPending)
	newConfig.UpdateToken("")

	err = uc.configRepo.CreateOrUpdate(ctx, newConfig)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update config: %w", err))
	}

	qrResp, err := uc.whatsappSvc.GetQRCode(ctx, "json")
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	qrExpires := time.Now().Add(time.Duration(qrResp.ExpiresIn) * time.Second)

	err = uc.configRepo.UpdateQRCode(ctx, qrResp.QRCode, qrExpires)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update QR code: %w", err))
	}

	return &UpdateConfigResponse{
		Message: "WhatsApp gateway configured successfully",
	}, nil
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}

func formatQRBase64Ptr(base64 string) *string {
	if base64 == "" {
		return nil
	}
	return &base64
}
