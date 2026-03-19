package auth

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain/errors"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// VerifyOTPRequest represents the request to verify OTP
type VerifyOTPRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
	OTPCode        string `json:"otp_code" validate:"required,len=6"`
}

// VerifyOTPResponse represents the response after OTP verification
type VerifyOTPResponse struct {
	Token string          `json:"token"`
	User  *UserResponse   `json:"user"`
}

// UserResponse represents user data in responses
type UserResponse struct {
	UserID         string `json:"user_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name,omitempty"`
}

// VerifyOTPUseCase handles OTP verification
type VerifyOTPUseCase struct {
	userRepo    ports.UserRepository
	otpRepo     ports.OTPRepository
	jwtSvc      *service.JWTService
	otpSvc      *service.OTPService
}

// NewVerifyOTPUseCase creates a new verify OTP use case
func NewVerifyOTPUseCase(
	userRepo ports.UserRepository,
	otpRepo ports.OTPRepository,
	jwtSvc *service.JWTService,
	otpSvc *service.OTPService,
) *VerifyOTPUseCase {
	return &VerifyOTPUseCase{
		userRepo: userRepo,
		otpRepo:  otpRepo,
		jwtSvc:   jwtSvc,
		otpSvc:   otpSvc,
	}
}

// Execute verifies the OTP and returns JWT token
func (uc *VerifyOTPUseCase) Execute(ctx context.Context, req *VerifyOTPRequest) (*VerifyOTPResponse, error) {
	// Validate OTP code format
	if !uc.otpSvc.ValidateCode(req.OTPCode) {
		return nil, errors.NewError(errors.ErrBadRequest, errors.New("invalid OTP code format"))
	}

	// Find valid OTP for this phone number
	otp, err := uc.otpRepo.FindValidByPhone(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, errors.NewError(errors.ErrNotFound, errors.New("no valid OTP found"))
	}

	// Check if OTP is expired
	if otp.IsExpired() {
		return nil, errors.NewError(errors.ErrBadRequest, errors.New("OTP has expired"))
	}

	// Check if OTP is already used
	if otp.IsUsed {
		return nil, errors.NewError(errors.ErrBadRequest, errors.New("OTP has already been used"))
	}

	// Verify OTP code
	if otp.OTPCode != req.OTPCode {
		return nil, errors.NewError(errors.ErrBadRequest, errors.New("invalid OTP code"))
	}

	// Mark OTP as used
	if err := uc.otpRepo.MarkAsUsed(ctx, otp.OTPID.String()); err != nil {
		return nil, err
	}

	// Find user
	user, err := uc.userRepo.FindByWhatsApp(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, err
	}

	// Mark user as verified
	user.MarkVerified()
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Generate JWT token
	token, err := uc.jwtSvc.GenerateToken(user.UserID.String(), user.WhatsAppNumber)
	if err != nil {
		return nil, errors.NewError(errors.ErrInternalFailure, err)
	}

	return &VerifyOTPResponse{
		Token: token,
		User: &UserResponse{
			UserID:         user.UserID.String(),
			WhatsAppNumber: user.WhatsAppNumber,
			FullName:       user.FullName,
		},
	}, nil
}
