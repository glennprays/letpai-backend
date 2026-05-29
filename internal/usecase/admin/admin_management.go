package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// CreateAdminRequest represents create admin request.
//
// Password is optional. When provided, the invited admin can sign in
// via the password flow without needing the WhatsApp gateway to be
// online — useful when an inviter wants to onboard someone before
// pairing WAGA. When empty the admin is created without a hash and
// must sign in via OTP for their first session.
type CreateAdminRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	FullName       string `json:"full_name" validate:"required,max=100"`
	Role           string `json:"role" validate:"required,oneof=super_admin,admin"`
	Password       string `json:"password,omitempty" validate:"omitempty,min=8,max=100"`
}

// CreateAdminResult represents create admin result
type CreateAdminResult struct {
	AdminID string `json:"admin_id"`
	Message string `json:"message"`
}

// CreateAdminUseCase handles creating a new admin
type CreateAdminUseCase struct {
	adminRepo   ports.AdminRepository
	passwordSvc *service.PasswordService
}

// NewCreateAdminUseCase creates a new create admin use case
func NewCreateAdminUseCase(
	adminRepo ports.AdminRepository,
	passwordSvc *service.PasswordService,
) *CreateAdminUseCase {
	return &CreateAdminUseCase{
		adminRepo:   adminRepo,
		passwordSvc: passwordSvc,
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

	// Optional temporary password: when the inviter supplies one we
	// bcrypt it on the way in so the invited admin can sign in via
	// the password flow before WAGA is paired.
	passwordHash := ""
	if req.Password != "" {
		hashed, err := uc.passwordSvc.Hash(req.Password)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to hash password: %w", err))
		}
		passwordHash = hashed
	}

	newAdmin := entity.NewAdmin(req.WhatsAppNumber, passwordHash, req.FullName, req.Role)

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

// Execute updates admin details. Enforces two safety rails to prevent
// accidental loss of administrative access:
//   - callers cannot demote themselves (would leave them without rights mid-flight)
//   - the last remaining super_admin cannot be demoted or deactivated
//     (would lock the system; recovery requires direct DB access)
func (uc *UpdateAdminUseCase) Execute(ctx context.Context, callerID, adminID string, req *UpdateAdminRequest) (*UpdateAdminResult, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	demoting := req.Role != "" && req.Role != "super_admin" && admin.Role == "super_admin"
	deactivating := req.IsActive != nil && !*req.IsActive && admin.IsActive

	if admin.AdminID.String() == callerID && (demoting || deactivating) {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("admins cannot demote or deactivate themselves"))
	}

	if (demoting || deactivating) && admin.Role == "super_admin" {
		count, err := uc.adminRepo.CountActiveSuperAdmins(ctx)
		if err != nil {
			return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to count super admins: %w", err))
		}
		if count <= 1 {
			return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot demote or deactivate the last active super_admin"))
		}
	}

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

// Execute deletes an admin. Forbids deleting yourself and forbids deleting
// the last remaining super_admin — recovery from either would require direct
// database access.
func (uc *DeleteAdminUseCase) Execute(ctx context.Context, callerID, adminID string) error {
	if _, err := uuid.Parse(adminID); err != nil {
		return domain.NewError(domain.ErrBadRequest, errors.New("invalid admin ID"))
	}

	if adminID == callerID {
		return domain.NewError(domain.ErrBadRequest, errors.New("admins cannot delete themselves"))
	}

	target, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}
	if target.Role == "super_admin" {
		count, err := uc.adminRepo.CountActiveSuperAdmins(ctx)
		if err != nil {
			return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to count super admins: %w", err))
		}
		if count <= 1 {
			return domain.NewError(domain.ErrBadRequest, errors.New("cannot delete the last active super_admin"))
		}
	}

	if err := uc.adminRepo.SoftDelete(ctx, adminID); err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to delete admin: %w", err))
	}

	return nil
}
