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
	AuthHandler   *handler.AuthHandler
}

func NewRouter(
	logger *log.Logger,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
) *Router {
	routerLogger := logger.With(log.String("component", "router"))
	return &Router{
		logger:        routerLogger,
		HealthHandler: healthHandler,
		AuthHandler:   authHandler,
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

	// Public routes (no auth required)
	r.setupHealthRoutes(v1)
	r.setupAuthRoutes(v1)

	// Protected routes (require auth)
	// protected := v1.Use(middleware.Authenticate(jwtSvc))
	// r.setupContactRoutes(protected)
	// etc.
}

func (r *Router) setupHealthRoutes(group fiber.Router) {
	group.Get("/health", r.HealthHandler.Check)
}

func (r *Router) setupAuthRoutes(group fiber.Router) {
	auth := group.Group("/auth")
	auth.Post("/register", r.AuthHandler.Register)
	auth.Post("/verify-otp", r.AuthHandler.VerifyOTP)
	auth.Post("/login", r.AuthHandler.Login)
	auth.Post("/logout", r.AuthHandler.Logout)
}
