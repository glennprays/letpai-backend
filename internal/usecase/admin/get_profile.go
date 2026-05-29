package admin

import (
	"context"
	"time"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// GetProfileResponse represents admin profile data
//
// PasswordSetupRequired is true when the admin has never set a
// password (only OTP login is available to them). Drives the
// dashboard banner that nudges the host to add a backup credential
// in case the WhatsApp gateway is unreachable.
type GetProfileResponse struct {
	AdminID               string     `json:"admin_id"`
	WhatsAppNumber        string     `json:"whatsapp_number"`
	FullName              string     `json:"full_name"`
	Role                  string     `json:"role"`
	IsVerified            bool       `json:"is_verified"`
	IsActive              bool       `json:"is_active"`
	PasswordSetupRequired bool       `json:"password_setup_required"`
	LastLoginAt           *time.Time `json:"last_login_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// GetProfileUseCase handles getting admin profile
type GetProfileUseCase struct {
	adminRepo ports.AdminRepository
}

// NewGetProfileUseCase creates a new get profile use case
func NewGetProfileUseCase(adminRepo ports.AdminRepository) *GetProfileUseCase {
	return &GetProfileUseCase{
		adminRepo: adminRepo,
	}
}

// Execute retrieves admin profile by ID
func (uc *GetProfileUseCase) Execute(ctx context.Context, adminID string) (*GetProfileResponse, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, err
	}

	return &GetProfileResponse{
		AdminID:               admin.AdminID.String(),
		WhatsAppNumber:        admin.WhatsAppNumber,
		FullName:              admin.FullName,
		Role:                  admin.Role,
		IsVerified:            true, // Admins are verified by default
		IsActive:              admin.IsActive,
		PasswordSetupRequired: admin.PasswordHash == "",
		LastLoginAt:           admin.LastLoginAt,
		CreatedAt:             admin.CreatedAt,
		UpdatedAt:             admin.UpdatedAt,
	}, nil
}
