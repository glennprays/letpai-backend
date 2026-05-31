package billing

import (
	"context"

	"github.com/glennprays/letpai-backend/domain/ports"
)

// GetBillImagesUseCase handles listing bill images for a session.
type GetBillImagesUseCase struct {
	sessionRepo   ports.SessionRepository
	billImageRepo ports.BillImageRepository
}

// NewGetBillImagesUseCase creates a new get bill images use case.
func NewGetBillImagesUseCase(
	sessionRepo ports.SessionRepository,
	billImageRepo ports.BillImageRepository,
) *GetBillImagesUseCase {
	return &GetBillImagesUseCase{
		sessionRepo:   sessionRepo,
		billImageRepo: billImageRepo,
	}
}

// Execute returns all bill images for a session.
func (uc *GetBillImagesUseCase) Execute(ctx context.Context, userID, sessionID string) ([]*BillImageItem, error) {
	// Verify session exists and belongs to user
	if _, err := uc.sessionRepo.FindByID(ctx, sessionID, userID); err != nil {
		return nil, err
	}

	images, err := uc.billImageRepo.FindBySessionID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	items := make([]*BillImageItem, 0, len(images))
	for _, img := range images {
		items = append(items, &BillImageItem{
			BillImageID:  img.BillImageID.String(),
			SessionID:    img.SessionID.String(),
			ImageURL:     img.ImageURL,
			ThumbnailURL: img.ThumbnailURL,
			FileName:     img.FileName,
			FileFormat:   img.FileFormat,
			FileSize:     img.FileSize,
			UploadedAt:   img.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Ordinal:      img.Ordinal,
		})
	}
	return items, nil
}
