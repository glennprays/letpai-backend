package service

import (
	"context"
	"fmt"

	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
	"github.com/glennprays/letpai-backend/domain"
)

// WhatsAppService handles sending messages via WhatsApp Gateway
type WhatsAppService struct {
	client *waga.Client
}

// WhatsAppServiceConfig holds configuration for WhatsApp service
type WhatsAppServiceConfig struct {
	BaseURL string
	APIKey  string
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(baseURL, apiKey string) *WhatsAppService {
	client := waga.NewClient(
		waga.WithBaseURL(baseURL),
		waga.WithToken(apiKey),
	)
	return &WhatsAppService{
		client: client,
	}
}

// SendResult contains the result of sending a message
type SendResult struct {
	MessageID string
	Error     error
}

// SendOTP sends an OTP message to a WhatsApp number
// Returns the message ID for tracking
func (s *WhatsAppService) SendOTP(ctx context.Context, phoneNumber, message string) (string, error) {
	// Format phone number to WhatsApp JID format
	recipient := waga.FormatMSISDN(phoneNumber)

	// Send message via WhatsApp Gateway
	resp, err := s.client.SendText(ctx, recipient, message)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to send OTP: %w", err))
	}

	return resp.MessageId, nil
}

// SendNotification sends a notification message
// Returns the message ID for tracking
func (s *WhatsAppService) SendNotification(ctx context.Context, phoneNumber, message string) (string, error) {
	// Format phone number to WhatsApp JID format
	recipient := waga.FormatMSISDN(phoneNumber)

	// Send message via WhatsApp Gateway
	resp, err := s.client.SendText(ctx, recipient, message)
	if err != nil {
		return "", domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to send notification: %w", err))
	}

	return resp.MessageId, nil
}
