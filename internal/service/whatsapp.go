package service

import (
	"context"
	"fmt"
)

// WhatsAppService handles sending messages via WhatsApp Gateway
type WhatsAppService struct {
	baseURL    string
	apiKey     string
}

// NewWhatsAppService creates a new WhatsApp service
// For MVP v1, this is a stub implementation
func NewWhatsAppService(baseURL, apiKey string) *WhatsAppService {
	return &WhatsAppService{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// SendOTP sends an OTP message to a WhatsApp number
func (s *WhatsAppService) SendOTP(ctx context.Context, phoneNumber, message string) error {
	// TODO: Implement actual WhatsApp Gateway SDK integration
	// For now, just log the message
	fmt.Printf("[WhatsAppService] Sending to %s: %s\n", phoneNumber, message)
	return nil
}

// SendNotification sends a notification message
func (s *WhatsAppService) SendNotification(ctx context.Context, phoneNumber, message string) error {
	// TODO: Implement actual WhatsApp Gateway SDK integration
	fmt.Printf("[WhatsAppService] Notification to %s: %s\n", phoneNumber, message)
	return nil
}
