package auth

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	FullName  *string `json:"full_name" validate:"omitempty,max=100"`
	AvatarURL *string `json:"avatar_url" validate:"omitempty,url"`
}

// UpdateProfileResponse represents response after updating profile
type UpdateProfileResponse struct {
	UserID         string `json:"user_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name"`
	AvatarURL      string `json:"avatar_url"`
	UpdatedAt      string `json:"updated_at"`
}

// UpdateProfileUseCase handles updating user profile
type UpdateProfileUseCase struct {
	userRepo ports.UserRepository
}

// NewUpdateProfileUseCase creates a new update profile use case
func NewUpdateProfileUseCase(userRepo ports.UserRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{
		userRepo: userRepo,
	}
}

// Execute updates a user's profile
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, userID string, req *UpdateProfileRequest) (*UpdateProfileResponse, error) {
	user, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &UpdateProfileResponse{
		UserID:         user.UserID.String(),
		WhatsAppNumber: user.WhatsAppNumber,
		FullName:       user.FullName,
		AvatarURL:      user.AvatarURL,
		UpdatedAt:      user.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
