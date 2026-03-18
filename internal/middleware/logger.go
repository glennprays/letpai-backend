package middleware

import (
	"time"

	"github.com/glennprays/log"
	"github.com/gofiber/fiber/v2"
)

func NewHTTPLogger(log *log.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := GetTraceID(c)
		start := time.Now()

		// Process request
		err := c.Next()

		latency := time.Since(start)

		log.Info(requestID, "http request", map[string]any{
			"status":  c.Response().StatusCode(),
			"method":  c.Method(),
			"path":    c.Path(),
			"ip":      c.IP(),
			"latency": latency.String(),
		})

		return err
	}
}
