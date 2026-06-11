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
		// Status update for an outgoing message. Parse to get the message ID.
		outgoingPayload, err := verifier.ParseOutgoingWebhook(body, signature)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error":   "Invalid webhook payload",
			})
		}

		// Resolve the log row this event belongs to. An unknown message ID
		// means there is nothing to update (e.g. a send we never recorded, or
		// a row already pruned) — acknowledge so the gateway stops retrying.
		log, err := h.notificationLogRepo.FindByWhatsAppMessageID(c.Context(), outgoingPayload.MessageId)
		if err != nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "no matching notification log",
			})
		}

		// Forward-only state machine: ignore duplicate / out-of-order /
		// terminal events so re-delivered webhooks are idempotent no-ops.
		next, ok := entity.NextWebhookStatus(log.Status, string(payload.Event))
		if !ok {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"success": true,
				"message": "no-op",
			})
		}

		var errMsg *string
		if next == entity.NotificationStatusFailed {
			m := "Failed to deliver via WhatsApp"
			errMsg = &m
		}

		// Compare-and-swap on the observed current status. A real DB error
		// returns 5xx so the gateway retries; a lost CAS (another delivery
		// advanced the row first) is reported as success inside the repo.
		if err := h.notificationLogRepo.UpdateStatusGuarded(
			c.Context(), log.LogID.String(), string(next), string(log.Status), errMsg,
		); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Failed to update notification status",
			})
		}

	case "message.incoming":
		// Incoming message - not currently used for Letpai.
	default:
		// Unknown event type - acknowledge without failing.
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Webhook received",
	})
}
