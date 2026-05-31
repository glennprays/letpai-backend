package billing

import (
	"context"
	"errors"

	"github.com/glennprays/letpai-backend/domain"
	"github.com/glennprays/letpai-backend/domain/ports"
	"github.com/glennprays/letpai-backend/internal/service"
)

// DeleteBillImageUseCase handles deleting a bill image from a session.
type DeleteBillImageUseCase struct {
	sessionRepo   ports.SessionRepository
	billImageRepo ports.BillImageRepository
	imageService  *service.ImageService
}

// NewDeleteBillImageUseCase creates a new delete bill image use case.
func NewDeleteBillImageUseCase(
	sessionRepo ports.SessionRepository,
	billImageRepo ports.BillImageRepository,
	imageService *service.ImageService,
) *DeleteBillImageUseCase {
	return &DeleteBillImageUseCase{
		sessionRepo:   sessionRepo,
		billImageRepo: billImageRepo,
		imageService:  imageService,
	}
}

// DeleteBillImageResponse is the response for deleting a bill image.
type DeleteBillImageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Execute soft-deletes a bill image and removes it from S3 (best-effort).
func (uc *DeleteBillImageUseCase) Execute(ctx context.Context, userID, sessionID, imageID string) (*DeleteBillImageResponse, error) {
	// Verify session exists and belongs to user
	if _, err := uc.sessionRepo.FindByID(ctx, sessionID, userID); err != nil {
		return nil, err
	}

	// Load the bill image to verify it belongs to this session
	img, err := uc.billImageRepo.FindByID(ctx, imageID)
	if err != nil {
		return nil, err
	}
	if img.SessionID.String() != sessionID {
		return nil, domain.NewError(domain.ErrNotFound, errors.New("image not found in this session"))
	}

	// Soft-delete from database
	if err := uc.billImageRepo.Delete(ctx, imageID); err != nil {
		return nil, err
	}

	// Best-effort: delete from S3 (don't fail if S3 delete fails)
	_ = uc.imageService.DeleteWithPrefix(ctx, img.ImageURL, "bill-images/")

	return &DeleteBillImageResponse{
		Success: true,
		Message: "Bill image deleted successfully",
	}, nil
}
