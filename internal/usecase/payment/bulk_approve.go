package payment

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/domain/valueobject"
)

// BulkApproveRequest represents the request to bulk approve payments
type BulkApproveRequest struct {
	ProofIDs []string `json:"proof_ids" validate:"required,min=1"`
}

// BulkApproveResponse represents the response after bulk approving payments
type BulkApproveResponse struct {
	ApprovedCount    int  `json:"approved_count"`
	SkippedCount     int  `json:"skipped_count"`
	SessionCompleted bool `json:"session_completed"`
}

// BulkApproveUseCase handles bulk approving payment proofs
type BulkApproveUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
}

// NewBulkApproveUseCase creates a new bulk approve use case
func NewBulkApproveUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
) *BulkApproveUseCase {
	return &BulkApproveUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
	}
}

// Execute bulk approves payment proofs
func (uc *BulkApproveUseCase) Execute(ctx context.Context, userID string, req *BulkApproveRequest) (*BulkApproveResponse, error) {
	approvedCount := 0
	skippedCount := 0
	sessionIDs := make(map[string]bool)

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

		// Approve payment
		if err := participant.ApprovePayment(); err != nil {
			skippedCount++
			continue
		}

		// Update participant
		if err := uc.participantRepo.Update(ctx, participant); err != nil {
			skippedCount++
			continue
		}

		approvedCount++
		sessionIDs[participant.SessionID.String()] = true
	}

	// Check if any sessions are now complete
	sessionCompleted := false
	for sessionID := range sessionIDs {
		if complete, _ := uc.checkSessionCompletion(ctx, sessionID); complete {
			// Mark session as completed
			session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
			if err == nil {
				if err := session.Complete(); err == nil {
					_ = uc.sessionRepo.Update(ctx, session)
					sessionCompleted = true
				}
			}
		}
	}

	return &BulkApproveResponse{
		ApprovedCount:    approvedCount,
		SkippedCount:     skippedCount,
		SessionCompleted: sessionCompleted,
	}, nil
}

// checkSessionCompletion checks if all participants in a session have paid
func (uc *BulkApproveUseCase) checkSessionCompletion(ctx context.Context, sessionID string) (bool, error) {
	participants, err := uc.participantRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return false, err
	}

	for _, p := range participants {
		if !p.IsPaid() {
			return false, nil
		}
	}

	return true, nil
}
