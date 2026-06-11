package payment

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// ApprovePaymentRequest represents the request to approve payment
type ApprovePaymentRequest struct {
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// ApprovePaymentResponse represents the response after approving payment
type ApprovePaymentResponse struct {
	PaymentStatus string `json:"payment_status"`
	ApprovedAt    string `json:"approved_at"`
}

// ApprovePaymentUseCase handles approving payment proof
type ApprovePaymentUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
}

// NewApprovePaymentUseCase creates a new approve payment use case
func NewApprovePaymentUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
) *ApprovePaymentUseCase {
	return &ApprovePaymentUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
	}
}

// Execute approves a payment proof
func (uc *ApprovePaymentUseCase) Execute(ctx context.Context, userID, participantID string, req *ApprovePaymentRequest) (*ApprovePaymentResponse, error) {
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
		return nil, domain.NewError(domain.ErrForbidden, errors.New("you can only approve payments for your own sessions"))
	}

	// Check if payment is in submitted status
	if participant.PaymentStatus != valueobject.PaymentStatusSubmitted {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("payment is not in submitted status"))
	}

	// Approve payment
	if err := participant.ApprovePayment(); err != nil {
		return nil, domain.NewError(domain.ErrInvalidPaymentStatusTransition, err)
	}

	// Update participant
	if err := uc.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	// Atomically complete the session iff every participant is now paid. A
	// single guarded UPDATE avoids the race where two concurrent approvals both
	// read a stale "still unpaid" snapshot and neither completes the session.
	_, _ = uc.sessionRepo.CompleteIfAllPaid(ctx, participant.SessionID.String())

	return &ApprovePaymentResponse{
		PaymentStatus: participant.PaymentStatus.String(),
		ApprovedAt:    participant.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}
