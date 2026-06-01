package participant

import (
	"context"
	"errors"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// ImportFromGroupRequest represents a request to import participants from a contact group
type ImportFromGroupRequest struct {
	SessionID string `json:"session_id" validate:"required"`
	GroupID   string `json:"group_id" validate:"required"`
}

// ImportFromGroupResponse represents response after importing participants
type ImportFromGroupResponse struct {
	ImportedCount int    `json:"imported_count"`
	Message       string `json:"message"`
}

// ImportFromGroupUseCase handles importing participants from a contact group
type ImportFromGroupUseCase struct {
	sessionRepo     ports.SessionRepository
	contactRepo     ports.ContactRepository
	participantRepo ports.ParticipantRepository
}

// NewImportFromGroupUseCase creates a new import from group use case
func NewImportFromGroupUseCase(
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	participantRepo ports.ParticipantRepository,
) *ImportFromGroupUseCase {
	return &ImportFromGroupUseCase{
		sessionRepo:     sessionRepo,
		contactRepo:     contactRepo,
		participantRepo: participantRepo,
	}
}

// Execute imports all contacts from a group as participants to a session.
// Verifies the caller owns the session before any work — without this any
// authenticated user could pollute another user's session by guessing the
// session UUID and triggering downstream notification spam.
func (uc *ImportFromGroupUseCase) Execute(ctx context.Context, userID, sessionID, groupID string) (*ImportFromGroupResponse, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	if _, err := uc.sessionRepo.FindByID(ctx, sessionID, userID); err != nil {
		return nil, err
	}

	// Get contacts in the group (FindAll already scopes by userID).
	contacts, err := uc.contactRepo.FindAll(ctx, userID, &ports.ContactFilterOptions{
		GroupID: &groupID,
		Limit:   10000, // Get all
	})
	if err != nil {
		return nil, err
	}

	if len(contacts.Contacts) == 0 {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("no contacts found in this group"))
	}

	// Create participants for each contact
	importedCount := 0
	for _, contact := range contacts.Contacts {
		participant := entity.NewParticipantFromContact(
			sessionUUID, contact.ContactID,
			contact.Name, contact.WhatsAppNumber,
		)
		if err := uc.participantRepo.Create(ctx, participant); err != nil {
			continue // Log error but continue with others
		}
		importedCount++
	}

	return &ImportFromGroupResponse{
		ImportedCount: importedCount,
		Message:       fmt.Sprintf("%d participants imported from contact group", importedCount),
	}, nil
}
