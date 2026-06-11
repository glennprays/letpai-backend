package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// RateLimitConfig holds configuration for rate limiting middleware
type RateLimitConfig struct {
	KeyExtractor   func(c *fiber.Ctx) string
	CheckRateLimit func(ctx context.Context, key string) (*service.RateLimitResult, error)
	OnRateLimited  func(c *fiber.Ctx, result *service.RateLimitResult) error
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

// extractWhatsAppNumber peeks at the JSON request body to extract a phone
// number for rate-limit keying. Fiber's c.Body() returns the raw bytes
// without consuming them, so this is safe to call before BodyParser runs.
// Returns "" on any failure (caller falls back to IP-only rate limit).
func extractWhatsAppNumber(c *fiber.Ctx) string {
	body := c.Body()
	if len(body) == 0 || len(body) > 4096 {
		return ""
	}
	var payload struct {
		WhatsAppNumber string `json:"whatsapp_number"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.WhatsAppNumber
}

// LoginRateLimiter returns rate limiting middleware for login endpoint.
// Keys on `ip:{ip}:phone:{phone}` so that an attacker with multiple IPs can't
// brute-force one account, and a victim isn't locked out by attackers
// targeting unrelated accounts from their IP.
func LoginRateLimiter(rateLimitService *service.RateLimitService) fiber.Handler {
	return RateLimitMiddleware(RateLimitConfig{
		KeyExtractor: func(c *fiber.Ctx) string {
			ip := c.IP()
			phone := extractWhatsAppNumber(c)
			if phone == "" {
				return fmt.Sprintf("login:ip:%s", ip)
			}
			return fmt.Sprintf("login:ip:%s:phone:%s", ip, phone)
		},
		CheckRateLimit: func(ctx context.Context, key string) (*service.RateLimitResult, error) {
			return rateLimitService.CheckLoginRateLimit(ctx, key)
		},
	})
}

// VerifyOTPRateLimiter rate-limits OTP verification on TWO dimensions: per-IP
// (stops one host hammering the endpoint) AND per-phone across all IPs (stops a
// botnet spreading brute-force guesses of the 6-digit code over many
// addresses). A request is rejected if either limit is exceeded.
func VerifyOTPRateLimiter(rateLimitService *service.RateLimitService) fiber.Handler {
	respond := func(c *fiber.Ctx, result *service.RateLimitResult) error {
		c.Set("Retry-After", strconv.Itoa(service.CalculateRetryAfterSeconds(result.RetryAfter)))
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "RATE_LIMIT_001",
				"message": fmt.Sprintf("Too many attempts. Please try again in %s", service.FormatRetryAfter(result.RetryAfter)),
			},
			"retry_after": service.CalculateRetryAfterSeconds(result.RetryAfter),
		})
	}

	return func(c *fiber.Ctx) error {
		// Per-phone limit across all IPs (botnet brute-force guard). Best-effort:
		// skip on extraction/limiter failure so a Redis blip doesn't block logins.
		if phone := extractWhatsAppNumber(c); phone != "" {
			if res, err := rateLimitService.CheckOTPPhoneRateLimit(c.Context(), phone); err == nil && !res.Allowed {
				return respond(c, res)
			}
		}

		// Per-IP limit.
		res, err := rateLimitService.CheckOTPRateLimit(c.Context(), "ip:"+c.IP())
		if err != nil {
			return c.Next() // fail-open on limiter error
		}
		if !res.Allowed {
			return respond(c, res)
		}
		return c.Next()
	}
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
			retryAfterSec := service.CalculateRetryAfterSeconds(result.RetryAfter)
			c.Set("Retry-After", strconv.Itoa(retryAfterSec))
			nextAvailableAt := time.Now().Add(result.RetryAfter).UTC().Format("2006-01-02T15:04:05Z07:00")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_002",
					"message": fmt.Sprintf("Try again in %s", service.FormatRetryAfter(result.RetryAfter)),
				},
				"retry_after":       retryAfterSec,
				"next_available_at": nextAvailableAt,
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
