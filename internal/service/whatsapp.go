package service

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
)

type WhatsAppService struct {
	client     *waga.Client
	configRepo ports.WhatsAppConfigRepository
}

func NewWhatsAppService(baseURL, apiKey string, configRepo ports.WhatsAppConfigRepository) *WhatsAppService {
	client := waga.NewClient(
		waga.WithBaseURL(baseURL),
		waga.WithToken(apiKey),
	)
	return &WhatsAppService{
		client:     client,
		configRepo: configRepo,
	}
}

type SendResult struct {
	MessageID string
	Error     error
}

func (s *WhatsAppService) SendOTP(ctx context.Context, phoneNumber, message string) (string, error) {
	recipient := waga.FormatMSISDN(phoneNumber)

	resp, err := s.client.SendText(ctx, recipient, message)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to send OTP: %w", err))
	}

	return resp.MessageId, nil
}

func (s *WhatsAppService) SendNotification(ctx context.Context, phoneNumber, message string) (string, error) {
	recipient := waga.FormatMSISDN(phoneNumber)

	resp, err := s.client.SendText(ctx, recipient, message)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to send notification: %w", err))
	}

	return resp.MessageId, nil
}

func (s *WhatsAppService) RegisterPhone(ctx context.Context, phoneNumber string) (string, error) {
	resp, err := s.client.Register(ctx, "6281234567890", phoneNumber)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to register phone: %w", err))
	}

	config := &entity.WhatsAppConfig{
		PhoneNumber: phoneNumber,
	}
	config.GatewayToken = resp.Token

	err = s.configRepo.CreateOrUpdate(ctx, config)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to store config: %w", err))
	}

	return resp.Token, nil
}

func (s *WhatsAppService) GetQRCode(ctx context.Context) (string, time.Time, error) {
	resp, err := s.client.GetQRCode(ctx, "json")
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	qrExpires := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	err = s.configRepo.UpdateQRCode(ctx, "", resp.QRCode, qrExpires)
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update QR code: %w", err))
	}

	return resp.QRCode, qrExpires, nil
}

func (s *WhatsAppService) GetLoginStatus(ctx context.Context) (bool, string, error) {
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	if config == nil || config.GatewayToken == "" {
		return false, "No gateway token configured", nil
	}

	status, err := s.client.GetLoginStatus(ctx)
	if err != nil {
		return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get login status: %w", err))
	}

	if status.Authenticated {
		err = s.configRepo.UpdateConnectionStatus(ctx, config.PhoneNumber, entity.ConnectionStatusConnected)
		if err != nil {
			return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update connection status: %w", err))
		}
	}

	return status.Authenticated, "", nil
}

func (s *WhatsAppService) Logout(ctx context.Context) error {
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	if config == nil {
		return nil
	}

	_, err = s.configRepo.UpdateConnectionStatus(ctx, config.PhoneNumber, entity.ConnectionStatusDisconnected)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update status on logout: %w", err))
	}

	return nil
}

func (s *WhatsAppService) SetToken(ctx context.Context, phoneNumber, token string) error {
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	var newConfig *entity.WhatsAppConfig
	if config == nil || config.PhoneNumber != phoneNumber {
		newConfig = &entity.WhatsAppConfig{
			PhoneNumber: phoneNumber,
		}
	} else {
		newConfig = config
	}
	newConfig.GatewayToken = token

	err = s.configRepo.CreateOrUpdate(ctx, newConfig)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to set gateway token: %w", err))
	}

	return nil
}
