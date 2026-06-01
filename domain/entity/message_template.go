package entity

import (
	"time"

	"github.com/google/uuid"
)

// MessageTemplate carries the Go text/template body for one of the
// admin-managed message types (initial notification, reminder, OTP,
// etc.). The `key` is the stable lookup id used by call sites — e.g.
// "session_notification". `Variables` is a hint list of variable names
// the template uses (e.g. ["ParticipantName","Share","URL"]) so the
// admin UI can show what placeholders are available without parsing
// the body.
type MessageTemplate struct {
	TemplateID  uuid.UUID `json:"template_id" db:"template_id"`
	Key         string    `json:"key" db:"key"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	Body        string    `json:"body" db:"body"`
	Variables   []string  `json:"variables" db:"-"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
