package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
	"github.com/glennprays/letpai-backend/internal/service"
	"github.com/google/uuid"
)

// BulkRejectRequest represents a request to bulk reject payments
type BulkRejectRequest struct {
	ProofIDs        []string `json:"proof_ids" validate:"required,min=1"`
	RejectionReason string   `json:"rejection_reason" validate:"required,min=5,max=500"`
}

// BulkRejectResponse represents response after bulk rejecting payments
type BulkRejectResponse struct {
	RejectedCount int    `json:"rejected_count"`
	SkippedCount  int    `json:"skipped_count"`
	Message       string `json:"message"`
}

// BulkRejectUseCase handles bulk rejecting payment proofs
type BulkRejectUseCase struct {
	participantRepo     ports.ParticipantRepository
	sessionRepo         ports.SessionRepository
	contactRepo         ports.ContactRepository
	whatsappSvc         *service.WhatsAppService
	notificationLogRepo ports.NotificationLogRepository
}

// NewBulkRejectUseCase creates a new bulk reject use case
func NewBulkRejectUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	contactRepo ports.ContactRepository,
	whatsappSvc *service.WhatsAppService,
	notificationLogRepo ports.NotificationLogRepository,
) *BulkRejectUseCase {
	return &BulkRejectUseCase{
		participantRepo:     participantRepo,
		sessionRepo:         sessionRepo,
		contactRepo:         contactRepo,
		whatsappSvc:         whatsappSvc,
		notificationLogRepo: notificationLogRepo,
	}
}

// Execute bulk rejects payment proofs
func (uc *BulkRejectUseCase) Execute(ctx context.Context, userID string, req *BulkRejectRequest) (*BulkRejectResponse, error) {
	rejectedCount := 0
	skippedCount := 0

	for _, proofID := range req.ProofIDs {
		// Get participant
		participant, err := uc.participantRepo.FindByID(ctx, proofID)
		if err != nil {
			skippedCount++
			continue
		}

		// Get session to verify ownership
		session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), userID)
		if err != nil {
			skippedCount++
			continue
		}

		// Verify session belongs to user
		if session.UserID.String() != userID {
			skippedCount++
			continue
		}

		// Check if payment is in submitted status
		if participant.PaymentStatus != valueobject.PaymentStatusSubmitted {
			skippedCount++
			continue
		}

		// Reject payment
		if err := participant.RejectPayment(req.RejectionReason); err != nil {
			skippedCount++
			continue
		}

		// Clear proof URL on rejection
		participant.PaymentProofURL = nil

		// Update participant
		if err := uc.participantRepo.Update(ctx, participant); err != nil {
			skippedCount++
			continue
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

			messageID, err := uc.whatsappSvc.SendNotification(ctx, whatsappNumber, message)
			if err != nil {
				// Log failure but don't block the rejection
				errMsg := err.Error()
				log := &entity.NotificationLog{
					LogID:            uuid.New(),
					ParticipantID:    participant.ParticipantID,
					NotificationType: valueobject.NotificationTypeRejection,
					MessageContent:   message,
					SentAt:           time.Now(),
					Status:           entity.NotificationStatusFailed,
					ErrorMessage:     &errMsg,
				}
				uc.notificationLogRepo.Create(ctx, log)
			} else {
				log := &entity.NotificationLog{
					LogID:             uuid.New(),
					ParticipantID:     participant.ParticipantID,
					NotificationType:  valueobject.NotificationTypeRejection,
					WhatsAppMessageID: &messageID,
					MessageContent:    message,
					SentAt:            time.Now(),
					Status:            entity.NotificationStatusQueued,
				}
				uc.notificationLogRepo.Create(ctx, log)
			}
		}

		rejectedCount++
	}

	message := ""
	if rejectedCount > 0 {
		message = fmt.Sprintf("%d payment(s) rejected successfully", rejectedCount)
	} else {
		message = "No payments were rejected"
	}

	return &BulkRejectResponse{
		RejectedCount: rejectedCount,
		SkippedCount:  skippedCount,
		Message:       message,
	}, nil
}
