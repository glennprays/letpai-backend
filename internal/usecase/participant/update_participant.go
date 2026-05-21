package participant

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
)

// UpdateParticipantRequest represents the request to update a participant's
// custom name / WhatsApp. Payment-status changes go through the dedicated
// /payments/* endpoints (SubmitPayment / ApprovePayment / RejectPayment).
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

// Execute updates a participant's custom name / whatsapp. Only valid for
// custom (unlinked) participants — for linked contacts, the underlying
// contact must be edited instead.
func (uc *UpdateParticipantUseCase) Execute(ctx context.Context, userID, sessionID, participantID string, req *UpdateParticipantRequest) (*UpdateParticipantResponse, error) {
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}

	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot update participants in a completed or cancelled session"))
	}

	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	if participant.SessionID.String() != sessionID {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("participant not found in this session"))
	}

	if !participant.IsCustom() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot rename linked contact participants"))
	}

	name := coalesceString(req.CustomName, participant.CustomName)
	whatsapp := coalesceString(req.CustomWhatsApp, participant.CustomWhatsApp)
	participant.Update(name, whatsapp)

	if err := uc.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	return &UpdateParticipantResponse{
		ParticipantID:  participant.ParticipantID.String(),
		Name:           participant.GetName(),
		WhatsAppNumber: participant.GetWhatsAppNumber(),
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
