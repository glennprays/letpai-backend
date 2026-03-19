package entity

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	UserID          uuid.UUID  `json:"user_id"`
	WhatsAppNumber  string     `json:"whatsapp_number"`
	PasswordHash    string     `json:"-"`
	FullName        string     `json:"full_name,omitempty"`
	AvatarURL       string     `json:"avatar_url,omitempty"`
	IsVerified      bool       `json:"is_verified"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

// NewUser creates a new user instance
func NewUser(whatsappNumber, passwordHash string) *User {
	now := time.Now()
	return &User{
		UserID:         uuid.New(),
		WhatsAppNumber: whatsappNumber,
		PasswordHash:   passwordHash,
		IsVerified:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// MarkVerified marks the user as verified
func (u *User) MarkVerified() {
	u.IsVerified = true
	u.UpdatedAt = time.Now()
}

// UpdateProfile updates user profile information
func (u *User) UpdateProfile(fullName, avatarURL string) {
	if fullName != "" {
		u.FullName = fullName
	}
	if avatarURL != "" {
		u.AvatarURL = avatarURL
	}
	u.UpdatedAt = time.Now()
}

// SoftDelete marks the user as deleted
func (u *User) SoftDelete() {
	now := time.Now()
	u.DeletedAt = &now
	u.UpdatedAt = now
}
