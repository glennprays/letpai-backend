package auth

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/errors"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// RegisterUserRequest represents the request to register a new user
type RegisterUserRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
	Password       string `json:"password" validate:"required,min=8"`
}

// RegisterUserResponse represents the response after user registration
type RegisterUserResponse struct {
	UserID    string    `json:"user_id"`
	ExpiresAt string    `json:"expires_at"`
}

// RegisterUserUseCase handles user registration with OTP verification
type RegisterUserUseCase struct {
	userRepo       ports.UserRepository
	otpRepo        ports.OTPRepository
	passwordSvc    *service.PasswordService
	otpSvc         *service.OTPService
	whatsappSvc    *service.WhatsAppService
}

// NewRegisterUserUseCase creates a new register user use case
func NewRegisterUserUseCase(
	userRepo ports.UserRepository,
	otpRepo ports.OTPRepository,
	passwordSvc *service.PasswordService,
	otpSvc *service.OTPService,
	whatsappSvc *service.WhatsAppService,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		userRepo:    userRepo,
		otpRepo:     otpRepo,
		passwordSvc: passwordSvc,
		otpSvc:      otpSvc,
		whatsappSvc: whatsappSvc,
	}
}

// Execute registers a new user and sends OTP
func (uc *RegisterUserUseCase) Execute(ctx context.Context, req *RegisterUserRequest) (*RegisterUserResponse, error) {
	// Validate password
	if err := uc.passwordSvc.ValidatePassword(req.Password); err != nil {
		return nil, errors.NewError(errors.ErrBadRequest, err)
	}

	// Check if user already exists
	exists, err := uc.userRepo.IsExists(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.NewError(errors.ErrConflict, errors.New("user already exists"))
	}

	// Hash password
	hashedPassword, err := uc.passwordSvc.Hash(req.Password)
	if err != nil {
		return nil, errors.NewError(errors.ErrInternalFailure, err)
	}

	// Create user
	user := entity.NewUser(req.WhatsAppNumber, hashedPassword)
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate OTP
	otpCode, err := uc.otpSvc.Generate()
	if err != nil {
		return nil, errors.NewError(errors.ErrInternalFailure, err)
	}

	// Create OTP verification
	otp := entity.NewOTPVerificationWithUser(user.UserID, req.WhatsAppNumber, otpCode, uc.otpSvc.GetExpiry())

	// Invalidate previous OTPs
	_ = uc.otpRepo.InvalidatePreviousOTPs(ctx, req.WhatsAppNumber)

	// Save new OTP
	if err := uc.otpRepo.Create(ctx, otp); err != nil {
		return nil, err
	}

	// Send OTP via WhatsApp
	message := uc.otpSvc.FormatWhatsAppMessage(otpCode)
	if err := uc.whatsappSvc.SendOTP(ctx, req.WhatsAppNumber, message); err != nil {
		// Log error but don't fail registration
		// OTP is saved, user can retry verification
	}

	return &RegisterUserResponse{
		UserID:    user.UserID.String(),
		ExpiresAt: otp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
