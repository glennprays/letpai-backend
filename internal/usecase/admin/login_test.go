package admin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/internal/service"
	adminuc "github.com/glennprays/letpai-backend/internal/usecase/admin"
)

type mockLoginAdminRepository struct {
	admin   *entity.Admin
	findErr error
	updated bool
}

func (m *mockLoginAdminRepository) Create(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (m *mockLoginAdminRepository) FindByWhatsAppNumber(ctx context.Context, whatsappNumber string) (*entity.Admin, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.admin, nil
}

func (m *mockLoginAdminRepository) FindByID(ctx context.Context, adminID string) (*entity.Admin, error) {
	return m.admin, nil
}

func (m *mockLoginAdminRepository) Update(ctx context.Context, admin *entity.Admin) error {
	m.updated = true
	return nil
}

func (m *mockLoginAdminRepository) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	return 1, nil
}

func (m *mockLoginAdminRepository) HasUsableSuperAdmin(ctx context.Context) (bool, error) {
	return true, nil
}

func (m *mockLoginAdminRepository) UpsertBootstrapSuperAdmin(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (m *mockLoginAdminRepository) SoftDelete(ctx context.Context, adminID string) error {
	return nil
}

func (m *mockLoginAdminRepository) List(ctx context.Context) ([]*entity.Admin, error) {
	return []*entity.Admin{}, nil
}

func (m *mockLoginAdminRepository) UpdateLastLogin(ctx context.Context, adminID string) error {
	return nil
}

func (m *mockLoginAdminRepository) CheckExistsByWhatsAppNumber(ctx context.Context, whatsappNumber string) (bool, error) {
	return m.admin != nil, nil
}

type mockLoginOTPRepository struct {
	otp      *entity.OTPVerification
	otpErr   error
	markUsed bool
}

func (m *mockLoginOTPRepository) Create(ctx context.Context, otp *entity.OTPVerification) error {
	return nil
}

func (m *mockLoginOTPRepository) FindValidByPhone(ctx context.Context, whatsappNumber string) (*entity.OTPVerification, error) {
	if m.otpErr != nil {
		return nil, m.otpErr
	}
	return m.otp, nil
}

func (m *mockLoginOTPRepository) FindByID(ctx context.Context, otpID string) (*entity.OTPVerification, error) {
	return m.otp, nil
}

func (m *mockLoginOTPRepository) MarkAsUsed(ctx context.Context, otpID string) error {
	m.markUsed = true
	return nil
}

func (m *mockLoginOTPRepository) InvalidatePreviousOTPs(ctx context.Context, whatsappNumber string) error {
	return nil
}

func (m *mockLoginOTPRepository) CleanupExpired(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}

func newLoginTestUseCase(otpRepo *mockLoginOTPRepository, adminRepo *mockLoginAdminRepository) *adminuc.LoginUseCase {
	jwtSvc := service.NewJWTService("test-secret", 1)
	passwordSvc := service.NewPasswordService(4)
	return adminuc.NewLoginUseCase(adminRepo, otpRepo, passwordSvc, jwtSvc)
}

func TestLoginUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	admin := &entity.Admin{
		AdminID:        uuid.New(),
		WhatsAppNumber: "+6281234567890",
		FullName:       "Super Admin",
		Role:           entity.AdminRoleSuper,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	otpRepo := &mockLoginOTPRepository{
		otp: entity.NewOTPVerification("+6281234567890", "123456", 5*time.Minute),
	}
	adminRepo := &mockLoginAdminRepository{admin: admin}

	uc := newLoginTestUseCase(otpRepo, adminRepo)
	result, err := uc.Execute(ctx, &adminuc.LoginRequest{
		WhatsAppNumber: "+6281234567890",
		OTPCode:        "123456",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.Token == "" {
		t.Error("expected non-empty token")
	}
	if result.ExpiresAt == "" {
		t.Error("expected non-empty expires_at")
	}
	if !otpRepo.markUsed {
		t.Error("expected OTP to be marked as used")
	}
	if !adminRepo.updated {
		t.Error("expected last login to be persisted")
	}
}

func TestLoginUseCase_Execute_WrongOTP(t *testing.T) {
	ctx := context.Background()
	otpRepo := &mockLoginOTPRepository{
		otp: entity.NewOTPVerification("+6281234567890", "123456", 5*time.Minute),
	}
	adminRepo := &mockLoginAdminRepository{admin: &entity.Admin{IsActive: true}}

	uc := newLoginTestUseCase(otpRepo, adminRepo)
	_, err := uc.Execute(ctx, &adminuc.LoginRequest{
		WhatsAppNumber: "+6281234567890",
		OTPCode:        "654321",
	})

	if err == nil {
		t.Fatal("expected error for wrong OTP")
	}
	var domainErr domain.Error
	if !errors.As(err, &domainErr) || domainErr.ServiceError() != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLoginUseCase_Execute_NoValidOTP(t *testing.T) {
	ctx := context.Background()
	otpRepo := &mockLoginOTPRepository{
		otpErr: domain.NewError(domain.ErrNotFound, nil),
	}
	adminRepo := &mockLoginAdminRepository{admin: &entity.Admin{IsActive: true}}

	uc := newLoginTestUseCase(otpRepo, adminRepo)
	_, err := uc.Execute(ctx, &adminuc.LoginRequest{
		WhatsAppNumber: "+6281234567890",
		OTPCode:        "123456",
	})

	if err == nil {
		t.Fatal("expected error when no valid OTP exists")
	}
	var domainErr domain.Error
	if !errors.As(err, &domainErr) || domainErr.ServiceError() != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}
