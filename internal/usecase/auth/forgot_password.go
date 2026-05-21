package auth

import (
	"context"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// ForgotPasswordRequest represents the request to start a password-reset flow.
type ForgotPasswordRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
}

// ForgotPasswordResponse represents the response after initiating a reset.
// The response is intentionally vague (no user_id) to avoid leaking which
// numbers correspond to real accounts.
type ForgotPasswordResponse struct {
	Message   string `json:"message"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// ForgotPasswordUseCase handles sending a one-time code that, paired with
// the existing /auth/verify-otp endpoint, lets a user prove ownership of
// their WhatsApp number ahead of a password reset.
type ForgotPasswordUseCase struct {
	userRepo    ports.UserRepository
	otpRepo     ports.OTPRepository
	otpSvc      *service.OTPService
	whatsappSvc *service.WhatsAppService
}

// NewForgotPasswordUseCase creates a new forgot-password use case.
func NewForgotPasswordUseCase(
	userRepo ports.UserRepository,
	otpRepo ports.OTPRepository,
	otpSvc *service.OTPService,
	whatsappSvc *service.WhatsAppService,
) *ForgotPasswordUseCase {
	return &ForgotPasswordUseCase{
		userRepo:    userRepo,
		otpRepo:     otpRepo,
		otpSvc:      otpSvc,
		whatsappSvc: whatsappSvc,
	}
}

// Execute generates and sends a one-time reset code. To avoid leaking
// account existence, the response is identical whether the number is
// registered or not — but no OTP is created and no WhatsApp message is
// dispatched for unknown numbers.
func (uc *ForgotPasswordUseCase) Execute(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	user, err := uc.userRepo.FindByWhatsApp(ctx, req.WhatsAppNumber)
	if err != nil || user == nil {
		// Same response regardless — don't leak whether the number exists.
		return &ForgotPasswordResponse{
			Message: "If that number is registered, we've sent a verification code.",
		}, nil
	}

	otpCode, err := uc.otpSvc.Generate()
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	otp := entity.NewOTPVerificationWithUser(user.UserID, req.WhatsAppNumber, otpCode, uc.otpSvc.GetExpiry())

	_ = uc.otpRepo.InvalidatePreviousOTPs(ctx, req.WhatsAppNumber)

	if err := uc.otpRepo.Create(ctx, otp); err != nil {
		return nil, err
	}

	// Best-effort delivery — OTP is persisted so the user can retry verify.
	message := uc.otpSvc.FormatWhatsAppMessage(otpCode)
	_, _ = uc.whatsappSvc.SendOTP(ctx, req.WhatsAppNumber, message)

	return &ForgotPasswordResponse{
		Message:   "If that number is registered, we've sent a verification code.",
		ExpiresAt: otp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
