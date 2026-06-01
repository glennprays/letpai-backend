package service

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
)

// WhatsAppService is a thin pass-through to the WAGA gateway SDK.
//
// We deliberately don't persist gateway state. The JWT is configured
// once at startup via WHATSAPP_API_KEY; the SDK derives the phone
// from it on every call. Connection status is fetched live from the
// gateway. No DB read or write happens on the OTP / notification /
// QR / status paths.
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
	resp, err := s.client.GetQRCode(ctx, "json")
	if err != nil {
		return "", time.Time{}, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to get QR code: %w", err))
	}

	qrExpires := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
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
// whatsapp_configs persistence layer was dropped. WAGA's SDK doesn't
// expose a gateway-level logout, so "disconnect" was a UI-only label
// flipping a column we never read; token rotation now happens via
// editing WHATSAPP_API_KEY in .env and restarting the backend.
