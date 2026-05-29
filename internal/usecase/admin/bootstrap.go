package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// NeedsSetupResult is the public response for GET /admin/auth/needs-setup.
//
// Returned by an unauthenticated endpoint, so it deliberately carries no
// admin identifying information — only the single boolean the FE needs
// to route to /admin/setup or /admin/login.
type NeedsSetupResult struct {
	NeedsSetup bool `json:"needs_setup"`
}

// NeedsSetupUseCase is a thin wrapper around the repository's gate
// check; it exists so the handler depends on a use case (consistent
// with every other admin endpoint) rather than directly on the repo.
type NeedsSetupUseCase struct {
	adminRepo ports.AdminRepository
}

// NewNeedsSetupUseCase wires the use case.
func NewNeedsSetupUseCase(adminRepo ports.AdminRepository) *NeedsSetupUseCase {
	return &NeedsSetupUseCase{adminRepo: adminRepo}
}

// Execute returns whether the first-boot wizard is allowed.
func (uc *NeedsSetupUseCase) Execute(ctx context.Context) (*NeedsSetupResult, error) {
	hasUsable, err := uc.adminRepo.HasUsableSuperAdmin(ctx)
	if err != nil {
		return nil, err
	}
	return &NeedsSetupResult{NeedsSetup: !hasUsable}, nil
}

// BootstrapRequest is the body for POST /admin/auth/bootstrap. Same
// shape as the OTP/password login responses on the FE side so the
// wizard can drop the caller on /admin without a second round trip.
type BootstrapRequest struct {
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,len=13,max=20"`
	FullName       string `json:"full_name"       validate:"required,min=2,max=100"`
	Password       string `json:"password"        validate:"required,min=8,max=100"`
}

// BootstrapResult mirrors VerifyOTPResult and LoginResult so the FE can
// treat all three sign-in success shapes interchangeably.
type BootstrapResult struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	AdminID   string `json:"admin_id"`
}

// BootstrapUseCase claims the first-super-admin slot.
//
// The repository's UpsertBootstrapSuperAdmin re-checks the gate inside
// a SERIALIZABLE transaction so two concurrent requests don't both
// succeed. The handler returns 409 when the gate is closed.
type BootstrapUseCase struct {
	adminRepo   ports.AdminRepository
	passwordSvc *service.PasswordService
	jwtSvc      *service.JWTService
}

// NewBootstrapUseCase wires the use case.
func NewBootstrapUseCase(
	adminRepo ports.AdminRepository,
	passwordSvc *service.PasswordService,
	jwtSvc *service.JWTService,
) *BootstrapUseCase {
	return &BootstrapUseCase{
		adminRepo:   adminRepo,
		passwordSvc: passwordSvc,
		jwtSvc:      jwtSvc,
	}
}

// Execute creates or repairs the seed super-admin row, hashes the
// password, and (when issueToken=true) returns a JWT for the FE wizard
// to drop the caller on /admin.
//
// The CLI passes issueToken=false because it has no use for the JWT
// and can avoid loading the JWT service from config.
func (uc *BootstrapUseCase) Execute(ctx context.Context, req *BootstrapRequest, issueToken bool) (*BootstrapResult, error) {
	// Cheap pre-check: refuses with the friendly error before doing
	// the bcrypt work. The real source of truth is the repo tx.
	if uc.adminRepo != nil {
		hasUsable, err := uc.adminRepo.HasUsableSuperAdmin(ctx)
		if err != nil {
			return nil, err
		}
		if hasUsable {
			return nil, domain.NewError(domain.ErrConflict, errors.New("bootstrap not allowed: a super admin already exists"))
		}
	}

	hash, err := uc.passwordSvc.Hash(req.Password)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to hash password: %w", err))
	}

	admin := entity.NewAdmin(req.WhatsAppNumber, hash, req.FullName, entity.AdminRoleSuper)
	if err := uc.adminRepo.UpsertBootstrapSuperAdmin(ctx, admin); err != nil {
		return nil, err
	}

	if !issueToken {
		return &BootstrapResult{
			AdminID: admin.AdminID.String(),
		}, nil
	}

	token, err := uc.jwtSvc.GenerateTokenWithRole(admin.AdminID.String(), admin.WhatsAppNumber, admin.Role)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to generate token: %w", err))
	}

	return &BootstrapResult{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		AdminID:   admin.AdminID.String(),
	}, nil
}
