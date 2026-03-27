package admin

import (
	"context"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// SetupPasswordRequest represents setup password request
type SetupPasswordRequest struct {
	Password string `json:"password" validate:"required,min=8,max=100"`
}

// SetupPasswordUseCase handles setting admin password
type SetupPasswordUseCase struct {
	adminRepo   ports.AdminRepository
	passwordSvc *service.PasswordService
}

// NewSetupPasswordUseCase creates a new setup password use case
func NewSetupPasswordUseCase(
	adminRepo ports.AdminRepository,
	passwordSvc *service.PasswordService,
) *SetupPasswordUseCase {
	return &SetupPasswordUseCase{
		adminRepo:   adminRepo,
		passwordSvc: passwordSvc,
	}
}

// Execute sets the password for an admin
func (uc *SetupPasswordUseCase) Execute(ctx context.Context, adminID string, req *SetupPasswordRequest) error {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	hashedPassword, err := uc.passwordSvc.Hash(req.Password)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to hash password: %w", err))
	}

	admin.PasswordHash = hashedPassword

	if err := uc.adminRepo.Update(ctx, admin); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update password: %w", err))
	}

	return nil
}
