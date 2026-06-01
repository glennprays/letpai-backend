package payment

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// MarkPaidWithoutProofRequest carries no body — the action is identified by
// the path param participant_id. The session-ownership check uses the
// caller's userID.
type MarkPaidWithoutProofRequest struct{}

// MarkPaidWithoutProofResponse mirrors the approve/reject shape used by
// the rest of the maker endpoints.
type MarkPaidWithoutProofResponse struct {
	Message       string `json:"message"`
	ParticipantID string `json:"participant_id"`
	PaymentStatus string `json:"payment_status"`
	PaidManually  bool   `json:"paid_manually"`
}

// MarkPaidWithoutProofUseCase lets a session host mark a participant as
// paid without a proof upload. Useful for cash payments, manual transfers
// you've already confirmed, etc.
//
// Idempotency: a second call when the participant is already Paid returns
// a 409 so the FE can show "already complete" rather than silently
// double-applying. Manual completion clears any prior proof URL and
// rejection metadata.
type MarkPaidWithoutProofUseCase struct {
	sessionRepo     ports.SessionRepository
	participantRepo ports.ParticipantRepository
}

// NewMarkPaidWithoutProofUseCase wires the use case.
func NewMarkPaidWithoutProofUseCase(
	sessionRepo ports.SessionRepository,
	participantRepo ports.ParticipantRepository,
) *MarkPaidWithoutProofUseCase {
	return &MarkPaidWithoutProofUseCase{
		sessionRepo:     sessionRepo,
		participantRepo: participantRepo,
	}
}

// Execute marks the participant as paid manually. Verifies that the
// participant belongs to a session owned by the caller.
func (uc *MarkPaidWithoutProofUseCase) Execute(ctx context.Context, userID, participantID string) (*MarkPaidWithoutProofResponse, error) {
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}
	if participant == nil {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("participant not found"))
	}

	if _, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), userID); err != nil {
		return nil, err
	}

	if participant.PaymentStatus == valueobject.PaymentStatusPaid {
		return nil, domain.NewError(domain.ErrConflict, errors.New("payment is already complete"))
	}

	if err := participant.MarkPaidManually(); err != nil {
		// Should only happen if a non-Paid state failed validation;
		// surface as a generic bad-request to keep the API consistent.
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	if err := uc.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	return &MarkPaidWithoutProofResponse{
		Message:       "Participant marked as paid",
		ParticipantID: participant.ParticipantID.String(),
		PaymentStatus: string(valueobject.PaymentStatusPaid),
		PaidManually:  true,
	}, nil
}

// Ensure the package still imports the entity type so refactors that drop
// helper references don't break the import graph.
var _ = entity.SessionParticipant{}
