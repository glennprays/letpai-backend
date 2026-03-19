package auth

import (
	"context"

	"github.com/glennprays/letpai-backend/internal/service"
)

// LogoutRequest represents the request to logout
type LogoutRequest struct {
	Token string `json:"-"`
}

// LogoutResponse represents the response after logout
type LogoutResponse struct {
	Message string `json:"message"`
}

// LogoutUserUseCase handles user logout
// For MVP v1, logout is a client-side operation (token removal)
// In future versions, we may implement token blacklisting
type LogoutUserUseCase struct {
	jwtSvc *service.JWTService
}

// NewLogoutUserUseCase creates a new logout user use case
func NewLogoutUserUseCase(jwtSvc *service.JWTService) *LogoutUserUseCase {
	return &LogoutUserUseCase{
		jwtSvc: jwtSvc,
	}
}

// Execute logs out the user
func (uc *LogoutUserUseCase) Execute(ctx context.Context, req *LogoutRequest) (*LogoutResponse, error) {
	// For MVP v1, logout is handled client-side by removing the token
	// In future versions, we can implement token blacklisting with Redis

	return &LogoutResponse{
		Message: "Logged out successfully",
	}, nil
}
