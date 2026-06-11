package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/infrastructure"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
)

func main() {
	lifecycleID := uuid.New().String()
	// Initialize app dependencies via Wire
	app, err := infrastructure.InitializeApp()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize app: %v", err))
	}
	defer app.Logger.Sync()

	logger := app.Logger.With(log.String("component", "main"))

	// debugErrors mirrors panic information into HTTP response bodies
	// so operators don't have to tail server logs to debug a panic in
	// development. Off in staging and production -- leaking stack
	// traces to clients is an information-disclosure risk.
	debugErrors := app.Config.Env == config.DEV
	if debugErrors {
		logger.Info(lifecycleID, "Panic traces will be returned in HTTP responses (development)", nil)
	}

	// Create Fiber app with custom error handler.
	// BodyLimit is sized for payment-proof uploads: the frontend caps the
	// image at 5 MB; base64 inflates that to ~6.7 MB. 8 MB leaves headroom
	// without inviting denial-of-service via giant payloads.
	fiberApp := fiber.New(fiber.Config{
		AppName:               app.Config.AppName,
		ErrorHandler:          middleware.ErrorHandler(debugErrors),
		DisableStartupMessage: true,
		BodyLimit:             8 * 1024 * 1024,
	})

	// Panic recovery: stash the panic value + stack trace on the
	// request locals so middleware.ErrorHandler can surface them in
	// dev mode, and always log them server-side so even production
	// has a record. Without this the recover middleware swallows the
	// panic, the ErrorHandler returns a generic 500, and you're left
	// with no breadcrumb.
	fiberApp.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			trace := string(debug.Stack())
			panicValue := fmt.Sprintf("%v", e)
			logger.Error("panic-handler", "panic recovered in HTTP handler", map[string]any{
				"panic":  panicValue,
				"trace":  trace,
				"method": c.Method(),
				"path":   c.Path(),
			})
			c.Locals(middleware.PanicValueKey, panicValue)
			c.Locals(middleware.PanicTraceKey, trace)
		},
	}))

	// Setup routes (includes custom middleware)
	app.Router.Setup(fiberApp)

	// Start the durable WhatsApp delivery worker. It drains the
	// notification_logs outbox in the background; cancelling workerCtx on
	// shutdown stops it cleanly. Any send interrupted mid-flight is left in
	// 'sending' and requeued by the worker's stuck-row reaper on next start.
	workerCtx, stopWorker := context.WithCancel(context.Background())
	go app.NotificationWorker.Run(workerCtx)

	// Start server in goroutine
	addr := fmt.Sprintf(":%d", app.Config.AppPort)
	go func() {
		logger.Info(lifecycleID, "Starting server", map[string]any{
			"address":  addr,
			"app_name": app.Config.AppName,
			"pid":      os.Getpid(),
		})
		if err := fiberApp.Listen(addr); err != nil {
			logger.Fatal(lifecycleID, "Failed to start server", map[string]any{
				"error": err.Error(),
			})
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info(lifecycleID, "Shutting down server", nil)

	// Stop the delivery worker before draining HTTP so no new sends start.
	stopWorker()

	// Timeout context for shutdown
	timeoutSeconds := 10
	if app.Config.Env == config.DEV {
		timeoutSeconds = 0 // No timeout in dev for easier debugging
	}
	_, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	if err := fiberApp.Shutdown(); err != nil {
		logger.Fatal(lifecycleID, "Server forced to shutdown", map[string]any{
			"error": err.Error(),
		})
	}

	logger.Info(lifecycleID, "Server exited", nil)
}
