package payment

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// GetPaymentProofResponse represents the payment proof data
type GetPaymentProofResponse struct {
	ParticipantID   string  `json:"participant_id"`
	ParticipantName string  `json:"participant_name"`
	ShareAmount     float64 `json:"share_amount"`
	PaymentStatus   string  `json:"payment_status"`
	ProofURL        *string `json:"proof_url,omitempty"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
	ProofUploadedAt string  `json:"proof_uploaded_at,omitempty"`
}

// GetPaymentProofUseCase retrieves payment proof information
type GetPaymentProofUseCase struct {
	participantRepo ports.ParticipantRepository
}

// NewGetPaymentProofUseCase creates a new get payment proof use case
func NewGetPaymentProofUseCase(participantRepo ports.ParticipantRepository) *GetPaymentProofUseCase {
	return &GetPaymentProofUseCase{
		participantRepo: participantRepo,
	}
}

// Execute retrieves payment proof for a participant
func (uc *GetPaymentProofUseCase) Execute(ctx context.Context, participantID string) (*GetPaymentProofResponse, error) {
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	resp := &GetPaymentProofResponse{
		ParticipantID:   participant.ParticipantID.String(),
		ParticipantName: participant.CustomName,
		ShareAmount:     participant.ShareAmount,
		PaymentStatus:   participant.PaymentStatus.String(),
		ProofURL:        participant.PaymentProofURL,
		RejectionReason: participant.RejectionReason,
	}

	if participant.UpdatedAt.After(participant.JoinedAt) {
		resp.ProofUploadedAt = participant.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return resp, nil
}
