package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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

	// Create Fiber app with custom error handler.
	// BodyLimit is sized for payment-proof uploads: the frontend caps the
	// image at 5 MB; base64 inflates that to ~6.7 MB. 8 MB leaves headroom
	// without inviting denial-of-service via giant payloads.
	fiberApp := fiber.New(fiber.Config{
		AppName:               app.Config.AppName,
		ErrorHandler:          middleware.ErrorHandler(),
		DisableStartupMessage: true,
		BodyLimit:             8 * 1024 * 1024,
	})

	// Built-in middleware
	fiberApp.Use(recover.New())

	// Setup routes (includes custom middleware)
	app.Router.Setup(fiberApp)

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
