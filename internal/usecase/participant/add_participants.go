package participant

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/usecase/billing"
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

// AddParticipantsUseCase handles adding participants to a session.
//
// After successfully inserting all requested participants, the use
// case re-runs CalculateSplits if the session already has at least
// one bill -- otherwise existing participants would keep their old
// share_amount while the new ones sit at 0.
type AddParticipantsUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
	contactRepo     ports.ContactRepository
	billItemRepo    ports.BillItemRepository
	calcSplits      *billing.CalculateSplitsUseCase
}

// NewAddParticipantsUseCase creates a new add participants use case
func NewAddParticipantsUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
	contactRepo ports.ContactRepository,
	billItemRepo ports.BillItemRepository,
	calcSplits *billing.CalculateSplitsUseCase,
) *AddParticipantsUseCase {
	return &AddParticipantsUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
		contactRepo:     contactRepo,
		billItemRepo:    billItemRepo,
		calcSplits:      calcSplits,
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

			// Snapshot name + whatsapp from the contact so the unique
			// (session_id, custom_whatsapp) key carries a real value and
			// GetSessionDetail can render the row without re-joining.
			participantEnt := entity.NewParticipantFromContact(
				sessionUUID, contactUUID, contact.Name, contact.WhatsAppNumber,
			)
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

	// If the session already has bills, redistribute them across the
	// new full participant set so existing participants' share_amount
	// reflects the new headcount. CalculateSplits is safe to call when
	// nothing has changed; we still gate on bills > 0 to avoid the
	// "no bills yet" early-return path that would otherwise wrap the
	// no-op in a domain error.
	billTotal, _ := uc.billItemRepo.SumBySessionID(ctx, sessionID)
	if billTotal > 0 {
		_, _ = uc.calcSplits.Execute(ctx, userID, sessionID)
	}

	return &AddParticipantsResponse{
		Message:      "Participants added successfully",
		Participants: participantItems,
	}, nil
}
