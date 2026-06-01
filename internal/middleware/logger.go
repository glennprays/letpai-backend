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

		err := c.Next()

		latency := time.Since(start)

		fields := map[string]any{
			"status":     c.Response().StatusCode(),
			"method":     c.Method(),
			"path":       c.Path(),
			"ip":         c.IP(),
			"latency":    latency.String(),
			"req_bytes":  len(c.Request().Body()),
			"resp_bytes": len(c.Response().Body()),
		}
		// User ID lands here after the auth middleware runs; absent for
		// unauthenticated routes, which is fine.
		if uid := c.Locals("user_id"); uid != nil {
			fields["user_id"] = uid
		}

		log.Info(requestID, "http request", fields)

		return err
	}
}
