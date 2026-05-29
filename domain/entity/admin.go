package entity

import (
	"time"

	"github.com/google/uuid"
)

// Admin represents an admin user in the system.
//
// db tags are explicit because sqlx's default mapper is
// strings.ToLower, which would turn PasswordHash into passwordhash
// (no underscore) and silently fail to populate the field from a
// password_hash column. That blew up the password login flow before
// these tags were added: bootstrap wrote a real bcrypt hash to
// password_hash, but FindByWhatsAppNumber's GetContext read it back
// as an empty string, so LoginUseCase always returned 401.
type Admin struct {
	AdminID        uuid.UUID  `json:"admin_id"            db:"admin_id"`
	WhatsAppNumber string     `json:"whatsapp_number"     db:"whatsapp_number"`
	PasswordHash   string     `json:"-"                   db:"password_hash"`
	FullName       string     `json:"full_name"           db:"full_name"`
	Role           string     `json:"role"                db:"role"`
	IsActive       bool       `json:"is_active"           db:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at"          db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"          db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
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
