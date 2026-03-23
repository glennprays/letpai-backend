package entity

import (
	"time"

	"github.com/google/uuid"
)

// Contact represents a contact in the system
type Contact struct {
	ContactID      uuid.UUID  `json:"contact_id" db:"contact_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	Name           string     `json:"name" db:"name"`
	WhatsAppNumber string     `json:"whatsapp_number" db:"whatsapp_number"`
	GroupID        *uuid.UUID `json:"group_id,omitempty" db:"group_id"`
	IsFavorite     bool       `json:"is_favorite" db:"is_favorite"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	// Joined fields (not in database)
	GroupName  *string `json:"group_name,omitempty" db:"group_name"`
	GroupColor *string `json:"group_color,omitempty" db:"group_color"`
}

// NewContact creates a new contact
func NewContact(userID uuid.UUID, name, whatsappNumber string) *Contact {
	now := time.Now()
	return &Contact{
		ContactID:      uuid.New(),
		UserID:         userID,
		Name:           name,
		WhatsAppNumber: whatsappNumber,
		IsFavorite:     false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Update updates the contact information
func (c *Contact) Update(name, whatsappNumber string) {
	if name != "" {
		c.Name = name
	}
	if whatsappNumber != "" {
		c.WhatsAppNumber = whatsappNumber
	}
	c.UpdatedAt = time.Now()
}

// AssignToGroup assigns the contact to a group
func (c *Contact) AssignToGroup(groupID *uuid.UUID) {
	c.GroupID = groupID
	c.UpdatedAt = time.Now()
}

// ToggleFavorite toggles the favorite status
func (c *Contact) ToggleFavorite() {
	c.IsFavorite = !c.IsFavorite
	c.UpdatedAt = time.Now()
}

// SetFavorite sets the favorite status
func (c *Contact) SetFavorite(isFavorite bool) {
	c.IsFavorite = isFavorite
	c.UpdatedAt = time.Now()
}

// SoftDelete marks the contact as deleted
func (c *Contact) SoftDelete() {
	now := time.Now()
	c.DeletedAt = &now
	c.UpdatedAt = now
}

// IsDeleted checks if the contact is deleted
func (c *Contact) IsDeleted() bool {
	return c.DeletedAt != nil
}
