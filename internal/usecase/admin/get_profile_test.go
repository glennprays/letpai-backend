package admin_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	adminuc "github.com/glennprays/letpai-backend/internal/usecase/admin"
)

func TestGetProfileUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	adminID := uuid.New().String()

	mockRepo := &mockAdminRepository{
		fullName: "Test Admin",
		role:     "admin",
		isActive: true,
	}

	uc := adminuc.NewGetProfileUseCase(mockRepo)
	result, err := uc.Execute(ctx, adminID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.AdminID != adminID {
		t.Errorf("admin ID mismatch: got %s, want %s", result.AdminID, adminID)
	}
	if result.FullName != "Test Admin" {
		t.Errorf("full name mismatch: got %s, want Test Admin", result.FullName)
	}
	if result.Role != "admin" {
		t.Errorf("role mismatch: got %s, want admin", result.Role)
	}
	if !result.IsActive {
		t.Error("expected active to be true")
	}
}

func TestGetProfileUseCase_Execute_NotFound(t *testing.T) {
	ctx := context.Background()
	adminID := uuid.New().String()

	mockRepo := &mockAdminRepository{shouldReturnError: true}

	uc := adminuc.NewGetProfileUseCase(mockRepo)
	result, err := uc.Execute(ctx, adminID)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("expected no result")
	}
}

type mockAdminRepository struct {
	adminID           *string
	fullName          string
	role              string
	isActive          bool
	shouldReturnError bool
}

func (m *mockAdminRepository) FindByWhatsAppNumber(ctx context.Context, whatsappNumber string) (*entity.Admin, error) {
	return &entity.Admin{
		AdminID:        uuid.New(),
		WhatsAppNumber: whatsappNumber,
		FullName:       m.fullName,
		Role:           m.role,
		IsActive:       m.isActive,
		LastLoginAt:    nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (m *mockAdminRepository) FindByID(ctx context.Context, adminID string) (*entity.Admin, error) {
	if m.shouldReturnError {
		return nil, domain.ErrNotFound
	}
	return &entity.Admin{
		AdminID:        uuid.MustParse(adminID),
		WhatsAppNumber: "+1234567890123",
		FullName:       m.fullName,
		Role:           m.role,
		IsActive:       m.isActive,
		LastLoginAt:    nil,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (m *mockAdminRepository) Create(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (m *mockAdminRepository) Update(ctx context.Context, admin *entity.Admin) error {
	return nil
}

func (m *mockAdminRepository) SoftDelete(ctx context.Context, adminID string) error {
	return nil
}

func (m *mockAdminRepository) List(ctx context.Context) ([]*entity.Admin, error) {
	return []*entity.Admin{}, nil
}

func (m *mockAdminRepository) UpdateLastLogin(ctx context.Context, adminID string) error {
	return nil
}

func (m *mockAdminRepository) CheckExistsByWhatsAppNumber(ctx context.Context, whatsappNumber string) (bool, error) {
	return false, nil
}
