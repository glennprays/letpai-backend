package billing

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/entity"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// UploadBillImageRequest is the request body for uploading a bill image.
type UploadBillImageRequest struct {
	Image      string `json:"image" validate:"required"`
	FileName   string `json:"file_name" validate:"required"`
	FileFormat string `json:"file_format,omitempty"`
}

// BillImageItem represents a single bill image in API responses.
type BillImageItem struct {
	BillImageID  string  `json:"bill_image_id"`
	SessionID    string  `json:"session_id"`
	ImageURL     string  `json:"image_url"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
	FileName     string  `json:"file_name"`
	FileFormat   string  `json:"file_format"`
	FileSize     int64   `json:"file_size"`
	UploadedAt   string  `json:"uploaded_at"`
	Ordinal      int     `json:"ordinal"`
}

// UploadBillImageUseCase handles uploading a bill image to a session.
type UploadBillImageUseCase struct {
	sessionRepo   ports.SessionRepository
	billImageRepo ports.BillImageRepository
	imageService  *service.ImageService
}

// NewUploadBillImageUseCase creates a new upload bill image use case.
func NewUploadBillImageUseCase(
	sessionRepo ports.SessionRepository,
	billImageRepo ports.BillImageRepository,
	imageService *service.ImageService,
) *UploadBillImageUseCase {
	return &UploadBillImageUseCase{
		sessionRepo:   sessionRepo,
		billImageRepo: billImageRepo,
		imageService:  imageService,
	}
}

// Execute uploads a bill image and persists the metadata.
func (uc *UploadBillImageUseCase) Execute(ctx context.Context, userID, sessionID string, req *UploadBillImageRequest) (*BillImageItem, error) {
	// Verify session exists and belongs to user
	session, err := uc.sessionRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if !session.IsActive() {
		return nil, domain.NewError(domain.ErrBadRequest, errors.New("cannot upload images to a completed or cancelled session"))
	}

	// Upload image to S3 under the bill-images/ prefix
	result, err := uc.imageService.UploadFromBase64WithPrefix(ctx, req.Image, req.FileName, "bill-images/")
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Get next ordinal
	ordinal, err := uc.billImageRepo.NextOrdinal(ctx, sessionID)
	if err != nil {
		return nil, domain.NewError(domain.ErrInternalFailure, err)
	}

	// Determine file format
	fileFormat := result.FileFormat
	if req.FileFormat != "" {
		fileFormat = req.FileFormat
	}

	// Create and persist the bill image entity
	sessionUUID := uuid.MustParse(sessionID)
	// Store the S3 key (not the s3:// URL) so presigned GET works
	billImage := entity.NewBillImage(sessionUUID, result.FileName, result.FileName, fileFormat, result.Size, ordinal)

	if err := uc.billImageRepo.Create(ctx, billImage); err != nil {
		return nil, err
	}

	return &BillImageItem{
		BillImageID:  billImage.BillImageID.String(),
		SessionID:    billImage.SessionID.String(),
		ImageURL:     billImage.ImageURL,
		ThumbnailURL: billImage.ThumbnailURL,
		FileName:     billImage.FileName,
		FileFormat:   billImage.FileFormat,
		FileSize:     billImage.FileSize,
		UploadedAt:   billImage.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Ordinal:      billImage.Ordinal,
	}, nil
}
