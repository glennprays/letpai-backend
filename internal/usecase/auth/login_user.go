package auth

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain/errors"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// LoginRequest represents the request to login
type LoginRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required"`
	Password       string `json:"password" validate:"required"`
}

// LoginResponse represents the response after successful login
type LoginResponse struct {
	Token string        `json:"token"`
	User  *UserResponse `json:"user"`
}

// LoginUserUseCase handles user login
type LoginUserUseCase struct {
	userRepo    ports.UserRepository
	passwordSvc *service.PasswordService
	jwtSvc      *service.JWTService
}

// NewLoginUserUseCase creates a new login user use case
func NewLoginUserUseCase(
	userRepo ports.UserRepository,
	passwordSvc *service.PasswordService,
	jwtSvc *service.JWTService,
) *LoginUserUseCase {
	return &LoginUserUseCase{
		userRepo:    userRepo,
		passwordSvc: passwordSvc,
		jwtSvc:      jwtSvc,
	}
}

// Execute authenticates the user and returns JWT token
func (uc *LoginUserUseCase) Execute(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Find user by WhatsApp number
	user, err := uc.userRepo.FindByWhatsApp(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, errors.NewError(errors.ErrUnauthorized, errors.New("invalid credentials"))
	}

	// Check if user is verified
	if !user.IsVerified {
		return nil, errors.NewError(errors.ErrUnauthorized, errors.New("please verify your account first"))
	}

	// Verify password
	if !uc.passwordSvc.Verify(req.Password, user.PasswordHash) {
		return nil, errors.NewError(errors.ErrUnauthorized, errors.New("invalid credentials"))
	}

	// Generate JWT token
	token, err := uc.jwtSvc.GenerateToken(user.UserID.String(), user.WhatsAppNumber)
	if err != nil {
		return nil, errors.NewError(errors.ErrInternalFailure, err)
	}

	return &LoginResponse{
		Token: token,
		User: &UserResponse{
			UserID:         user.UserID.String(),
			WhatsAppNumber: user.WhatsAppNumber,
			FullName:       user.FullName,
		},
	}, nil
}
