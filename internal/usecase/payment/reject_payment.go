package payment

import (
	"context"
	"errors"
	"fmt"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/glennprays/letpai-backend/internal/service"
)

// RejectPaymentRequest represents a request to reject payment
type RejectPaymentRequest struct {
	RejectionReason string `json:"rejection_reason" validate:"required,min=5,max=500"`
}

// RejectPaymentResponse represents response after rejecting payment
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
	contactRepo     ports.ContactRepository
	notifier        *service.AsyncNotifier
}

// NewRejectPaymentUseCase creates a new reject payment use case
func NewRejectPaymentUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	notifier *service.AsyncNotifier,
) *RejectPaymentUseCase {
	return &RejectPaymentUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		contactRepo:     contactRepo,
		notifier:        notifier,
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

	// Send rejection notification via WhatsApp
	whatsappNumber := ""
	participantName := participant.CustomName
	if participant.ContactID != nil {
		contact, err := uc.contactRepo.FindByID(ctx, participant.ContactID.String(), userID)
		if err != nil {
			// Contact not found, use custom data
			whatsappNumber = participant.CustomWhatsApp
			participantName = participant.CustomName
		} else {
			whatsappNumber = contact.WhatsAppNumber
			participantName = contact.Name
		}
	} else {
		whatsappNumber = participant.CustomWhatsApp
		participantName = participant.CustomName
	}

	if whatsappNumber != "" {
		message := fmt.Sprintf("*Letpai - Payment Rejected*\n\n"+
			"Hi %s!\n\n"+
			"Unfortunately, your payment proof for the bill split session was rejected.\n\n"+
			"Reason: %s\n\n"+
			"Please upload a clearer proof and resubmit.\n\n"+
			"Thank you for using Letpai!",
			participantName, req.RejectionReason)

		// Enqueue into the delivery outbox instead of sending inline. The
		// rejection has already been persisted above; the WhatsApp send must
		// not block the HTTP response on the (slow) gateway, and the worker
		// retries it durably. A failed enqueue is best-effort — the rejection
		// still stands.
		_ = uc.notifier.Dispatch(participant.ParticipantID, valueobject.NotificationTypeRejection, whatsappNumber, message)
	}

	return &RejectPaymentResponse{
		PaymentStatus:   participant.PaymentStatus.String(),
		RejectionCount:  participant.RejectionCount,
		RejectedAt:      participant.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		RejectionReason: req.RejectionReason,
	}, nil
}
