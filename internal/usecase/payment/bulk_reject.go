package payment

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// BulkRejectRequest represents the request to bulk reject payments
type BulkRejectRequest struct {
	ProofIDs        []string `json:"proof_ids" validate:"required,min=1"`
	RejectionReason string   `json:"rejection_reason" validate:"required,min=5,max=500"`
}

// BulkRejectResponse represents the response after bulk rejecting payments
type BulkRejectResponse struct {
	RejectedCount int    `json:"rejected_count"`
	SkippedCount  int    `json:"skipped_count"`
	Message       string `json:"message"`
}

// BulkRejectUseCase handles bulk rejecting payment proofs
type BulkRejectUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
}

// NewBulkRejectUseCase creates a new bulk reject use case
func NewBulkRejectUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
) *BulkRejectUseCase {
	return &BulkRejectUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
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

		rejectedCount++
	}

	message := ""
	if rejectedCount > 0 {
		message = "Payments rejected successfully"
	} else {
		message = "No payments were rejected"
	}

	return &BulkRejectResponse{
		RejectedCount: rejectedCount,
		SkippedCount:  skippedCount,
		Message:       message,
	}, nil
}
