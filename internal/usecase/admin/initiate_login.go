package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// LoginRequest represents admin login request (password-based)
type LoginRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	Password       string `json:"password" validate:"required,min=8,max=100"`
}

// InitiateLoginRequest represents initiate login request
type InitiateLoginRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
}

// InitiateLoginResult represents login initiation result.
// Fields mirror the InitiateLoginResponse schema in docs/swagger.yaml.
type InitiateLoginResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

// InitiateLoginUseCase handles initiating admin login flow
type InitiateLoginUseCase struct {
	adminRepo   ports.AdminRepository
	otpRepo     ports.OTPRepository
	otpSvc      *service.OTPService
	whatsappSvc *service.WhatsAppService
}

// NewInitiateLoginUseCase creates a new initiate login use case
func NewInitiateLoginUseCase(
	adminRepo ports.AdminRepository,
	otpRepo ports.OTPRepository,
	otpSvc *service.OTPService,
	whatsappSvc *service.WhatsAppService,
) *InitiateLoginUseCase {
	return &InitiateLoginUseCase{
		adminRepo:   adminRepo,
		otpRepo:     otpRepo,
		otpSvc:      otpSvc,
		whatsappSvc: whatsappSvc,
	}
}

// Execute initiates login flow by generating and storing OTP
func (uc *InitiateLoginUseCase) Execute(ctx context.Context, req *InitiateLoginRequest) (*InitiateLoginResult, error) {
	// Check if admin exists
	exists, err := uc.adminRepo.CheckExistsByWhatsAppNumber(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to check admin: %w", err))
	}

	if !exists {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("admin not found"))
	}

	// Invalidate previous OTPs
	if err := uc.otpRepo.InvalidatePreviousOTPs(ctx, req.WhatsAppNumber); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to invalidate previous OTPs: %w", err))
	}

	// Generate new OTP
	otpCode, err := uc.otpSvc.Generate()
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to generate OTP: %w", err))
	}

	// Create OTP verification record
	otpRecord := entity.NewOTPVerification(req.WhatsAppNumber, otpCode, uc.otpSvc.GetExpiry())
	if err := uc.otpRepo.Create(ctx, otpRecord); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to create OTP: %w", err))
	}

	// Send OTP via WhatsApp gateway (best effort — OTP is already persisted,
	// so the admin can retry verification if delivery glitches).
	message := uc.otpSvc.FormatWhatsAppMessage(otpCode)
	_, _ = uc.whatsappSvc.SendOTP(ctx, req.WhatsAppNumber, message)

	return &InitiateLoginResult{
		Success:   true,
		Message:   "Login initiated",
		SessionID: otpRecord.OTPID.String(),
		ExpiresAt: otpRecord.ExpiresAt.Format(time.RFC3339),
	}, nil
}

// VerifyOTPRequest represents OTP verification request
type VerifyOTPRequest struct {
	SessionID string `json:"session_id" validate:"required"`
	OTPCode   string `json:"otp_code" validate:"required,len=6"`
}

// VerifyOTPResult represents verification result
type VerifyOTPResult struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// VerifyOTPUseCase handles OTP verification for admin login
type VerifyOTPUseCase struct {
	adminRepo  ports.AdminRepository
	otpRepo    ports.OTPRepository
	jwtService *service.JWTService
}

// NewVerifyOTPUseCase creates a new verify OTP use case
func NewVerifyOTPUseCase(
	adminRepo ports.AdminRepository,
	otpRepo ports.OTPRepository,
	jwtService *service.JWTService,
) *VerifyOTPUseCase {
	return &VerifyOTPUseCase{
		adminRepo:  adminRepo,
		otpRepo:    otpRepo,
		jwtService: jwtService,
	}
}

// Execute verifies OTP and returns JWT token
func (uc *VerifyOTPUseCase) Execute(ctx context.Context, req *VerifyOTPRequest) (*VerifyOTPResult, error) {
	// Find OTP by ID
	otpRecord, err := uc.otpRepo.FindByID(ctx, req.SessionID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("invalid or expired OTP"))
	}

	if otpRecord == nil {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid or expired OTP"))
	}

	if otpRecord.IsUsed {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("OTP has already been used"))
	}

	if otpRecord.IsExpired() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("OTP has expired"))
	}

	if otpRecord.OTPCode != req.OTPCode {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid OTP code"))
	}

	// Mark OTP as used
	if err := uc.otpRepo.MarkAsUsed(ctx, otpRecord.OTPID.String()); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to mark OTP as used: %w", err))
	}

	// Find admin by WhatsApp number
	admin, err := uc.adminRepo.FindByWhatsAppNumber(ctx, otpRecord.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	// Update last login
	admin.UpdateLastLogin()
	if err := uc.adminRepo.Update(ctx, admin); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update last login: %w", err))
	}

	// Generate JWT token with role
	token, err := uc.jwtService.GenerateTokenWithRole(admin.AdminID.String(), admin.WhatsAppNumber, admin.Role)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to generate token: %w", err))
	}

	return &VerifyOTPResult{
		Token:     token,
		ExpiresAt: otpRecord.ExpiresAt.Format(time.RFC3339),
	}, nil
}
