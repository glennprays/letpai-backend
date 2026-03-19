package entity

import (
	"time"

	"github.com/google/uuid"
)

// ContactGroup represents a contact group for organizing contacts
type ContactGroup struct {
	GroupID      uuid.UUID  `json:"group_id" db:"group_id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	Name         string     `json:"name" db:"name"`
	Color        string     `json:"color" db:"color"`
	SortOrder    int        `json:"sort_order" db:"sort_order"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewContactGroup creates a new contact group
func NewContactGroup(userID uuid.UUID, name, color string, sortOrder int) *ContactGroup {
	now := time.Now()
	return &ContactGroup{
		GroupID:   uuid.New(),
		UserID:    userID,
		Name:      name,
		Color:     color,
		SortOrder: sortOrder,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Update updates the contact group information
func (g *ContactGroup) Update(name, color string) {
	if name != "" {
		g.Name = name
	}
	if color != "" {
		g.Color = color
	}
	g.UpdatedAt = time.Now()
}

// UpdateSortOrder updates the sort order
func (g *ContactGroup) UpdateSortOrder(sortOrder int) {
	g.SortOrder = sortOrder
	g.UpdatedAt = time.Now()
}

// SoftDelete marks the contact group as deleted
func (g *ContactGroup) SoftDelete() {
	now := time.Now()
	g.DeletedAt = &now
	g.UpdatedAt = now
}

// IsDeleted checks if the contact group is deleted
func (g *ContactGroup) IsDeleted() bool {
	return g.DeletedAt != nil
}
