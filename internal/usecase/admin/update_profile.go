package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateProfileRequest is the body for PUT /admin/profile (self-edit).
// Only fields safe for an admin to change on their own row live here —
// role, is_active, and whatsapp_number are intentionally absent.
type UpdateProfileRequest struct {
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
}

// UpdateProfileResult mirrors the AdminProfile DTOs other admin
// endpoints return so the FE can swap it in without conditional shapes.
type UpdateProfileResult struct {
	AdminID  string `json:"admin_id"`
	FullName string `json:"full_name"`
	Message  string `json:"message"`
}

// UpdateProfileUseCase lets the calling admin rename themselves
// without granting super-admin privileges. The handler must source
// adminID from middleware.GetUserID so a caller can only ever edit
// their own row.
type UpdateProfileUseCase struct {
	adminRepo ports.AdminRepository
}

// NewUpdateProfileUseCase wires the use case.
func NewUpdateProfileUseCase(adminRepo ports.AdminRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{adminRepo: adminRepo}
}

// Execute updates the admin's display name.
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, adminID string, req *UpdateProfileRequest) (*UpdateProfileResult, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}
	if admin == nil {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("admin not found"))
	}

	admin.UpdateProfile(req.FullName)

	if err := uc.adminRepo.Update(ctx, admin); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update profile: %w", err))
	}

	return &UpdateProfileResult{
		AdminID:  admin.AdminID.String(),
		FullName: admin.FullName,
		Message:  "Profile updated",
	}, nil
}
