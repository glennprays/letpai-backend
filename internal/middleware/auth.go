package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/glennprays/letpai-backend/internal/service"
)

const (
	userIDKey       = "user_id"
	whatsappKey     = "whatsapp_number"
	authorizationHeader = "Authorization"
	bearerPrefix    = "Bearer "
)

// Authenticate creates a JWT authentication middleware
func Authenticate(jwtSvc *service.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get(authorizationHeader)
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_001",
					"message": "Missing authorization header",
				},
			})
		}

		// Check if it's a Bearer token
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_001",
					"message": "Invalid authorization header format",
				},
			})
		}

		// Extract token
		tokenString := strings.TrimPrefix(authHeader, bearerPrefix)

		// Validate token
		claims, err := jwtSvc.ValidateToken(tokenString)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "AUTH_001",
					"message": "Invalid or expired token",
				},
			})
		}

		// Store user info in context
		c.Locals(userIDKey, claims.UserID)
		c.Locals(whatsappKey, claims.WhatsAppNumber)

		return c.Next()
	}
}

// GetUserID extracts the user ID from the context
func GetUserID(c *fiber.Ctx) string {
	userID, _ := c.Locals(userIDKey).(string)
	return userID
}

// GetWhatsAppNumber extracts the WhatsApp number from the context
func GetWhatsAppNumber(c *fiber.Ctx) string {
	whatsapp, _ := c.Locals(whatsappKey).(string)
	return whatsapp
}
