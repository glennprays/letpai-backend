package handler

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WhatsAppWebhookHandler handles WhatsApp webhook events
type WhatsAppWebhookHandler struct {
	whatsappSvc *service.WhatsAppService
}

// NewWhatsAppWebhookHandler creates a new WhatsApp webhook handler
func NewWhatsAppWebhookHandler(whatsappSvc *service.WhatsAppService) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{
		whatsappSvc: whatsappSvc,
	}
}

// HandleStatusUpdate handles WhatsApp connection status updates
func (h *WhatsAppWebhookHandler) HandleStatusUpdate(c *fiber.Ctx) error {
	type WebhookPayload struct {
		EventType string `json:"event_type"`
		Token     string `json:"token,omitempty"`
		Status    string `json:"status,omitempty"`
		Phone     string `json:"phone,omitempty"`
	}

	var payload WebhookPayload
	if err := c.BodyParser(&payload); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Verify webhook secret if configured
	webhookSecret := c.Get("X-Webhook-Secret")
	if webhookSecret == "" || webhookSecret != "your-webhook-secret" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid webhook secret",
		})
	}

	// Handle different event types
	switch payload.EventType {
	case "connection.status":
		err := h.handleConnectionStatus(payload.Phone, payload.Status)
		if err != nil {
			apiErr := httperror.FromError(err)
			return c.Status(apiErr.Status).JSON(apiErr.Response())
		}
	default:
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"success": false,
			"error":   "Unknown event type",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
	})
}

// handleConnectionStatus handles connection status updates
func (h *WhatsAppWebhookHandler) handleConnectionStatus(phone, status string) error {
	// TODO: Update WhatsApp config with connection status
	// For now, we just acknowledge the webhook
	return nil
}
