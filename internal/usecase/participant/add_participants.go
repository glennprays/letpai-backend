package participant

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/google/uuid"
)

// AddParticipantRequest represents a single participant to add
type AddParticipantRequest struct {
	ContactID      *string `json:"contact_id,omitempty"`
	CustomName     string  `json:"custom_name,omitempty"`
	CustomWhatsApp string  `json:"custom_whatsapp,omitempty"`
}

// AddParticipantsRequest represents the request to add participants
type AddParticipantsRequest struct {
	Participants []AddParticipantRequest `json:"participants" validate:"required,min=1"`
}

// ParticipantItem represents a participant in the response
type ParticipantItem struct {
	ParticipantID  string  `json:"participant_id"`
	ContactID      *string `json:"contact_id,omitempty"`
	Name           string  `json:"name"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	ShareAmount    float64 `json:"share_amount"`
	PaymentStatus  string  `json:"payment_status"`
}

// AddParticipantsResponse represents the response after adding participants
type AddParticipantsResponse struct {
	Message      string             `json:"message"`
	Participants []*ParticipantItem `json:"participants"`
}

// AddParticipantsUseCase handles adding participants to a session
type AddParticipantsUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
	contactRepo     ports.ContactRepository
}

// NewAddParticipantsUseCase creates a new add participants use case
func NewAddParticipantsUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	contactRepo ports.ContactRepository,
) *AddParticipantsUseCase {
	return &AddParticipantsUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		contactRepo:     contactRepo,
	}
}

// Execute adds participants to a session
func (uc *AddParticipantsUseCase) Execute(ctx context.Context, userID, sessionID string, req *AddParticipantsRequest) (*AddParticipantsResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow adding participants to active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot add participants to a completed or cancelled session"))
	}

	sessionUUID := uuid.MustParse(sessionID)

	participantItems := make([]*ParticipantItem, 0, len(req.Participants))

	for _, p := range req.Participants {
		var participant *ParticipantItem

		if p.ContactID != nil {
			// Add participant from contact
			contactUUID, err := uuid.Parse(*p.ContactID)
			if err != nil {
				return nil, domain.NewError(domain.ErrBadRequest, errors.New("invalid contact ID"))
			}

			// Verify contact exists and belongs to user
			contact, err := uc.contactRepo.FindByID(ctx, *p.ContactID, userID)
			if err != nil {
				return nil, err
			}

			participantEnt := entity.NewParticipantFromContact(sessionUUID, contactUUID)
			if err := uc.participantRepo.Create(ctx, participantEnt); err != nil {
				return nil, err
			}

			participant = &ParticipantItem{
				ParticipantID:  participantEnt.ParticipantID.String(),
				ContactID:      p.ContactID,
				Name:           contact.Name,
				WhatsAppNumber: contact.WhatsAppNumber,
				ShareAmount:    participantEnt.ShareAmount,
				PaymentStatus:  participantEnt.PaymentStatus.String(),
			}
		} else {
			// Add custom participant
			if p.CustomName == "" {
				return nil, domain.NewError(domain.ErrBadRequest, errors.New("custom_name is required for custom participants"))
			}

			participantEnt := entity.NewCustomParticipant(sessionUUID, p.CustomName, p.CustomWhatsApp)
			if err := uc.participantRepo.Create(ctx, participantEnt); err != nil {
				return nil, err
			}

			cid := p.ContactID
			participant = &ParticipantItem{
				ParticipantID:  participantEnt.ParticipantID.String(),
				ContactID:      cid,
				Name:           p.CustomName,
				WhatsAppNumber: p.CustomWhatsApp,
				ShareAmount:    participantEnt.ShareAmount,
				PaymentStatus:  participantEnt.PaymentStatus.String(),
			}
		}

		participantItems = append(participantItems, participant)
	}

	return &AddParticipantsResponse{
		Message:      "Participants added successfully",
		Participants: participantItems,
	}, nil
}
