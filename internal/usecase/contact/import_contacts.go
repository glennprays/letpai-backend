package contact

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// ImportContactsRequest represents a request to import contacts
type ImportContactsRequest struct {
	Contacts []ImportContact `json:"contacts" validate:"required,min=1"`
	GroupID  *string         `json:"group_id,omitempty" validate:"omitempty,uuid"`
}

// ImportContact represents a single contact to import
type ImportContact struct {
	Name           string `json:"name" validate:"required,min=1,max=100"`
	WhatsAppNumber string `json:"whatsapp_number" validate:"required,min=10,max=20"`
}

// ImportContactsResponse represents response after importing contacts
type ImportContactsResponse struct {
	ImportedCount     int      `json:"imported_count"`
	SkippedCount      int      `json:"skipped_count"`
	SkippedDuplicates []string `json:"skipped_duplicates,omitempty"`
	Message           string   `json:"message"`
}

// ImportContactsUseCase handles importing contacts
type ImportContactsUseCase struct {
	contactRepo ports.ContactRepository
}

// NewImportContactsUseCase creates a new import contacts use case
func NewImportContactsUseCase(contactRepo ports.ContactRepository) *ImportContactsUseCase {
	return &ImportContactsUseCase{
		contactRepo: contactRepo,
	}
}

// Execute imports contacts with deduplication
func (uc *ImportContactsUseCase) Execute(ctx context.Context, userID string, req *ImportContactsRequest) (*ImportContactsResponse, error) {
	// Parse group ID if provided
	var groupID *uuid.UUID
	if req.GroupID != nil {
		gid, err := uuid.Parse(*req.GroupID)
		if err != nil {
			return nil, domain.NewError(domain.ErrBadRequest, err)
		}
		groupID = &gid
	}

	// Process contacts
	importedCount := 0
	skippedCount := 0
	skippedDuplicates := []string{}
	contactsToCreate := make([]*entity.Contact, 0, len(req.Contacts))

	for _, contactReq := range req.Contacts {
		// Check if contact already exists for this user
		exists, err := uc.contactRepo.ExistsByWhatsApp(ctx, userID, contactReq.WhatsAppNumber, nil)
		if err != nil {
			// Continue with other contacts
			skippedCount++
			continue
		}

		if exists {
			// Contact already exists, skip
			skippedDuplicates = append(skippedDuplicates, contactReq.WhatsAppNumber)
			skippedCount++
			continue
		}

		// Create new contact
		contact := &entity.Contact{
			ContactID:      uuid.New(),
			UserID:         uuid.MustParse(userID),
			Name:           contactReq.Name,
			WhatsAppNumber: contactReq.WhatsAppNumber,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Assign to group if provided
		if groupID != nil {
			contact.GroupID = groupID
		}

		contactsToCreate = append(contactsToCreate, contact)
		importedCount++
	}

	// Skip if no contacts to create
	if len(contactsToCreate) == 0 {
		return &ImportContactsResponse{
			ImportedCount:     0,
			SkippedCount:      skippedCount,
			SkippedDuplicates: skippedDuplicates,
			Message:           "No contacts to import",
		}, nil
	}

	// Bulk create all contacts
	if err := uc.contactRepo.BulkCreate(ctx, contactsToCreate); err != nil {
		return nil, err
	}

	// Build message
	message := ""
	if len(skippedDuplicates) > 0 {
		message = fmt.Sprintf("%d contact(s) imported, %d skipped (duplicates)", importedCount, skippedCount)
	} else {
		message = fmt.Sprintf("%d contact(s) imported", importedCount)
	}

	return &ImportContactsResponse{
		ImportedCount:     importedCount,
		SkippedCount:      skippedCount,
		SkippedDuplicates: skippedDuplicates,
		Message:           message,
	}, nil
}
