package middleware

import (
	"github.com/glennprays/letpai-backend/internal/httperror"
	"github.com/gofiber/fiber/v2"
)

// PanicValueKey and PanicTraceKey are the locals keys the recover
// middleware (configured in cmd/api/main.go) writes the recovered
// panic value and goroutine stack trace under. ErrorHandler reads
// them so it can surface the real cause in dev-mode responses
// instead of the generic 500 the user would otherwise see.
const (
	PanicValueKey = "panic_value"
	PanicTraceKey = "panic_trace"
)

// ErrorHandler is a global error handler for Fiber.
//
// debug controls whether panic information is included in the JSON
// response body. It's expected to be set to true only in development
// — leaking stack traces in production would be an information
// disclosure risk. Panics are always logged server-side by the
// recover middleware regardless of this flag.
func ErrorHandler(debug bool) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// 1. Handle Fiber errors FIRST
		if fe, ok := err.(*fiber.Error); ok {
			body := fiber.Map{"error": fe.Message}
			withPanicDetails(c, body, debug)
			return c.Status(fe.Code).JSON(body)
		}

		// 2. Handle domain/application errors
		apiError := httperror.FromError(err)

		if apiError.Status == 0 {
			apiError.Status = fiber.StatusInternalServerError
			apiError.Message = "Internal Server Error"
		}

		body := fiber.Map{"error": apiError.Message}
		withPanicDetails(c, body, debug)
		return c.Status(apiError.Status).JSON(body)
	}
}

// withPanicDetails attaches the recovered panic value + stack trace
// to the response body when debug mode is on AND the recover
// middleware actually captured one for this request. No-op otherwise.
func withPanicDetails(c *fiber.Ctx, body fiber.Map, debug bool) {
	if !debug {
		return
	}
	if v := c.Locals(PanicValueKey); v != nil {
		body["panic"] = v
	}
	if t := c.Locals(PanicTraceKey); t != nil {
		body["trace"] = t
	}
}
