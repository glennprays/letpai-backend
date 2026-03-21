package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	KeyExtractor    func(c *fiber.Ctx) string
	CheckRateLimit  func(ctx context.Context, key string) (*service.RateLimitResult, error)
	OnRateLimited   func(c *fiber.Ctx, result *service.RateLimitResult) error
}

// RateLimitMiddleware returns a middleware that applies rate limiting
func RateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract the key for rate limiting
		key := config.KeyExtractor(c)
		if key == "" {
			return c.Next()
		}

		// Check rate limit
		result, err := config.CheckRateLimit(c.Context(), key)
		if err != nil {
			// Log error but don't block request on rate limit service failure
			// (fail-open approach)
			return c.Next()
		}

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(result.Limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(result.ResetAfter).Unix(), 10))

		// Check if allowed
		if !result.Allowed {
			c.Set("Retry-After", strconv.Itoa(service.CalculateRetryAfterSeconds(result.RetryAfter)))

			if config.OnRateLimited != nil {
				return config.OnRateLimited(c, result)
			}

			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_001",
					"message": fmt.Sprintf("Too many attempts. Please try again in %s", service.FormatRetryAfter(result.RetryAfter)),
				},
				"retry_after": service.CalculateRetryAfterSeconds(result.RetryAfter),
			})
		}

		return c.Next()
	}
}

// LoginRateLimiter returns rate limiting middleware for login endpoint
func LoginRateLimiter(rateLimitService *service.RateLimitService) fiber.Handler {
	return RateLimitMiddleware(RateLimitConfig{
		KeyExtractor: func(c *fiber.Ctx) string {
			// Use IP address + WhatsApp number combination for login rate limiting
			// This prevents both IP-based and credential-based attacks
			ip := c.IP()
			// Also extract the WhatsApp number from request if available
			// For login, we check rate limit before parsing body, so use IP first
			return fmt.Sprintf("login:ip:%s", ip)
		},
		CheckRateLimit: func(ctx context.Context, key string) (*service.RateLimitResult, error) {
			return rateLimitService.CheckLoginRateLimit(ctx, key)
		},
	})
}

// VerifyOTPRateLimiter returns rate limiting middleware for OTP verification endpoint
func VerifyOTPRateLimiter(rateLimitService *service.RateLimitService) fiber.Handler {
	return RateLimitMiddleware(RateLimitConfig{
		KeyExtractor: func(c *fiber.Ctx) string {
			// For OTP, we'll rate limit per IP to prevent OTP enumeration attacks
			ip := c.IP()
			return fmt.Sprintf("verifyotp:ip:%s", ip)
		},
		CheckRateLimit: func(ctx context.Context, key string) (*service.RateLimitResult, error) {
			// Use OTP verification rate limit (3 per IP per 15 minutes)
			return rateLimitService.CheckOTPRateLimit(ctx, key)
		},
	})
}

// RegisterRateLimiter returns rate limiting middleware for registration endpoint
func RegisterRateLimiter(rateLimitService *service.RateLimitService) fiber.Handler {
	return RateLimitMiddleware(RateLimitConfig{
		KeyExtractor: func(c *fiber.Ctx) string {
			ip := c.IP()
			return fmt.Sprintf("register:ip:%s", ip)
		},
		CheckRateLimit: func(ctx context.Context, key string) (*service.RateLimitResult, error) {
			// Same as login - 5 attempts per 15 minutes per IP
			return rateLimitService.CheckLoginRateLimit(ctx, key)
		},
	})
}

// ReminderRateLimiter returns rate limiting middleware for reminder endpoint
func ReminderRateLimiter(rateLimitService *service.RateLimitService) fiber.Handler {
	return RateLimitMiddleware(RateLimitConfig{
		KeyExtractor: func(c *fiber.Ctx) string {
			// Extract participant ID from URL params
			participantID := c.Params("participant_id")
			return participantID
		},
		CheckRateLimit: func(ctx context.Context, key string) (*service.RateLimitResult, error) {
			return rateLimitService.CheckReminderRateLimit(ctx, key)
		},
		OnRateLimited: func(c *fiber.Ctx, result *service.RateLimitResult) error {
			c.Set("Retry-After", strconv.Itoa(service.CalculateRetryAfterSeconds(result.RetryAfter)))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_002",
					"message": fmt.Sprintf("Reminder rate limit exceeded. Next reminder available in %s", service.FormatRetryAfter(result.RetryAfter)),
				},
				"retry_after": service.CalculateRetryAfterSeconds(result.RetryAfter),
			})
		},
	})
}

// GeneralRateLimiter returns a general-purpose rate limiting middleware
// Useful for protecting API endpoints from abuse
func GeneralRateLimiter(rateLimitService *service.RateLimitService, limit int, window time.Duration) fiber.Handler {
	return RateLimitMiddleware(RateLimitConfig{
		KeyExtractor: func(c *fiber.Ctx) string {
			// Use user ID from auth context if available, otherwise use IP
			userID := c.Locals("user_id")
			if userID != nil {
				return fmt.Sprintf("api:user:%v", userID)
			}
			return fmt.Sprintf("api:ip:%s", c.IP())
		},
		CheckRateLimit: func(ctx context.Context, key string) (*service.RateLimitResult, error) {
			return rateLimitService.GetRateLimitStatus(ctx, "api", key, limit, window)
		},
	})
}
