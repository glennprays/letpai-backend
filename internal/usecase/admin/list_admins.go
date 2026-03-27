package admin

import (
	"context"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// ListAdminsResponse represents list of admins response
type ListAdminsResponse struct {
	Admins []AdminProfile `json:"admins"`
}

// AdminProfile represents admin profile data
type AdminProfile struct {
	AdminID        string `json:"admin_id"`
	WhatsAppNumber string `json:"whatsapp_number"`
	FullName       string `json:"full_name"`
	Role           string `json:"role"`
	IsActive       bool   `json:"is_active"`
	CreatedAt      string `json:"created_at"`
}

// ListAdminsUseCase handles listing all admins
type ListAdminsUseCase struct {
	adminRepo ports.AdminRepository
}

// NewListAdminsUseCase creates a new list admins use case
func NewListAdminsUseCase(adminRepo ports.AdminRepository) *ListAdminsUseCase {
	return &ListAdminsUseCase{
		adminRepo: adminRepo,
	}
}

// Execute retrieves all admins
func (uc *ListAdminsUseCase) Execute(ctx context.Context) (*ListAdminsResponse, error) {
	admins, err := uc.adminRepo.List(ctx)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to list admins: %w", err))
	}

	adminProfiles := make([]AdminProfile, 0, len(admins))
	for _, admin := range admins {
		adminProfiles = append(adminProfiles, AdminProfile{
			AdminID:        admin.AdminID.String(),
			WhatsAppNumber: admin.WhatsAppNumber,
			FullName:       admin.FullName,
			Role:           admin.Role,
			IsActive:       admin.IsActive,
			CreatedAt:      admin.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	return &ListAdminsResponse{
		Admins: adminProfiles,
	}, nil
}
