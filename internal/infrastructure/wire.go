//go:build wireinject
// +build wireinject

package infrastructure

import (
	"github.com/google/wire"
	"github.com/jmoiron/sqlx"

	"github.com/glennprays/letpai-backend/config"
	"github.com/glennprays/letpai-backend/internal/handler"
	"github.com/glennprays/letpai-backend/internal/middleware"
	"github.com/glennprays/letpai-backend/internal/repository"
	"github.com/glennprays/letpai-backend/internal/router"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/letpai-backend/internal/usecase/auth"
	"github.com/glennprays/letpai-backend/pkg/logger"
)

var CoreSet = wire.NewSet(
	config.Load,
	logger.ProviderLogger,
	NewPostgresConnection,
)

var RepositorySet = wire.NewSet(
	repository.NewPostgresUserRepository,
	repository.NewPostgresOTPRepository,
)

var ServiceSet = wire.NewSet(
	NewJWTService,
	NewOTPService,
	NewPasswordService,
	NewWhatsAppService,
)

var UseCaseSet = wire.NewSet(
	auth.NewRegisterUserUseCase,
	auth.NewVerifyOTPUseCase,
	auth.NewLoginUserUseCase,
	auth.NewLogoutUserUseCase,
)

var HandlerSet = wire.NewSet(
	handler.NewHealthHandler,
	handler.NewAuthHandler,
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
func NewWhatsAppService(cfg *config.Config) *service.WhatsAppService {
	return service.NewWhatsAppService(
		cfg.WhatsAppGatewayURL,
		cfg.WhatsAppAPIKey,
	)
}

func InitializeApp() (*App, error) {
	wire.Build(
		CoreSet,
		RepositorySet,
		ServiceSet,
		UseCaseSet,
		ApiSet,
		middleware.Authenticate,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
