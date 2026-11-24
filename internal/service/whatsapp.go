package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
)

// WhatsAppService is a thin pass-through to the WAGA gateway SDK.
//
// The JWT no longer comes from WHATSAPP_API_KEY (that env is now the
// /register secret). It is minted at runtime by
// RegisterGatewayUseCase, persisted encrypted in whatsapp_configs,
// and rehydrated onto the shared client at boot by
// infrastructure.RestoreGatewayToken. Connection status is fetched
// live from the gateway.
type WhatsAppService struct {
	client *waga.Client
}

func NewWhatsAppService(baseURL, apiKey string) *WhatsAppService {
	client := waga.NewClient(
		waga.WithBaseURL(baseURL),
		waga.WithToken(apiKey),
	)
	return &WhatsAppService{client: client}
}

// SetToken swaps the gateway JWT on the shared client. Used by boot
// restore and by RegisterGatewayUseCase right after minting.
func (s *WhatsAppService) SetToken(token string) {
	s.client.SetToken(token)
}

// GetToken returns the JWT currently loaded on the shared client.
func (s *WhatsAppService) GetToken() string {
	return s.client.GetToken()
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

// Register mints a gateway JWT for phoneNumber via the gateway's
// POST /register using the configured registration secret. The SDK
// stores the token on the shared client; persisting it (encrypted) is
// the caller's job.
func (s *WhatsAppService) Register(ctx context.Context, phoneNumber, secretKey string) (*waga.RegisterResponse, error) {
	resp, err := s.client.Register(ctx, phoneNumber, secretKey)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("gateway register failed: %w", err))
	}
	return resp, nil
}

// Logout unpairs the currently-scoped device session on the gateway.
// Best-effort by callers when switching phones: an error here doesn't
// block registering a new phone.
func (s *WhatsAppService) Logout(ctx context.Context) error {
	if err := s.client.Logout(ctx); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("gateway logout failed: %w", err))
	}
	return nil
}

// GetQRCode pulls a pairing QR from the gateway.
//
// WAGA's JWT (set via WHATSAPP_API_KEY -> waga.WithToken) is bound to
// the phone that called POST /register on the gateway. Every
// subsequent SDK call -- GetQRCode, GetLoginStatus, SendText -- is
// implicitly scoped to that phone.
//
// We deliberately don't cache the QR locally. Each call is cheap, the
// QR has a short TTL, and storing it in the DB just creates a
// secondary source of truth that drifts from the gateway. The gateway
// is the authority for "current QR" and "current connection status";
// we just pass that through.
func (s *WhatsAppService) GetQRCode(ctx context.Context) (string, time.Time, error) {
	// Edge case: the device is already paired, so minting a QR would be
	// meaningless at best (gateway errors) and confusing at worst.
	authenticated, _, statusErr := s.GetLoginStatus(ctx)
	if statusErr == nil && authenticated {
		return "", time.Time{}, domain.NewError(domain.ErrBadRequest, errors.New("gateway already paired"))
	}

	resp, err := s.client.GetQRCode(ctx, "json")
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	// WAGA's json QR payload carries `timeout` (seconds), not
	// `expires_in`. SDK v0.8.1+ binds it as resp.Timeout, so prefer the
	// gateway-reported value; ExpiresIn covers gateways that send the
	// other spelling, and the 60 s last resort is WhatsApp's own QR
	// validity for pre-v0.8.1 SDKs and gateways that omit both.
	ttl := resp.Timeout
	if ttl <= 0 {
		ttl = resp.ExpiresIn
	}
	if ttl <= 0 {
		ttl = 60
	}
	qrExpires := time.Now().Add(time.Duration(ttl) * time.Second)
	return resp.QrCode, qrExpires, nil
}

// GetLoginStatus calls the gateway and returns whether the device is
// currently paired. No DB read or write — the gateway is the source
// of truth.
func (s *WhatsAppService) GetLoginStatus(ctx context.Context) (bool, string, error) {
	status, err := s.client.GetLoginStatus(ctx)
	if err != nil {
		return false, "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get login status: %w", err))
	}
	return status.Authenticated, "", nil
}

// Logout, SetToken, RegisterPhone were removed when the local
// whatsapp_configs persistence layer was dropped; they are reintroduced
// by the gateway-JWT store feature.
