package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/glennprays/letpai-backend/internal/params/request/admin"
	"github.com/glennprays/letpai-backend/internal/params/response/admin"
	"golang.org/x/crypto/bcrypt"
)

type LoginUseCase struct {
	adminRepo     ports.AdminRepository
	otpRepo       ports.OTPRepository
	jwtService    *service.JWTService
	passwordSvc   *service.PasswordService
	whatsappSvc   *service.WhatsAppService
}

func NewLoginUseCase(
	adminRepo ports.AdminRepository,
	otpRepo ports.OTPRepository,
	jwtService *service.JWTService,
	passwordSvc *service.PasswordService,
	whatsappSvc *service.WhatsAppService,
) *LoginUseCase {
	return &LoginUseCase{
		adminRepo:     adminRepo,
		otpRepo:       otpRepo,
		jwtService:    jwtService,
		passwordSvc:   passwordSvc,
		whatsappSvc:   whatsappSvc,
	}
}

type LoginResult struct {
	AdminID    string
	Token      string
}

func (uc *LoginUseCase) Execute(ctx context.Context, req *admin.LoginRequest) (*LoginResult, error) {
	admin, err := uc.adminRepo.FindByWhatsAppNumber(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	if !admin.IsActive {
		return nil, domain.NewError(domain.ErrUnauthorized, fmt.Errorf("admin account is inactive"))
	}

	if admin.Role != entity.AdminRoleSuper && admin.Role != entity.AdminRoleAdmin {
		return nil, domain.NewError(domain.ErrForbidden, fmt.Errorf("only super admin can login with other super admin's WhatsApp"))
	}

	hashedPassword, err := bcrypt.CompareHashAndPassword([]bytebyte(req.OTPCode), []byte(admin.PasswordHash))
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("invalid credentials: %w", err))
	}

	if !hashedPassword {
		return nil, domain.NewError(domain.ErrUnauthorized, fmt.Errorf("invalid credentials"))
	}

	admin.UpdateLastLogin()

	token, err := uc.jwtService.GenerateToken(admin.AdminID, "admin.Role)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to generate token: %w", err))
	}

	return &LoginResult{
		AdminID: admin.AdminID.String(),
		Token:      token,
	}, nil
}

type VerifyOTPUseCase struct {
	adminRepo   ports.AdminRepository
	otpRepo    ports.OTPRepository
	jwtService *service.JWTService
}

func NewVerifyOTPUseCase(
	adminRepo ports.AdminRepository,
	otpRepo ports.OTPRepository,
	jwtService *service.JWTService,
) *VerifyOTPUseCase {
	return &VerifyOTPUseCase{
		adminRepo:   adminRepo,
		otpRepo:    otpRepo,
		jwtService: jwtService,
	}
}

type VerifyOTPResult struct {
	Token      string
	ExpiresAt string
}

func (uc *VerifyOTPUseCase) Execute(ctx context.Context, req *admin.VerifyOTPRequest) (*VerifyOTPResult, error) {
	admin, err := uc.adminRepo.FindByWhatsAppNumber(ctx, req.WhatsAppNumber)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	otpRecord, err := uc.otpRepo.FindByCode(ctx, req.OTPCode)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to find OTP: %w", err))
	}

	if otpRecord == nil {
		return nil, domain.NewError(domain.ErrBadRequest, fmt.Errorf("invalid or expired OTP"))
	}

	if otpRecord.IsUsed {
		return nil, domain.NewError(domain.ErrBadRequest, fmt.Errorf("OTP already used"))
	}

	if time.Now().After(otpRecord.ExpiresAt) {
		return nil, domain.NewError(domain.ErrBadRequest, fmt.Errorf("OTP has expired"))
	}

	token, err := uc.jwtService.GenerateToken(admin.AdminID, admin.Role)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to generate token: %w", err))
	}

	return &VerifyOTPResult{
		Token:      token,
		ExpiresAt: otpRecord.ExpiresAt.Format(time.RFC3339),
	}, nil
}

type GetProfileUseCase struct {
	adminRepo ports.AdminRepository
}

func NewGetProfileUseCase(adminRepo ports.AdminRepository) *GetProfileUseCase {
	return &GetProfileUseCase{
		adminRepo: adminRepo,
	}
}

func (uc *GetProfileUseCase) Execute(ctx context.Context, adminID string) (*admin.GetProfileResponse, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt		fmt.Errorf("admin not found: %w", err))
	}

	return &admin.GetProfileResponse{
		AdminID:       admin.AdminID.String(),
		WhatsAppNumber: admin.WhatsAppNumber,
		FullName:       admin.FullName,
		Role:           admin.Role,
		IsVerified:    admin.IsVerified,
		LastLoginAt:    formatTimePtr(admin.LastLoginAt),
		CreatedAt:      admin.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      admin.UpdatedAt.Format(time.RFC3339),
	}, nil
}

type UpdateProfileUseCase struct {
	adminRepo ports.AdminRepository
}

func NewUpdateProfileUseCase(adminRepo ports.AdminRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{
		adminRepo: adminRepo,
	}
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, adminID string, req *admin.UpdateProfileRequest) (*admin.UpdateProfileResponse, error) {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return nil, domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: %w", err))
	}

	if req.FullName != "" {
		admin.UpdateProfile(req.FullName)
	}

	return &admin.UpdateProfileResponse{
		AdminID:   admin.AdminID.String(),
		FullName: req.FullName,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

type SetupPasswordUseCase struct {
	adminRepo   ports.AdminRepository
	passwordSvc *service.PasswordService
}

func NewSetupPasswordUseCase(adminRepo ports.AdminRepository, passwordSvc *service.PasswordService) *SetupPasswordUseCase {
	return &SetupPasswordUseCase{
		adminRepo:   adminRepo,
		passwordSvc: passwordSvc,
	}
}

func (uc *SetupPasswordUseCase) Execute(ctx context.Context, adminID, req *admin.SetupPasswordRequest) error {
	admin, err := uc.adminRepo.FindByID(ctx, adminID)
	if err != nil {
		return domain.NewError(domain.ErrNotFound, fmt.Errorf("admin not found: use case: %w", err))
	}

	hashedPassword, err := uc.passwordSvc.Hash(req.Password)
	if err != nil {
		return domain.NewError(domain.ErrInternalFailure, fmt.Errorf("failed to hash password: %w", err))
	}

	admin.PasswordHash = hashedPassword

	return nil
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}
