package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	TraceIDHeader     = "X-Trace-ID"
	TraceIDContextKey = "trace_id"
)

func TraceID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		traceID := c.Get(TraceIDHeader)

		if !isValidTraceID(traceID) {
			traceID = uuid.New().String()
		}

		c.Locals(TraceIDContextKey, traceID)
		c.Set(TraceIDHeader, traceID)

		return c.Next()
	}
}

func isValidTraceID(id string) bool {
	if id == "" {
		return false
	}
	_, err := uuid.Parse(id)
	return err == nil
}

// GetTraceID retrieves trace ID from context
func GetTraceID(c *fiber.Ctx) string {
	if id, ok := c.Locals(TraceIDContextKey).(string); ok {
		return id
	}
	return ""
}
