package payment

import (
	"context"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// SubmitPaymentRequest represents the request to submit payment proof
type SubmitPaymentRequest struct {
	ProofImage string `json:"proof_image" validate:"required"`
	FileName   string `json:"file_name" validate:"required"`
	FileFormat string `json:"file_format,omitempty"`
}

// SubmitPaymentResponse represents the response after submitting payment proof
type SubmitPaymentResponse struct {
	ProofID    string `json:"proof_id"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploaded_at"`
	ProofURL   string `json:"proof_url"`
}

// SubmitPaymentUseCase handles submitting payment proof
type SubmitPaymentUseCase struct {
	participantRepo ports.ParticipantRepository
	sessionRepo     ports.SessionRepository
	imageService    *service.ImageService
}

// NewSubmitPaymentUseCase creates a new submit payment use case
func NewSubmitPaymentUseCase(
	participantRepo ports.ParticipantRepository,
	sessionRepo ports.SessionRepository,
	imageService *service.ImageService,
) *SubmitPaymentUseCase {
	return &SubmitPaymentUseCase{
		participantRepo: participantRepo,
		sessionRepo:     sessionRepo,
		imageService:    imageService,
	}
}

// Execute submits payment proof for a participant
func (uc *SubmitPaymentUseCase) Execute(ctx context.Context, participantID string, req *SubmitPaymentRequest) (*SubmitPaymentResponse, error) {
	// Get participant
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// Check if payment is still pending
	if !participant.HasPendingPayment() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("payment has already been submitted"))
	}

	// Get session to check expiry
	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), "")
	if err != nil {
		return nil, err
	}

	// Check if session has expired
	if session.SessionDate != nil && session.SessionDate.Before(time.Now()) {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("this session has expired and no longer accepts payments"))
	}

	// Validate and upload image
	if err := uc.imageService.ValidateBase64(req.ProofImage); err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	uploadResult, err := uc.imageService.UploadFromBase64(ctx, req.ProofImage, req.FileName)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Submit payment
	if err := participant.SubmitPayment(uploadResult.URL); err != nil {
		return nil, domain.NewError(domain.ErrInvalidPaymentStatusTransition, err)
	}

	// Update participant
	if err := uc.participantRepo.Update(ctx, participant); err != nil {
		return nil, err
	}

	return &SubmitPaymentResponse{
		ProofID:    participant.ParticipantID.String(),
		Status:     participant.PaymentStatus.String(),
		UploadedAt: uploadResult.UploadedAt.Format("2006-01-02T15:04:05Z07:00"),
		ProofURL:   uploadResult.URL,
	}, nil
}
