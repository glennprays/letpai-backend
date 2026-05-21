package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// WhatsAppWebhookHandler handles WhatsApp gateway connection-status events.
type WhatsAppWebhookHandler struct {
	whatsappSvc   *service.WhatsAppService
	configRepo    ports.WhatsAppConfigRepository
	webhookSecret string
}

// NewWhatsAppWebhookHandler creates a new WhatsApp webhook handler
func NewWhatsAppWebhookHandler(
	cfg *config.Config,
	whatsappSvc *service.WhatsAppService,
	configRepo ports.WhatsAppConfigRepository,
) *WhatsAppWebhookHandler {
	return &WhatsAppWebhookHandler{
		whatsappSvc:   whatsappSvc,
		configRepo:    configRepo,
		webhookSecret: cfg.WhatsAppWebhookSecret,
	}
}

type webhookPayload struct {
	EventType string `json:"event_type"`
	Token     string `json:"token,omitempty"`
	Status    string `json:"status,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// HandleStatusUpdate handles WhatsApp connection status updates from the gateway.
//
// Auth: the gateway can send either a static shared secret in `X-Webhook-Secret`
// or an HMAC-SHA256 signature in `X-Webhook-Signature` (hex, optionally
// `sha256=`-prefixed). Both modes verify against cfg.WhatsAppWebhookSecret using
// constant-time comparison.
func (h *WhatsAppWebhookHandler) HandleStatusUpdate(c *fiber.Ctx) error {
	if h.webhookSecret == "" {
		// Refuse to process if the secret was never configured — fail closed.
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"error":   "Webhook secret not configured on server",
		})
	}

	body := c.Body()

	if !h.verifyRequest(c, body) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid webhook signature",
		})
	}

	var payload webhookPayload
	if err := c.BodyParser(&payload); err != nil {
		apiErr := httperror.FromError(err)
		return c.Status(apiErr.Status).JSON(apiErr.Response())
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

// verifyRequest accepts either a static shared-secret header or an HMAC-SHA256
// signature over the request body. Both are compared constant-time.
func (h *WhatsAppWebhookHandler) verifyRequest(c *fiber.Ctx, body []byte) bool {
	if sig := c.Get("X-Webhook-Signature"); sig != "" {
		expected := hmacHex(body, h.webhookSecret)
		// Allow either "sha256=<hex>" or bare "<hex>".
		got := strings.TrimPrefix(sig, "sha256=")
		return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
	}
	if shared := c.Get("X-Webhook-Secret"); shared != "" {
		return subtle.ConstantTimeCompare([]byte(shared), []byte(h.webhookSecret)) == 1
	}
	return false
}

func hmacHex(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
