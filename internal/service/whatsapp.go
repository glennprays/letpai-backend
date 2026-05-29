package service

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
	"github.com/google/uuid"
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

// RegisterPhone registers a phone number with the WAGA gateway and
// persists the returned token. The SDK signature is
// `Register(ctx, phoneNumber, secretKey)` — earlier code had these
// swapped (the phone was hard-coded and the caller's phone was being
// used as the secret). We generate a fresh secret per call; the
// gateway will keep accepting it as long as we replay the same
// phoneNumber.
func (s *WhatsAppService) RegisterPhone(ctx context.Context, phoneNumber string) (string, error) {
	secretKey := uuid.NewString()

	resp, err := s.client.Register(ctx, phoneNumber, secretKey)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to register phone: %w", err))
	}

	config := &entity.WhatsAppConfig{
		PhoneNumber:      phoneNumber,
		GatewayToken:     resp.Token,
		ConnectionStatus: entity.ConnectionStatusQRPending,
	}

	if err := s.configRepo.CreateOrUpdate(ctx, config); err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to store config: %w", err))
	}

	return resp.Token, nil
}

// GetQRCode pulls a pairing QR from the gateway and caches it against
// the given phone number in whatsapp_configs.
//
// State machine for that row:
//
//	qr_pending  → set here when the QR is generated.
//	connected   → set by GetLoginStatus once the device pairs.
//	disconnected→ set by GetLoginStatus when the gateway reports the
//	              session has ended, or by Logout().
//
// If no row exists yet for this phone (first pairing on a fresh DB) we
// create a stub so the subsequent UpdateQRCode UPDATE has a target.
func (s *WhatsAppService) GetQRCode(ctx context.Context, phoneNumber string) (string, time.Time, error) {
	if phoneNumber == "" {
		return "", time.Time{}, domain.NewError(domain.ErrBadRequest, fmt.Errorf("phone number is required"))
	}

	existing, err := s.configRepo.Get(ctx)
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to load config: %w", err))
	}
	if existing == nil || existing.PhoneNumber != phoneNumber {
		stub := &entity.WhatsAppConfig{
			PhoneNumber:      phoneNumber,
			ConnectionStatus: entity.ConnectionStatusQRPending,
		}
		if err := s.configRepo.CreateOrUpdate(ctx, stub); err != nil {
			return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to persist config row: %w", err))
		}
	}

	resp, err := s.client.GetQRCode(ctx, "json")
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	qrExpires := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	if err := s.configRepo.UpdateQRCode(ctx, phoneNumber, resp.QrCode, qrExpires); err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update QR code: %w", err))
	}
	// Best-effort: flip the connection_status column to qr_pending so
	// /admin/status reflects the intermediate state. Ignore errors —
	// the row may have been raced ahead to `connected` by a parallel
	// GetLoginStatus poll, which is the desired terminal state.
	_ = s.configRepo.UpdateConnectionStatus(ctx, phoneNumber, entity.ConnectionStatusQRPending)

	return resp.QrCode, qrExpires, nil
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

	// Persist the fresh state back so that anyone reading
	// whatsapp_configs (e.g. GetStatusUseCase) sees the truth without
	// needing its own gateway round trip.
	if status.Authenticated && config.ConnectionStatus != entity.ConnectionStatusConnected {
		if err := s.configRepo.UpdateConnectionStatus(ctx, config.PhoneNumber, entity.ConnectionStatusConnected); err != nil {
			return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update connection status: %w", err))
		}
	} else if !status.Authenticated && config.ConnectionStatus == entity.ConnectionStatusConnected {
		if err := s.configRepo.UpdateConnectionStatus(ctx, config.PhoneNumber, entity.ConnectionStatusDisconnected); err != nil {
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

	err = s.configRepo.UpdateConnectionStatus(ctx, config.PhoneNumber, entity.ConnectionStatusDisconnected)
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
