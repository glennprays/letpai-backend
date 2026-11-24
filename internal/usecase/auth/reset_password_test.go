package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/internal/service"
)

type resetPasswordUserRepo struct {
	user    *entity.User
	updated bool
}

func (r *resetPasswordUserRepo) Create(context.Context, *entity.User) error { return nil }
func (r *resetPasswordUserRepo) FindByID(_ context.Context, userID string) (*entity.User, error) {
	if r.user == nil || r.user.UserID.String() != userID {
		return nil, errors.New("user not found")
	}
	return r.user, nil
}
func (r *resetPasswordUserRepo) FindByWhatsApp(_ context.Context, phone string) (*entity.User, error) {
	if r.user == nil || r.user.WhatsAppNumber != phone {
		return nil, errors.New("user not found")
	}
	return r.user, nil
}
func (r *resetPasswordUserRepo) Update(_ context.Context, user *entity.User) error {
	r.user = user
	r.updated = true
	return nil
}
func (r *resetPasswordUserRepo) Delete(context.Context, string) error { return nil }
func (r *resetPasswordUserRepo) IsExists(_ context.Context, phone string) (bool, error) {
	return r.user != nil && r.user.WhatsAppNumber == phone, nil
}

type resetPasswordOTPRepo struct {
	otp         *entity.OTPVerification
	marked      bool
	invalidated bool
}

func (r *resetPasswordOTPRepo) Create(context.Context, *entity.OTPVerification) error { return nil }
func (r *resetPasswordOTPRepo) FindValidByPhone(_ context.Context, phone string) (*entity.OTPVerification, error) {
	if r.otp == nil || r.otp.WhatsAppNumber != phone {
		return nil, errors.New("OTP not found")
	}
	return r.otp, nil
}
func (r *resetPasswordOTPRepo) FindByID(context.Context, string) (*entity.OTPVerification, error) {
	return r.otp, nil
}
func (r *resetPasswordOTPRepo) MarkAsUsed(_ context.Context, _ string) error {
	r.marked = true
	r.otp.MarkUsed()
	return nil
}
func (r *resetPasswordOTPRepo) InvalidatePreviousOTPs(context.Context, string) error {
	r.invalidated = true
	return nil
}
func (r *resetPasswordOTPRepo) CleanupExpired(context.Context, time.Duration) (int64, error) {
	return 0, nil
}

func newResetPasswordTestUseCase(t *testing.T, code string, expiresAt time.Time) (*ResetPasswordUseCase, *resetPasswordUserRepo, *resetPasswordOTPRepo, *service.PasswordService) {
	t.Helper()
	passwordSvc := service.NewPasswordService(4)
	oldHash, err := passwordSvc.Hash("old-password")
	if err != nil {
		t.Fatalf("hash old password: %v", err)
	}
	user := entity.NewUser("6281234567890", oldHash)
	userRepo := &resetPasswordUserRepo{user: user}
	otp := entity.NewOTPVerificationWithUser(user.UserID, user.WhatsAppNumber, code, time.Minute)
	otp.ExpiresAt = expiresAt
	otpRepo := &resetPasswordOTPRepo{otp: otp}
	return NewResetPasswordUseCase(userRepo, otpRepo, service.NewOTPService(5), passwordSvc), userRepo, otpRepo, passwordSvc
}

func TestResetPassword_UpdatesPasswordAndConsumesOTP(t *testing.T) {
	uc, userRepo, otpRepo, passwordSvc := newResetPasswordTestUseCase(t, "123456", time.Now().Add(time.Minute))

	result, err := uc.Execute(context.Background(), &ResetPasswordRequest{
		WhatsAppNumber: "6281234567890",
		OTPCode:        "123456",
		NewPassword:    "new-password",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Message == "" {
		t.Fatal("expected success message")
	}
	if !userRepo.updated || !passwordSvc.Verify("new-password", userRepo.user.PasswordHash) {
		t.Fatal("expected the stored password hash to match the new password")
	}
	if !otpRepo.marked || !otpRepo.otp.IsUsed || !otpRepo.invalidated {
		t.Fatal("expected reset OTP to be consumed and stale OTPs invalidated")
	}
}

func TestResetPassword_RejectsInvalidCodeWithoutChangingPassword(t *testing.T) {
	uc, userRepo, otpRepo, passwordSvc := newResetPasswordTestUseCase(t, "123456", time.Now().Add(time.Minute))
	oldHash := userRepo.user.PasswordHash

	_, err := uc.Execute(context.Background(), &ResetPasswordRequest{
		WhatsAppNumber: "6281234567890",
		OTPCode:        "654321",
		NewPassword:    "new-password",
	})
	if err == nil {
		t.Fatal("expected invalid code error")
	}
	if userRepo.updated || otpRepo.marked || userRepo.user.PasswordHash != oldHash || !passwordSvc.Verify("old-password", userRepo.user.PasswordHash) {
		t.Fatal("invalid code must not consume the OTP or change the password")
	}
}

func TestResetPassword_WeakPasswordDoesNotConsumeOTP(t *testing.T) {
	uc, userRepo, otpRepo, _ := newResetPasswordTestUseCase(t, "123456", time.Now().Add(time.Minute))

	_, err := uc.Execute(context.Background(), &ResetPasswordRequest{
		WhatsAppNumber: "6281234567890",
		OTPCode:        "123456",
		NewPassword:    "short",
	})
	if err == nil {
		t.Fatal("expected password validation error")
	}
	if userRepo.updated || otpRepo.marked || otpRepo.otp.IsUsed {
		t.Fatal("invalid password must leave the code available for retry")
	}
}
