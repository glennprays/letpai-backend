package entity

import (
	"time"

	"github.com/google/uuid"
)

// BillItem represents a bill item in a session.
//
// ParticipantIDs is populated by the repository via the
// bill_item_participants join table. An empty slice means "applies to
// everyone in the session" (legacy behaviour); a non-empty slice means
// the bill is shared only among the listed participants.
type BillItem struct {
	BillItemID               uuid.UUID   `json:"bill_item_id" db:"bill_item_id"`
	SessionID                uuid.UUID   `json:"session_id" db:"session_id"`
	Description              string      `json:"description" db:"description"`
	Amount                   float64     `json:"amount" db:"amount"`
	Category                 *string     `json:"category,omitempty" db:"category"`
	IncludesServiceCharge    bool        `json:"includes_service_charge" db:"includes_service_charge"`
	IncludesTax              bool        `json:"includes_tax" db:"includes_tax"`
	CreatedAt                time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time   `json:"updated_at" db:"updated_at"`
	ParticipantIDs           []uuid.UUID `json:"participant_ids" db:"-"`
}

// NewBillItem creates a new bill item. participantIDs may be nil/empty to
// apply the bill to every participant in the session.
func NewBillItem(sessionID uuid.UUID, description string, amount float64, category *string, participantIDs []uuid.UUID) *BillItem {
	now := time.Now()
	return &BillItem{
		BillItemID:     uuid.New(),
		SessionID:      sessionID,
		Description:    description,
		Amount:         amount,
		Category:       category,
		CreatedAt:      now,
		UpdatedAt:      now,
		ParticipantIDs: participantIDs,
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

// SetFeeFlags updates whether service charge and tax apply to this item.
// nil pointers mean "leave unchanged".
func (b *BillItem) SetFeeFlags(includesServiceCharge, includesTax *bool) {
	if includesServiceCharge != nil {
		b.IncludesServiceCharge = *includesServiceCharge
	}
	if includesTax != nil {
		b.IncludesTax = *includesTax
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
