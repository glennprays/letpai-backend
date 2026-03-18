package router

import (
	"github.com/glennprays/letpai-backend/internal/handler"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/log"
	"github.com/gofiber/fiber/v2"
)

type Router struct {
	logger        *log.Logger
	HealthHandler *handler.HealthHandler
}

func NewRouter(
	logger *log.Logger,
	healthHandler *handler.HealthHandler,
) *Router {
	routerLogger := logger.With(log.String("component", "router"))
	return &Router{
		logger:        routerLogger,
		HealthHandler: healthHandler,
	}
}

// Setup configures all application routes
func (r *Router) Setup(app *fiber.App) {
	// Global middleware
	app.Use(middleware.TraceID())
	app.Use(middleware.CORS())

	app.Use(middleware.NewHTTPLogger(r.logger))

	// API v1 group
	v1 := app.Group("/api/v1")

	// Health routes
	r.setupHealthRoutes(v1)

	// Future route groups can be added here:
	// r.setupUserRoutes(v1)
	// r.setupAuthRoutes(v1)
}

func (r *Router) setupHealthRoutes(group fiber.Router) {
	group.Get("/health", r.HealthHandler.Check)
}
