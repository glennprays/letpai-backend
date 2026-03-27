package entity

import (
	"time"

	"github.com/google/uuid"
)

// Admin represents an admin user in the system
type Admin struct {
	AdminID        uuid.UUID  `json:"admin_id"`
	WhatsAppNumber string     `json:"whatsapp_number"`
	PasswordHash   string     `json:"-"`
	FullName       string     `json:"full_name"`
	Role           string     `json:"role"`
	IsActive       bool       `json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

const (
	AdminRoleSuper = "super_admin"
	AdminRoleAdmin = "admin"
)

// NewAdmin creates a new admin instance
func NewAdmin(whatsappNumber, passwordHash, fullName, role string) *Admin {
	now := time.Now()
	return &Admin{
		AdminID:        uuid.New(),
		WhatsAppNumber: whatsappNumber,
		PasswordHash:   passwordHash,
		FullName:       fullName,
		Role:           role,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// UpdateLastLogin updates the last login timestamp
func (a *Admin) UpdateLastLogin() {
	now := time.Now()
	a.LastLoginAt = &now
	a.UpdatedAt = now
}

// UpdateProfile updates admin profile information
func (a *Admin) UpdateProfile(fullName string) {
	if fullName != "" {
		a.FullName = fullName
	}
	a.UpdatedAt = time.Now()
}

// SoftDelete marks the admin as deleted
func (a *Admin) SoftDelete() {
	now := time.Now()
	a.DeletedAt = &now
	a.IsActive = false
	a.UpdatedAt = now
}
