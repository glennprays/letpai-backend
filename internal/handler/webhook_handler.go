package handler

import (
	waga "github.com/glennprays/whatsapp-gateway-sdk-go"
	"github.com/gofiber/fiber/v2"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// WebhookHandler handles webhook requests from WhatsApp Gateway
type WebhookHandler struct {
	webhookSecret       string
	notificationLogRepo ports.NotificationLogRepository
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(cfg *config.Config, notificationLogRepo ports.NotificationLogRepository) *WebhookHandler {
	return &WebhookHandler{
		webhookSecret:       cfg.WhatsAppWebhookSecret,
		notificationLogRepo: notificationLogRepo,
	}
}

// HandleWhatsAppStatus handles incoming webhook requests from WhatsApp Gateway
// @Summary WhatsApp status webhook
// @Description Receive status updates from WhatsApp Gateway
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param body body WhatsAppStatusWebhook true "Webhook payload"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /webhooks/whatsapp-status [post]
func (h *WebhookHandler) HandleWhatsAppStatus(c *fiber.Ctx) error {
	// Read request body as []byte
	body := c.Body()

	// Get webhook signature from header
	signature := c.Get("X-Webhook-Signature")
	if signature == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Missing webhook signature",
		})
	}

	// Verify webhook signature and parse payload
	verifier := waga.NewWebhookVerifier(h.webhookSecret)
	payload, err := verifier.ParseIncomingWebhook(body, signature)
	if err != nil {
		// Log error for debugging
		if err.Error() == "invalid signature" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid webhook signature",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid webhook payload",
		})
	}

	// Process webhook based on event type string
	switch string(payload.Event) {
	case "message.queued", "message.sent", "message.failed":
		// These are status update events (outgoing messages)
		// Parse as outgoing webhook payload to get message ID
		outgoingPayload, err := verifier.ParseOutgoingWebhook(body, signature)
		if err != nil {
			// Failed to parse as outgoing, skip
			break
		}

		// Find notification log by WhatsApp message ID
		log, err := h.notificationLogRepo.FindByWhatsAppMessageID(c.Context(), outgoingPayload.MessageId)
		if err != nil {
			// Log not found, skip
			break
		}

		// Update status based on event
		switch string(payload.Event) {
		case "message.queued":
			// Already queued, no update needed
			break
		case "message.sent":
			err := h.notificationLogRepo.UpdateStatus(c.Context(), log.LogID.String(), string(entity.NotificationStatusSent))
			if err != nil {
				break
			}
		case "message.failed":
			errMsg := "Failed to deliver via WhatsApp"
			err := h.notificationLogRepo.UpdateStatusWithError(c.Context(), log.LogID.String(), string(entity.NotificationStatusFailed), errMsg)
			if err != nil {
				break
			}
		}

	case "message.incoming":
		// Incoming message - not currently used for Letpai
		// But could be used for future features
		break
	default:
		// Unknown event type - log but don't fail
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Webhook received",
	})
}
