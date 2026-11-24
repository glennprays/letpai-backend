package auth

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// ResetPasswordRequest represents the request to complete a password reset:
// the WhatsApp number, the 6-digit code sent by ForgotPassword, and the new
// password.
type ResetPasswordRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
	OTPCode        string `json:"otp_code" validate:"required,len=6"`
	NewPassword    string `json:"new_password" validate:"required"`
}

// ResetPasswordResponse represents the response after a successful reset.
// The response is intentionally minimal; the user then signs in normally.
type ResetPasswordResponse struct {
	Message string `json:"message"`
}

// ResetPasswordUseCase completes the password-reset flow started by
// ForgotPassword: it consumes the one-time WhatsApp code to prove ownership
// of the number, then sets the new password. This closes the loop the
// forgot-password half built — the UI shows a code entry, not a magic link.
type ResetPasswordUseCase struct {
	userRepo    ports.UserRepository
	otpRepo     ports.OTPRepository
	otpSvc      *service.OTPService
	passwordSvc *service.PasswordService
}

// NewResetPasswordUseCase creates a new reset-password use case.
func NewResetPasswordUseCase(
	userRepo ports.UserRepository,
	otpRepo ports.OTPRepository,
	otpSvc *service.OTPService,
	passwordSvc *service.PasswordService,
) *ResetPasswordUseCase {
	return &ResetPasswordUseCase{
		userRepo:    userRepo,
		otpRepo:     otpRepo,
		otpSvc:      otpSvc,
		passwordSvc: passwordSvc,
	}
}

// Execute verifies the reset code and updates the user's password. Error
// semantics mirror VerifyOTP so callers can show the same messages.
func (uc *ResetPasswordUseCase) Execute(ctx context.Context, req *ResetPasswordRequest) (*ResetPasswordResponse, error) {
	if err := uc.passwordSvc.ValidatePassword(req.NewPassword); err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	if !uc.otpSvc.ValidateCode(req.OTPCode) {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid OTP code format"))
	}

	otp, err := uc.otpRepo.FindValidByPhone(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("no valid OTP found"))
	}
	if otp.IsExpired() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("OTP has expired"))
	}
	if otp.IsUsed {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("OTP has already been used"))
	}
	if otp.OTPCode != req.OTPCode {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid OTP code"))
	}

	if err := uc.otpRepo.MarkAsUsed(ctx, otp.OTPID.String()); err != nil {
		return nil, err
	}

	user, err := uc.userRepo.FindByWhatsApp(ctx, req.WhatsAppNumber)
	if err != nil || user == nil {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("account not found"))
	}

	hashedPassword, err := uc.passwordSvc.Hash(req.NewPassword)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}
	user.PasswordHash = hashedPassword
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// The reset consumed its code; stale codes for this number must not
	// linger as valid.
	_ = uc.otpRepo.InvalidatePreviousOTPs(ctx, req.WhatsAppNumber)

	return &ResetPasswordResponse{
		Message: "Password updated. You can now sign in with your new password.",
	}, nil
}
