package entity

import (
	"time"

	"github.com/google/uuid"
)

// BillItem represents a bill item in a session
type BillItem struct {
	BillItemID   uuid.UUID  `json:"bill_item_id" db:"bill_item_id"`
	SessionID    uuid.UUID  `json:"session_id" db:"session_id"`
	Description  string     `json:"description" db:"description"`
	Amount       float64    `json:"amount" db:"amount"`
	Category     *string    `json:"category,omitempty" db:"category"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// NewBillItem creates a new bill item
func NewBillItem(sessionID uuid.UUID, description string, amount float64, category *string) *BillItem {
	now := time.Now()
	return &BillItem{
		BillItemID:  uuid.New(),
		SessionID:   sessionID,
		Description: description,
		Amount:      amount,
		Category:    category,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Update updates the bill item information
func (b *BillItem) Update(description string, amount float64, category *string) {
	if description != "" {
		b.Description = description
	}
	b.Amount = amount
	if category != nil {
		b.Category = category
	}
	b.UpdatedAt = time.Now()
}

// SetAmount sets the amount
func (b *BillItem) SetAmount(amount float64) {
	b.Amount = amount
	b.UpdatedAt = time.Now()
}

// SetDescription sets the description
func (b *BillItem) SetDescription(description string) {
	b.Description = description
	b.UpdatedAt = time.Now()
}

// SetCategory sets the category
func (b *BillItem) SetCategory(category *string) {
	b.Category = category
	b.UpdatedAt = time.Now()
}
