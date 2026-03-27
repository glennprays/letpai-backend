package participant

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateParticipantRequest represents the request to update a participant
type UpdateParticipantRequest struct {
	CustomName     *string `json:"custom_name,omitempty"`
	CustomWhatsApp *string `json:"custom_whatsapp,omitempty"`
}

// UpdateParticipantResponse represents the response after updating a participant
type UpdateParticipantResponse struct {
	ParticipantID  string  `json:"participant_id"`
	Name           string  `json:"name"`
	WhatsAppNumber string  `json:"whatsapp_number"`
	ShareAmount    float64 `json:"share_amount"`
	PaymentStatus  string  `json:"payment_status"`
}

// UpdateParticipantUseCase handles updating a participant
type UpdateParticipantUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
}

// NewUpdateParticipantUseCase creates a new update participant use case
func NewUpdateParticipantUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
) *UpdateParticipantUseCase {
	return &UpdateParticipantUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
	}
}

// Execute updates a participant
func (uc *UpdateParticipantUseCase) Execute(ctx context.Context, userID, sessionID, participantID string, req *UpdateParticipantRequest) (*UpdateParticipantResponse, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	// Only allow updating participants in active sessions
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot update participants in a completed or cancelled session"))
	}

	// Fetch existing participant
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// Verify participant belongs to session
	if participant.SessionID.String() != sessionID {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("participant not found in this session"))
	}

	// Only allow updating custom participants
	if !participant.IsCustom() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot update linked contacts"))
	}

	// Update participant
	name := coalesceString(req.CustomName, participant.CustomName)
	whatsapp := coalesceString(req.CustomWhatsApp, participant.CustomWhatsApp)

	participant.Update(name, whatsapp)

	if err := uc.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	return &UpdateParticipantResponse{
		ParticipantID:  participant.ParticipantID.String(),
		Name:           name,
		WhatsAppNumber: whatsapp,
		ShareAmount:    participant.ShareAmount,
		PaymentStatus:  participant.PaymentStatus.String(),
	}, nil
}

func coalesceString(s *string, defaultVal string) string {
	if s != nil {
		return *s
	}
	return defaultVal
}
