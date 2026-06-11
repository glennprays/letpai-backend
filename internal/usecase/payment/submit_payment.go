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

// Execute submits payment proof for a participant.
//
// Concurrency: two simultaneous submits both pass the initial pending-check
// and both upload an image, but only one wins the atomic
// MarkSubmittedWithProof DB update. The loser's image is orphaned in object
// storage — acceptable trade-off; a periodic cleanup job can collect URLs
// not referenced by any participant.
func (uc *SubmitPaymentUseCase) Execute(ctx context.Context, participantID string, req *SubmitPaymentRequest) (*SubmitPaymentResponse, error) {
	participant, err := uc.participantRepo.FindByID(ctx, participantID)
	if err != nil {
		return nil, err
	}

	// A host-driven "Mark as paid" closes the participant out before a
	// proof is uploaded. Subsequent uploads should fail with a clear
	// message rather than silently overwriting the manual closure.
	if participant.IsPaid() && participant.PaidManually {
		return nil, domain.NewError(domain.ErrConflict, errors.New("your host already marked this payment as complete"))
	}

	if !participant.HasPendingPayment() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("payment has already been submitted"))
	}

	session, err := uc.sessionRepo.FindByID(ctx, participant.SessionID.String(), "")
	if err != nil {
		return nil, err
	}

	if session.SessionDate != nil && session.SessionDate.Before(time.Now()) {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("this session has expired and no longer accepts payments"))
	}

	// Enforce the same 7-day link lifetime the payment page advertises
	// (get_payment_page computes CreatedAt+7d for the is_expired flag). Without
	// this, an "expired" link still accepts proof uploads server-side.
	if time.Now().After(session.CreatedAt.Add(7 * 24 * time.Hour)) {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("this payment link has expired and no longer accepts payments"))
	}

	if err := uc.imageService.ValidateBase64(req.ProofImage); err != nil {
		return nil, domain.NewError(domain.ErrBadRequest, err)
	}

	uploadResult, err := uc.imageService.UploadFromBase64(ctx, req.ProofImage, req.FileName)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// UploadResult.URL is the raw object reference (s3://bucket/key) — not
	// browser-resolvable. UploadResult.PublicURL honours CDN_URL and produces
	// the https URL we actually want stored and returned. Reading the wrong
	// field here was leaking s3:// URIs into payment_proof_url, which then
	// broke every <img> render on the FE.
	claimed, err := uc.participantRepo.MarkSubmittedWithProof(ctx, participantID, uploadResult.PublicURL)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("payment has already been submitted"))
	}

	return &SubmitPaymentResponse{
		ProofID:    participantID,
		Status:     "submitted",
		UploadedAt: uploadResult.UploadedAt.Format("2006-01-02T15:04:05Z07:00"),
		ProofURL:   uploadResult.PublicURL,
	}, nil
}
