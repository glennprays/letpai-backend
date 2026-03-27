package admin

import (
	"context"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// ListAdminsUseCase lists all admins (super_admin only)
type ListAdminsUseCase struct {
	adminRepo ports.AdminRepository
}

// NewListAdminsUseCase creates a new list admins use case instance
func NewListAdminsUseCase(adminRepo ports.AdminRepository) *ListAdminsUseCase {
	return &ListAdminsUseCase{
		adminRepo: adminRepo,
	}
}

// Execute retrieves all admins
func (uc *ListAdminsUseCase) Execute(ctx context.Context) ([]*entity.Admin, error) {
	admins, err := uc.adminRepo.List(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to list admins: %w", err))
	}
	return admins, nil
}

type CreateAdminUseCase struct {
	adminRepo ports.AdminRepository
}

func NewCreateAdminUseCase(adminRepo ports.AdminRepository) *CreateAdminUseCase {
	return &CreateAdminUseCase{
		adminRepo: adminRepo,
	}
}

type CreateAdminRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	FullName       string `json:"full_name" validate:"required,max=100"`
	Role           string `json:"role" validate:"required,oneof=super_admin,admin"`
}

type CreateAdminResponse struct {
	AdminID string `json:"admin_id"`
	Message string `json:"message"`
}

func (uc *CreateAdminUseCase) Execute(ctx context.Context, req *CreateAdminRequest) (*CreateAdminResponse, error) {
	existingAdmin, err := uc.adminRepo.FindByWhatsAppNumber(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to check existing admin: %w", err))
	}

	if existingAdmin != nil {
		return nil, domain.NewError(domain.ErrConflict, fmt.Errorf("whatsapp number already in use: %s", req.WhatsAppNumber))
	}

	admin := &entity.Admin{
		AdminID:        entity.AdminRoleSuper,
		WhatsAppNumber: req.WhatsAppNumber,
		FullName:       req.FullName,
		Role:           req.Role,
		IsActive:       true,
	}

	err = uc.adminRepo.Create(ctx, admin)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to create admin: %w", err))
	}

	return &CreateAdminResponse{
		AdminID: admin.AdminID.String(),
		Message: "Admin created successfully",
	}, nil
}

type UpdateAdminUseCase struct {
	adminRepo ports.AdminRepository
}

func NewUpdateAdminUseCase(adminRepo ports.AdminRepository) *UpdateAdminUseCase {
	return &UpdateAdminUseCase{
		adminRepo: adminRepo,
	}
}

type UpdateAdminRequest struct {
	FullName string `json:"full_name" validate:"omitempty,max=100"`
	Role     string `json:"role" validate:"omitempty,oneof=super_admin,admin"`
	IsActive bool   `json:"is_active" validate:"omitempty"`
}

type UpdateAdminResponse struct {
	AdminID string `json:"admin_id"`
	Message string `json:"message"`
}

func (uc *UpdateAdminUseCase) Execute(ctx context.Context, adminID string, req *UpdateAdminRequest) (*UpdateAdminResponse, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin: %w", err))
	}

	if admin == nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found"))
	}

	if admin.Role == entity.AdminRoleSuper && req.Role == entity.AdminRoleAdmin {
		return nil, domain.NewError(domain.ErrForbidden, fmt.Errorf("cannot change super admin role"))
	}

	admin.FullName = req.FullName
	admin.IsActive = req.IsActive
	admin.UpdatedAt = time.Now()

	err = uc.adminRepo.Update(ctx, admin)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to update admin: %w", err))
	}

	return &UpdateAdminResponse{
		AdminID: adminID.String(),
		Message: "Admin updated successfully",
	}, nil
}

type DeleteAdminUseCase struct {
	adminRepo ports.AdminRepository
}

func NewDeleteAdminUseCase(adminRepo ports.AdminRepository) *DeleteAdminUseCase {
	return &DeleteAdminUseCase{
		adminRepo: adminRepo,
	}
}

type DeleteAdminResponse struct {
	Message string `json:"message"`
}

func (uc *DeleteAdminUseCase) Execute(ctx context.Context, adminID string) (*DeleteAdminResponse, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find admin: %w", err))
	}

	if admin == nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found"))
	}

	if admin.AdminID == entity.AdminRoleSuper {
		return nil, domain.NewError(domain.ErrForbidden, fmt.Errorf("cannot delete super admin"))
	}

	err = uc.adminRepo.SoftDelete(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to delete admin: %w", err))
	}

	return &DeleteAdminResponse{
		Message: "Admin deleted successfully",
	}, nil
}
