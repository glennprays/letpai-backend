//go:build wireinject
// +build wireinject

//go:generate go run github.com/google/wire/cmd/wire

package infrastructure

import (
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/handler"
	"github.com/glennprays/letpai-backend/internal/repository"
	"github.com/glennprays/letpai-backend/internal/router"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/letpai-backend/internal/usecase/admin"
	"github.com/glennprays/letpai-backend/internal/usecase/auth"
	"github.com/glennprays/letpai-backend/internal/usecase/billing"
	"github.com/glennprays/letpai-backend/internal/usecase/contact"
	"github.com/glennprays/letpai-backend/internal/usecase/contactgroup"
	"github.com/glennprays/letpai-backend/internal/usecase/dashboard"
	"github.com/glennprays/letpai-backend/internal/usecase/notification"
	"github.com/glennprays/letpai-backend/internal/usecase/participant"
	"github.com/glennprays/letpai-backend/internal/usecase/payment"
	"github.com/glennprays/letpai-backend/internal/usecase/session"
	"github.com/glennprays/letpai-backend/pkg/logger"
)

var CoreSet = wire.NewSet(
	config.Load,
	logger.ProviderLogger,
	NewPostgresConnection,
	NewRedisConnection,
)

var RepositorySet = wire.NewSet(
	repository.NewPostgresUserRepository,
	repository.NewPostgresOTPRepository,
	repository.NewPostgresContactGroupRepository,
	repository.NewPostgresContactRepository,
	repository.NewPostgresSessionRepository,
	repository.NewPostgresParticipantRepository,
	repository.NewPostgresBillItemRepository,
	repository.NewPostgresNotificationLogRepository,
	repository.NewPostgresWhatsAppConfigRepository,
	repository.NewPostgresAdminRepository,
)

var ServiceSet = wire.NewSet(
	NewJWTService,
	NewOTPService,
	NewPasswordService,
	NewWhatsAppService,
	NewImageServiceProvider,
	NewRateLimitService,
)

var UseCaseSet = wire.NewSet(
	// Auth use cases
	auth.NewRegisterUserUseCase,
	auth.NewVerifyOTPUseCase,
	auth.NewLoginUserUseCase,
	auth.NewLogoutUserUseCase,
	auth.NewUpdateProfileUseCase,
	// Admin use cases
	admin.NewInitiateLoginUseCase,
	admin.NewVerifyOTPUseCase,
	admin.NewGetProfileUseCase,
	admin.NewSetupPasswordUseCase,
	admin.NewListAdminsUseCase,
	admin.NewCreateAdminUseCase,
	admin.NewUpdateAdminUseCase,
	admin.NewDeleteAdminUseCase,
	admin.NewGetStatusUseCase,
	admin.NewGetQRCodeUseCase,
	admin.NewLogoutUseCase,
	admin.NewUpdateConfigUseCase,
	// Contact use cases
	contact.NewCreateContactUseCase,
	contact.NewGetContactsUseCase,
	contact.NewGetContactByIDUseCase,
	contact.NewUpdateContactUseCase,
	contact.NewDeleteContactUseCase,
	contact.NewBulkOperationsUseCase,
	contact.NewImportContactsUseCase,
	// Contact group use cases
	contactgroup.NewCreateGroupUseCase,
	contactgroup.NewGetGroupsUseCase,
	contactgroup.NewUpdateGroupUseCase,
	contactgroup.NewDeleteGroupUseCase,
	// Session use cases
	session.NewCreateSessionUseCase,
	session.NewGetSessionsUseCase,
	session.NewGetSessionDetailUseCase,
	session.NewUpdateSessionUseCase,
	session.NewCancelSessionUseCase,
	// Participant use cases
	participant.NewAddParticipantsUseCase,
	participant.NewRemoveParticipantUseCase,
	participant.NewUpdateParticipantUseCase,
	participant.NewImportFromGroupUseCase,
	// Billing use cases
	billing.NewAddBillItemUseCase,
	billing.NewUpdateBillItemUseCase,
	billing.NewDeleteBillItemUseCase,
	billing.NewCalculateSplitsUseCase,
	// Payment use cases
	payment.NewSubmitPaymentUseCase,
	payment.NewApprovePaymentUseCase,
	payment.NewRejectPaymentUseCase,
	payment.NewBulkApproveUseCase,
	payment.NewBulkRejectUseCase,
	payment.NewGetPaymentPageUseCase,
	payment.NewGetPaymentProofUseCase,
	// Notification use cases
	notification.NewSendNotificationsUseCase,
	notification.NewSendReminderUseCase,
	notification.NewBulkReminderUseCase,
	// Dashboard use cases
	dashboard.NewGetDashboardUseCase,
)

var HandlerSet = wire.NewSet(
	handler.NewHealthHandler,
	handler.NewAuthHandler,
	handler.NewAdminHandler,
	handler.NewContactGroupHandler,
	handler.NewContactHandler,
	handler.NewSessionHandler,
	handler.NewPaymentHandler,
	handler.NewNotificationHandler,
	handler.NewWebhookHandler,
	handler.NewDashboardHandler,
)

var ApiSet = wire.NewSet(
	HandlerSet,
	router.NewRouter,
)

// NewJWTService creates a new JWT service with config values
func NewJWTService(cfg *config.Config) *service.JWTService {
	return service.NewJWTService(
		cfg.JWTSecret,
		cfg.JWTExpiryHours,
	)
}

// NewOTPService creates a new OTP service with config values
func NewOTPService(cfg *config.Config) *service.OTPService {
	return service.NewOTPService(
		cfg.OTPExpiryMinutes,
	)
}

// NewPasswordService creates a new password service
func NewPasswordService() *service.PasswordService {
	return service.NewPasswordService(12) // bcrypt cost
}

// NewWhatsAppService creates a new WhatsApp service with config values
func NewWhatsAppService(cfg *config.Config, configRepo ports.WhatsAppConfigRepository) *service.WhatsAppService {
	return service.NewWhatsAppService(
		cfg.WhatsAppGatewayURL,
		cfg.WhatsAppAPIKey,
		configRepo,
	)
}

// NewImageService creates a new image service with AWS S3 configuration
func NewImageService(cfg *config.Config) (*service.ImageService, error) {
	imageCfg := &service.ImageConfig{
		AWSEndpoint: cfg.AWSEndpoint,
		AWSRegion:   cfg.AWSRegion,
		AWSAccessID: cfg.AWSAccessID,
		AWSSecret:   cfg.AWSSecret,
		BucketName:  cfg.S3BucketName,
		CDNURL:      cfg.CDNURL,
		EnableWebP:  cfg.EnableWebP,
		WebPQuality: cfg.WebPQuality,
		MaxFileSize: int64(cfg.MaxImageSizeMB) * 1024 * 1024, // Convert MB to bytes
	}
	return service.NewImageService(imageCfg)
}

// NewRateLimitService creates a new rate limit service with Redis client
func NewRateLimitService(redisClient *redis.Client) *service.RateLimitService {
	return service.NewRateLimitService(redisClient)
}

// NewImageServiceProvider creates a new image service provider that handles initialization errors
func NewImageServiceProvider(cfg *config.Config) (*service.ImageService, error) {
	return NewImageService(cfg)
}

// InitializeApp creates the application and injects all dependencies
func InitializeApp() (*App, error) {
	wire.Build(
		CoreSet,
		RepositorySet,
		ServiceSet,
		UseCaseSet,
		ApiSet,
		wire.Struct(new(App), "*"),
	)
	return &App{}, nil
}
