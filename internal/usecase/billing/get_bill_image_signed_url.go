package billing

import (
	"context"
	"errors"
	"time"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// BillImageSignedURLResponse is the response for a signed URL request.
type BillImageSignedURLResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

// GetBillImageSignedUrlUseCase generates a presigned GET URL for a bill image.
type GetBillImageSignedUrlUseCase struct {
	sessionRepo   ports.SessionRepository
	billImageRepo ports.BillImageRepository
	imageService  *service.ImageService
}

// NewGetBillImageSignedUrlUseCase creates a new signed URL use case.
func NewGetBillImageSignedUrlUseCase(
	sessionRepo ports.SessionRepository,
	billImageRepo ports.BillImageRepository,
	imageService *service.ImageService,
) *GetBillImageSignedUrlUseCase {
	return &GetBillImageSignedUrlUseCase{
		sessionRepo:   sessionRepo,
		billImageRepo: billImageRepo,
		imageService:  imageService,
	}
}

// Execute generates a presigned GET URL for a bill image.
func (uc *GetBillImageSignedUrlUseCase) Execute(ctx context.Context, userID, sessionID, imageID string) (*BillImageSignedURLResponse, error) {
	// Verify session exists and belongs to user
	if _, err := uc.sessionRepo.FindByID(ctx, sessionID, userID); err != nil {
		return nil, err
	}

	// Load the bill image
	img, err := uc.billImageRepo.FindByID(ctx, imageID)
	if err != nil {
		return nil, err
	}

	// Verify the image belongs to this session
	if img.SessionID.String() != sessionID {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("image not found in this session"))
	}

	// Generate presigned GET URL
	s3Key := "bill-images/" + img.ImageURL
	expiresIn := 15 * time.Minute
	url, err := uc.imageService.GetPresignedGetURL(ctx, s3Key, expiresIn)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	expiresAt := time.Now().Add(expiresIn).UTC().Format("2006-01-02T15:04:05Z07:00")

	return &BillImageSignedURLResponse{
		URL:       url,
		ExpiresAt: expiresAt,
	}, nil
}
