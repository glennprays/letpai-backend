package router

import (
	"github.com/glennprays/letpai-backend/internal/handler"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/log"
	"github.com/gofiber/fiber/v2"
)

type Router struct {
	logger              *log.Logger
	HealthHandler       *handler.HealthHandler
	AuthHandler         *handler.AuthHandler
	ContactGroupHandler *handler.ContactGroupHandler
	ContactHandler      *handler.ContactHandler
	SessionHandler      *handler.SessionHandler
	PaymentHandler      *handler.PaymentHandler
	NotificationHandler *handler.NotificationHandler
	WebhookHandler      *handler.WebhookHandler
	DashboardHandler    *handler.DashboardHandler
	jwtService          *service.JWTService
	rateLimitService    *service.RateLimitService
}

func NewRouter(
	logger *log.Logger,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	contactGroupHandler *handler.ContactGroupHandler,
	contactHandler *handler.ContactHandler,
	sessionHandler *handler.SessionHandler,
	paymentHandler *handler.PaymentHandler,
	notificationHandler *handler.NotificationHandler,
	webhookHandler *handler.WebhookHandler,
	dashboardHandler *handler.DashboardHandler,
	jwtService          *service.JWTService,
	rateLimitService    *service.RateLimitService,
) *Router {
	routerLogger := logger.With(log.String("component", "router"))
	return &Router{
		logger:              routerLogger,
		HealthHandler:       healthHandler,
		AuthHandler:         authHandler,
		ContactGroupHandler: contactGroupHandler,
		ContactHandler:      contactHandler,
		SessionHandler:      sessionHandler,
		PaymentHandler:      paymentHandler,
		NotificationHandler: notificationHandler,
		WebhookHandler:      webhookHandler,
		DashboardHandler:   dashboardHandler,
		jwtService:          jwtService,
		rateLimitService:    rateLimitService,
	}
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
	r.setupWebhookRoutes(v1)
	r.setupAuthRoutes(v1)
	r.setupPublicPaymentRoutes(v1)

	// Protected routes (require auth)
	protected := v1.Use(middleware.Authenticate(r.jwtService))
	r.setupContactGroupRoutes(protected)
	r.setupContactRoutes(protected)
	r.setupSessionRoutes(protected)
	r.setupProtectedPaymentRoutes(protected)
	r.setupNotificationRoutes(protected)
	r.setupDashboardRoutes(protected)
}

func (r *Router) setupHealthRoutes(group fiber.Router) {
	group.Get("/health", r.HealthHandler.Check)
}

func (r *Router) setupWebhookRoutes(group fiber.Router) {
	webhooks := group.Group("/webhooks")
	webhooks.Post("/whatsapp-status", r.WebhookHandler.HandleWhatsAppStatus)
}

func (r *Router) setupAuthRoutes(group fiber.Router) {
	auth := group.Group("/auth")
	auth.Post("/register", middleware.RegisterRateLimiter(r.rateLimitService), r.AuthHandler.Register)
	auth.Post("/verify-otp", middleware.VerifyOTPRateLimiter(r.rateLimitService), r.AuthHandler.VerifyOTP)
	auth.Post("/login", middleware.LoginRateLimiter(r.rateLimitService), r.AuthHandler.Login)
	auth.Post("/logout", r.AuthHandler.Logout)
	auth.Post("/profile", r.AuthHandler.UpdateProfile)
}

func (r *Router) setupPublicPaymentRoutes(group fiber.Router) {
	// Public payment routes (no auth required)
	payments := group.Group("/payments")
	payments.Post("/:participant_id/submit", r.PaymentHandler.SubmitPayment)
	payments.Get("/:participant_id/public", r.PaymentHandler.GetPaymentPage)
}

func (r *Router) setupContactGroupRoutes(group fiber.Router) {
	contactGroups := group.Group("/contact-groups")
	contactGroups.Post("/", r.ContactGroupHandler.Create)
	contactGroups.Get("/", r.ContactGroupHandler.GetGroups)
	contactGroups.Put("/:id", r.ContactGroupHandler.Update)
	contactGroups.Delete("/:id", r.ContactGroupHandler.Delete)
}

func (r *Router) setupContactRoutes(group fiber.Router) {
	contacts := group.Group("/contacts")
	contacts.Post("/", r.ContactHandler.Create)
	contacts.Get("/", r.ContactHandler.GetContacts)
	contacts.Get("/:id", r.ContactHandler.GetByID)
	contacts.Put("/:id", r.ContactHandler.Update)
	contacts.Delete("/:id", r.ContactHandler.Delete)
	contacts.Post("/bulk", r.ContactHandler.BulkOperations)
	contacts.Post("/import", r.ContactHandler.ImportContacts)
}

func (r *Router) setupSessionRoutes(group fiber.Router) {
	sessions := group.Group("/sessions")
	sessions.Post("/", r.SessionHandler.Create)
	sessions.Get("/", r.SessionHandler.GetSessions)
	sessions.Get("/:id", r.SessionHandler.GetByID)
	sessions.Put("/:id", r.SessionHandler.Update)
	sessions.Delete("/:id", r.SessionHandler.Delete)
	sessions.Post("/:id/participants", r.SessionHandler.AddParticipants)
	sessions.Delete("/:id/participants/:participant_id", r.SessionHandler.RemoveParticipant)
	sessions.Put("/:id/participants/:participant_id", r.SessionHandler.UpdateParticipant)
	sessions.Post("/:id/bills", r.SessionHandler.AddBillItem)
	sessions.Put("/:id/bills/:bill_item_id", r.SessionHandler.UpdateBillItem)
	sessions.Delete("/:id/bills/:bill_item_id", r.SessionHandler.DeleteBillItem)
	sessions.Put("/:id/calculate-splits", r.SessionHandler.CalculateSplits)
}

func (r *Router) setupProtectedPaymentRoutes(group fiber.Router) {
	// Protected payment routes (auth required)
	payments := group.Group("/payments")
	payments.Post("/:proof_id/approve", r.PaymentHandler.ApprovePayment)
	payments.Post("/:proof_id/reject", r.PaymentHandler.RejectPayment)
	payments.Post("/bulk-approve", r.PaymentHandler.BulkApprove)
	payments.Post("/bulk-reject", r.PaymentHandler.BulkReject)
}

func (r *Router) setupNotificationRoutes(group fiber.Router) {
	// Notification routes
	group.Post("/sessions/:id/send-notifications", r.NotificationHandler.SendNotifications)
	group.Post("/sessions/:id/bulk-reminder", r.NotificationHandler.BulkReminder)
	group.Post("/participants/:participant_id/reminder", middleware.ReminderRateLimiter(r.rateLimitService), r.NotificationHandler.SendReminder)
}

func (r *Router) setupPaymentRoutes(group fiber.Router) {
	payments := group.Group("/payments")
	payments.Get("/:participant_id/public", r.PaymentHandler.GetPaymentPage)
}

func (r *Router) setupDashboardRoutes(group fiber.Router) {
	group.Get("/dashboard", r.DashboardHandler.GetDashboard)
}
