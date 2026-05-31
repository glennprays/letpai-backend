package router

import (
	"github.com/glennprays/letpai-backend/internal/handler"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/log"
	"github.com/gofiber/fiber/v2"
)

type Router struct {
	logger                 *log.Logger
	HealthHandler          *handler.HealthHandler
	AuthHandler            *handler.AuthHandler
	AdminHandler           *handler.AdminHandler
	AdminTemplatesHandler  *handler.AdminTemplatesHandler
	WhatsAppWebhookHandler *handler.WhatsAppWebhookHandler
	ContactGroupHandler    *handler.ContactGroupHandler
	ContactHandler         *handler.ContactHandler
	SessionHandler         *handler.SessionHandler
	PaymentHandler         *handler.PaymentHandler
	NotificationHandler    *handler.NotificationHandler
	WebhookHandler         *handler.WebhookHandler
	DashboardHandler       *handler.DashboardHandler
	jwtService             *service.JWTService
	rateLimitService       *service.RateLimitService
}

func NewRouter(
	logger *log.Logger,
	healthHandler *handler.HealthHandler,
	authHandler *handler.AuthHandler,
	adminHandler *handler.AdminHandler,
	adminTemplatesHandler *handler.AdminTemplatesHandler,
	whatsappWebhookHandler *handler.WhatsAppWebhookHandler,
	contactGroupHandler *handler.ContactGroupHandler,
	contactHandler *handler.ContactHandler,
	sessionHandler *handler.SessionHandler,
	paymentHandler *handler.PaymentHandler,
	notificationHandler *handler.NotificationHandler,
	webhookHandler *handler.WebhookHandler,
	dashboardHandler *handler.DashboardHandler,
	jwtService *service.JWTService,
	rateLimitService *service.RateLimitService,
) *Router {
	routerLogger := logger.With(log.String("component", "router"))
	return &Router{
		logger:                 routerLogger,
		HealthHandler:          healthHandler,
		AuthHandler:            authHandler,
		AdminHandler:           adminHandler,
		AdminTemplatesHandler:  adminTemplatesHandler,
		WhatsAppWebhookHandler: whatsappWebhookHandler,
		ContactGroupHandler:    contactGroupHandler,
		ContactHandler:         contactHandler,
		SessionHandler:         sessionHandler,
		PaymentHandler:         paymentHandler,
		NotificationHandler:    notificationHandler,
		WebhookHandler:         webhookHandler,
		DashboardHandler:       dashboardHandler,
		jwtService:             jwtService,
		rateLimitService:       rateLimitService,
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
	r.setupAdminRoutes(v1)
	r.setupPublicPaymentRoutes(v1)

	// Protected routes (require auth)
	protected := v1.Use(middleware.Authenticate(r.jwtService))
	r.setupProtectedAuthRoutes(protected)
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
	webhooks.Post("/whatsapp-gateway", r.WhatsAppWebhookHandler.HandleStatusUpdate)
}

func (r *Router) setupAuthRoutes(group fiber.Router) {
	auth := group.Group("/auth")
	auth.Post("/register", middleware.RegisterRateLimiter(r.rateLimitService), r.AuthHandler.Register)
	auth.Post("/verify-otp", middleware.VerifyOTPRateLimiter(r.rateLimitService), r.AuthHandler.VerifyOTP)
	auth.Post("/login", middleware.LoginRateLimiter(r.rateLimitService), r.AuthHandler.Login)
	auth.Post("/forgot-password", middleware.LoginRateLimiter(r.rateLimitService), r.AuthHandler.ForgotPassword)
}

// Auth routes that require a valid token (logout, profile editing).
func (r *Router) setupProtectedAuthRoutes(group fiber.Router) {
	auth := group.Group("/auth")
	auth.Post("/logout", r.AuthHandler.Logout)
	auth.Put("/profile", r.AuthHandler.UpdateProfile)
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
	// Multi-account bank info. The legacy single-row fields on
	// PUT /sessions/:id still work during the compat window;
	// /bank-accounts is the canonical write path going forward.
	sessions.Put("/:id/bank-accounts", r.SessionHandler.ReplaceBankAccounts)
	// Bill image attachments
	sessions.Post("/:id/bill-images", r.SessionHandler.UploadBillImage)
	sessions.Get("/:id/bill-images", r.SessionHandler.GetBillImages)
	sessions.Get("/:id/bill-images/:image_id", r.SessionHandler.GetBillImageSignedUrl)
	sessions.Delete("/:id/bill-images/:image_id", r.SessionHandler.DeleteBillImage)
	// Fee configuration (service charge & tax percentages)
	sessions.Put("/:id/fee-config", r.SessionHandler.UpdateFeeConfig)
}

func (r *Router) setupProtectedPaymentRoutes(group fiber.Router) {
	// Protected payment routes (auth required)
	payments := group.Group("/payments")
	payments.Post("/:proof_id/approve", r.PaymentHandler.ApprovePayment)
	payments.Post("/:proof_id/reject", r.PaymentHandler.RejectPayment)
	payments.Post("/bulk-approve", r.PaymentHandler.BulkApprove)
	payments.Post("/bulk-reject", r.PaymentHandler.BulkReject)

	// Host can manually mark a participant as paid without a proof
	// upload (e.g. cash payments). 409 if the participant is already paid.
	group.Post("/participants/:participant_id/mark-paid", r.PaymentHandler.MarkPaidWithoutProof)
}

func (r *Router) setupNotificationRoutes(group fiber.Router) {
	// Notification routes
	group.Post("/sessions/:id/send-notifications", r.NotificationHandler.SendNotifications)
	// Escape-hatch: bypasses the dirty-for-notify check. Same auth.
	group.Post("/sessions/:id/send-notifications/resend", r.NotificationHandler.ResendNotifications)
	group.Post("/sessions/:id/bulk-reminder", r.NotificationHandler.BulkReminder)
	group.Post("/participants/:participant_id/reminder", middleware.ReminderRateLimiter(r.rateLimitService), r.NotificationHandler.SendReminder)
	group.Get("/participants/:participant_id/reminder-status", r.NotificationHandler.ReminderStatus)
	// Retry the most recent notification for one participant.
	// NOT gated by the session-level dirty predicate — that gate
	// exists to stop spam-resends of the batch; individual retry of
	// a failed delivery is exactly the case we want to allow.
	group.Post("/participants/:participant_id/notifications/retry", r.NotificationHandler.RetryNotification)
}

func (r *Router) setupDashboardRoutes(group fiber.Router) {
	group.Get("/dashboard", r.DashboardHandler.GetDashboard)
}

func (r *Router) setupAdminRoutes(group fiber.Router) {
	// Admin authentication routes
	admin := group.Group("/admin")
	admin.Get("/auth/needs-setup", r.AdminHandler.NeedsSetup)
	admin.Post("/auth/bootstrap", r.AdminHandler.Bootstrap)
	admin.Post("/auth/initiate", r.AdminHandler.InitiateLogin)
	admin.Post("/auth/login", r.AdminHandler.Login)
	admin.Post("/auth/verify-otp", r.AdminHandler.VerifyOTP)

	// Protected admin routes
	protectedAdmin := admin.Use(middleware.Authenticate(r.jwtService))
	protectedAdmin.Get("/profile", r.AdminHandler.GetProfile)
	protectedAdmin.Put("/profile", r.AdminHandler.UpdateProfile)
	protectedAdmin.Put("/profile/setup-password", r.AdminHandler.SetupPassword)

	// Super admin only routes
	protectedAdmin.Get("/status", r.AdminHandler.GetStatus)
	protectedAdmin.Post("/qr-code", r.AdminHandler.GetQRCode)
	protectedAdmin.Post("/logout", r.AdminHandler.Logout)

	// Admin message-template management (any admin can read; updates
	// available to anyone in the admin scope — narrowing to super
	// admin only would be cheap to add later).
	protectedAdmin.Get("/templates", r.AdminTemplatesHandler.List)
	protectedAdmin.Put("/templates/:key", r.AdminTemplatesHandler.Update)

	// Admin management routes (super admin only)
	superAdmin := protectedAdmin.Use(middleware.RequireSuperAdminRole(r.jwtService))
	superAdmin.Get("/admins", r.AdminHandler.ListAdmins)
	superAdmin.Post("/admins", r.AdminHandler.CreateAdmin)
	superAdmin.Put("/admins/:id", r.AdminHandler.UpdateAdmin)
	superAdmin.Delete("/admins/:id", r.AdminHandler.DeleteAdmin)
}
