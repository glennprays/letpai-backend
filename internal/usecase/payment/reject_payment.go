package payment

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// RejectPaymentRequest represents the request to reject payment
type RejectPaymentRequest struct {
	RejectionReason string `json:"rejection_reason" validate:"required,min=5,max=500"`
}

// RejectPaymentResponse represents the response after rejecting payment
type RejectPaymentResponse struct {
	PaymentStatus   string `json:"payment_status"`
	RejectionCount  int    `json:"rejection_count"`
	RejectedAt      string `json:"rejected_at"`
	RejectionReason string `json:"rejection_reason"`
}

// RejectPaymentUseCase handles rejecting payment proof
type RejectPaymentUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
}

// NewRejectPaymentUseCase creates a new reject payment use case
func NewRejectPaymentUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
) *RejectPaymentUseCase {
	return &RejectPaymentUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
	}
}

// Execute rejects a payment proof
func (uc *RejectPaymentUseCase) Execute(ctx context.Context, userID, participantID string, req *RejectPaymentRequest) (*RejectPaymentResponse, error) {
	// Get participant
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// Get session to verify ownership
	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), userID)
	if err != nil {
		return nil, err
	}

	// Verify session belongs to user
	if session.UserID.String() != userID {
		return nil, domain.NewError(domain.ErrForbidden, errors.New("you can only reject payments for your own sessions"))
	}

	// Check if payment is in submitted status
	if participant.PaymentStatus != valueobject.PaymentStatusSubmitted {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("payment is not in submitted status"))
	}

	// Reject payment with reason (increments rejection count)
	if err := participant.RejectPayment(req.RejectionReason); err != nil {
		return nil, domain.NewError(domain.ErrInvalidPaymentStatusTransition, err)
	}

	// Clear proof URL on rejection
	participant.PaymentProofURL = nil

	// Update participant
	if err := uc.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	return &RejectPaymentResponse{
		PaymentStatus:   participant.PaymentStatus.String(),
		RejectionCount:  participant.RejectionCount,
		RejectedAt:      participant.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		RejectionReason: req.RejectionReason,
	}, nil
}
