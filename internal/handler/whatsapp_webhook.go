package handler

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WhatsAppWebhookHandler handles WhatsApp gateway webhook events
type WhatsAppWebhookHandler struct {
	whatsappSvc *service.WhatsAppService
	configRepo  ports.WhatsAppConfigRepository
}

// NewWhatsAppWebhookHandler creates a new WhatsApp webhook handler
func NewWhatsAppWebhookHandler(
	whatsappSvc *service.WhatsAppService,
	configRepo ports.WhatsAppConfigRepository,
) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{
		whatsappSvc: whatsappSvc,
		configRepo:  configRepo,
	}
}

type webhookPayload struct {
	EventType string `json:"event_type"`
	Token     string `json:"token,omitempty"`
	Status    string `json:"status,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// HandleStatusUpdate handles WhatsApp connection status updates from the gateway
func (h *WhatsAppWebhookHandler) HandleStatusUpdate(c *fiber.Ctx) error {
	var payload webhookPayload
	if err := c.BodyParser(&payload); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
	}

	// Verify webhook secret if configured
	// TODO: move the expected secret into config (cfg.WhatsAppWebhookSecret).
	webhookSecret := c.Get("X-Webhook-Secret")
	if webhookSecret == "" || webhookSecret != "your-webhook-secret" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid webhook secret",
		})
	}

	switch payload.EventType {
	case "connection.status":
		if payload.Phone == "" || payload.Status == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "phone and status are required for connection.status events",
			})
		}
		if err := h.persistConnectionStatus(c.Context(), payload.Phone, payload.Status); err != nil {
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

// persistConnectionStatus updates the WhatsApp config row for the given phone.
func (h *WhatsAppWebhookHandler) persistConnectionStatus(ctx context.Context, phone, status string) error {
	return h.configRepo.UpdateConnectionStatus(ctx, phone, status)
}
