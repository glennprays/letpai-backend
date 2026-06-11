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

// BulkApproveFailure explains why a single ID couldn't be approved so the
// caller can act on the result instead of just seeing a skipped count.
type BulkApproveFailure struct {
	ProofID string `json:"proof_id"`
	Reason  string `json:"reason"`
}

// BulkApproveResponse represents the response after bulk approving payments
type BulkApproveResponse struct {
	ApprovedCount    int                  `json:"approved_count"`
	SkippedCount     int                  `json:"skipped_count"`
	Failed           []BulkApproveFailure `json:"failed,omitempty"`
	SessionCompleted bool                 `json:"session_completed"`
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

// Execute bulk approves payment proofs.
// Per-ID failures are returned in the response so the caller can show "12
// approved, 2 failed (participant not found)" instead of just a skipped count.
func (uc *BulkApproveUseCase) Execute(ctx context.Context, userID string, req *BulkApproveRequest) (*BulkApproveResponse, error) {
	approvedCount := 0
	skippedCount := 0
	failures := make([]BulkApproveFailure, 0)
	sessionIDs := make(map[string]bool)

	skip := func(proofID, reason string) {
		skippedCount++
		failures = append(failures, BulkApproveFailure{ProofID: proofID, Reason: reason})
	}

	for _, proofID := range req.ProofIDs {
		participant, err := uc.participantRepo.FindByID(ctx, proofID)
		if err != nil {
			skip(proofID, "participant not found")
			continue
		}

		session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), userID)
		if err != nil {
			skip(proofID, "session not found or not yours")
			continue
		}

		if session.UserID.String() != userID {
			skip(proofID, "session belongs to another user")
			continue
		}

		if participant.PaymentStatus != valueobject.PaymentStatusSubmitted {
			skip(proofID, "payment is not in submitted state")
			continue
		}

		if err := participant.ApprovePayment(); err != nil {
			skip(proofID, "invalid state transition")
			continue
		}

		if err := uc.participantRepo.Update(ctx, participant); err != nil {
			skip(proofID, "database update failed")
			continue
		}

		approvedCount++
		sessionIDs[participant.SessionID.String()] = true
	}

	// Atomically complete any session whose participants are now all paid.
	// Ownership was already verified for every session in the map during the
	// approval loop above. The single guarded UPDATE avoids the read-then-write
	// completion race.
	sessionCompleted := false
	for sessionID := range sessionIDs {
		if done, _ := uc.sessionRepo.CompleteIfAllPaid(ctx, sessionID); done {
			sessionCompleted = true
		}
	}

	return &BulkApproveResponse{
		ApprovedCount:    approvedCount,
		SkippedCount:     skippedCount,
		Failed:           failures,
		SessionCompleted: sessionCompleted,
	}, nil
}
