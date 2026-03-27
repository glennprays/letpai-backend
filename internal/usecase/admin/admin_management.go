package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// CreateAdminRequest represents create admin request
type CreateAdminRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	FullName       string `json:"full_name" validate:"required,max=100"`
	Role           string `json:"role" validate:"required,oneof=super_admin,admin"`
}

// CreateAdminResult represents create admin result
type CreateAdminResult struct {
	AdminID string `json:"admin_id"`
	Message string `json:"message"`
}

// CreateAdminUseCase handles creating a new admin
type CreateAdminUseCase struct {
	adminRepo ports.AdminRepository
}

// NewCreateAdminUseCase creates a new create admin use case
func NewCreateAdminUseCase(adminRepo ports.AdminRepository) *CreateAdminUseCase {
	return &CreateAdminUseCase{
		adminRepo: adminRepo,
	}
}

// Execute creates a new admin
func (uc *CreateAdminUseCase) Execute(ctx context.Context, req *CreateAdminRequest) (*CreateAdminResult, error) {
	// Check if admin already exists
	exists, err := uc.adminRepo.CheckExistsByWhatsAppNumber(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to check existing admin: %w", err))
	}

	if exists {
		return nil, domain.NewError(domain.ErrConflict, errors.New("whatsapp number already in use"))
	}

	// Create new admin
	newAdmin := entity.NewAdmin(req.WhatsAppNumber, "", req.FullName, req.Role)

	if err := uc.adminRepo.Create(ctx, newAdmin); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to create admin: %w", err))
	}

	return &CreateAdminResult{
		AdminID: newAdmin.AdminID.String(),
		Message: "Admin created successfully",
	}, nil
}

// UpdateAdminRequest represents update admin request
type UpdateAdminRequest struct {
	FullName string `json:"full_name" validate:"omitempty,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=super_admin,admin"`
	IsActive *bool  `json:"is_active" validate:"omitempty"`
}

// UpdateAdminResult represents update admin result
type UpdateAdminResult struct {
	AdminID string `json:"admin_id"`
	Message string `json:"message"`
}

// UpdateAdminUseCase handles updating admin details
type UpdateAdminUseCase struct {
	adminRepo ports.AdminRepository
}

// NewUpdateAdminUseCase creates a new update admin use case
func NewUpdateAdminUseCase(adminRepo ports.AdminRepository) *UpdateAdminUseCase {
	return &UpdateAdminUseCase{
		adminRepo: adminRepo,
	}
}

// Execute updates admin details
func (uc *UpdateAdminUseCase) Execute(ctx context.Context, adminID string, req *UpdateAdminRequest) (*UpdateAdminResult, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	// Update fields if provided
	if req.FullName != "" {
		admin.UpdateProfile(req.FullName)
	}

	if req.Role != "" {
		admin.Role = req.Role
	}

	if req.IsActive != nil {
		admin.IsActive = *req.IsActive
	}

	if err := uc.adminRepo.Update(ctx, admin); err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update admin: %w", err))
	}

	return &UpdateAdminResult{
		AdminID: admin.AdminID.String(),
		Message: "Admin updated successfully",
	}, nil
}

// DeleteAdminUseCase handles deleting an admin
type DeleteAdminUseCase struct {
	adminRepo ports.AdminRepository
}

// NewDeleteAdminUseCase creates a new delete admin use case
func NewDeleteAdminUseCase(adminRepo ports.AdminRepository) *DeleteAdminUseCase {
	return &DeleteAdminUseCase{
		adminRepo: adminRepo,
	}
}

// Execute deletes an admin
func (uc *DeleteAdminUseCase) Execute(ctx context.Context, adminID string) error {
	// Parse admin ID to verify it's valid
	_, err := uuid.Parse(adminID)
	if err != nil {
		return domain.NewError(domain.ErrBadRequest, errors.New("invalid admin ID"))
	}

	if err := uc.adminRepo.SoftDelete(ctx, adminID); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to delete admin: %w", err))
	}

	return nil
}
