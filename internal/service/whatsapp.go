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

// GetQRCode pulls a pairing QR from the gateway and caches it on the
// singleton whatsapp_configs row.
//
// WAGA's JWT (set via WHATSAPP_API_KEY -> waga.WithToken) is bound to
// the phone that called POST /register on the gateway. Every
// subsequent SDK call — GetQRCode, GetLoginStatus, SendText — is
// implicitly scoped to that phone; the SDK takes no phone parameter
// on any of them. So we don't ask the operator to type one.
//
// State machine for the local row:
//
//	qr_pending  → set here when the QR is generated.
//	connected   → set by GetLoginStatus once the device pairs.
//	disconnected→ set by GetLoginStatus when the gateway reports the
//	              session has ended, or by Logout().
func (s *WhatsAppService) GetQRCode(ctx context.Context) (string, time.Time, error) {
	// Make sure there's a row to UPDATE before we issue the SQL below.
	// EnsureSingletonRow is idempotent and uses ON CONFLICT (phone_number)
	// against the empty-string singleton key.
	if err := s.configRepo.EnsureSingletonRow(ctx); err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to ensure config row: %w", err))
	}

	resp, err := s.client.GetQRCode(ctx, "json")
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	qrExpires := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)

	if err := s.configRepo.UpdateQRCodeSingleton(ctx, resp.QrCode, qrExpires); err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update QR code: %w", err))
	}
	// Best-effort: flip the connection_status column to qr_pending so
	// /admin/status reflects the intermediate state. Ignore errors —
	// the row may have been raced ahead to `connected` by a parallel
	// GetLoginStatus poll, which is the desired terminal state.
	_ = s.configRepo.UpdateConnectionStatusSingleton(ctx, entity.ConnectionStatusQRPending)

	return resp.QrCode, qrExpires, nil
}

// GetLoginStatus polls the gateway and writes the fresh state back to
// the singleton config row.
//
// The early-return-when-no-token check is intentionally permissive:
// we only short-circuit when there's no env-supplied gateway token at
// all (the SDK client is constructed with WithToken(apiKey) from env).
// The DB row's gateway_token column is only populated by the
// RegisterPhone flow — it's not the source of truth for "can we talk
// to the gateway", and gating on it caused the status badge to stay
// "Not paired" forever for operators who only configured the env.
func (s *WhatsAppService) GetLoginStatus(ctx context.Context) (bool, string, error) {
	config, err := s.configRepo.Get(ctx)
	if err != nil {
		return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get config: %w", err))
	}

	status, err := s.client.GetLoginStatus(ctx)
	if err != nil {
		return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get login status: %w", err))
	}

	// Persist the fresh state back so anyone reading whatsapp_configs
	// (e.g. GetStatusUseCase) sees the truth without needing its own
	// gateway round trip. Use the singleton variants since we no
	// longer key UPDATEs by phone number.
	currentStatus := ""
	if config != nil {
		currentStatus = config.ConnectionStatus
	}
	if status.Authenticated && currentStatus != entity.ConnectionStatusConnected {
		if err := s.configRepo.EnsureSingletonRow(ctx); err != nil {
			return false, "", err
		}
		if err := s.configRepo.UpdateConnectionStatusSingleton(ctx, entity.ConnectionStatusConnected); err != nil {
			return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update connection status: %w", err))
		}
	} else if !status.Authenticated && currentStatus == entity.ConnectionStatusConnected {
		if err := s.configRepo.UpdateConnectionStatusSingleton(ctx, entity.ConnectionStatusDisconnected); err != nil {
			return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update connection status: %w", err))
		}
	}

	return status.Authenticated, "", nil
}

func (s *WhatsAppService) Logout(ctx context.Context) error {
	// Singleton row variant: drops the phone-key match so we work in
	// the post-cleanup world where rows live without a phone label.
	if err := s.configRepo.UpdateConnectionStatusSingleton(ctx, entity.ConnectionStatusDisconnected); err != nil {
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
