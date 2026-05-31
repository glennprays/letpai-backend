package entity

import (
	"time"

	"github.com/google/uuid"
)

// BillImage represents an uploaded bill/receipt image attached to a session.
type BillImage struct {
	BillImageID  uuid.UUID `json:"bill_image_id" db:"bill_image_id"`
	SessionID    uuid.UUID `json:"session_id" db:"session_id"`
	ImageURL     string    `json:"image_url" db:"image_url"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty" db:"thumbnail_url"`
	FileName     string    `json:"file_name" db:"file_name"`
	FileFormat   string    `json:"file_format" db:"file_format"`
	FileSize     int64     `json:"file_size" db:"file_size"`
	Ordinal      int       `json:"ordinal" db:"ordinal"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// NewBillImage creates a new BillImage entity.
func NewBillImage(sessionID uuid.UUID, imageURL, fileName, fileFormat string, fileSize int64, ordinal int) *BillImage {
	now := time.Now()
	return &BillImage{
		BillImageID: uuid.New(),
		SessionID:   sessionID,
		ImageURL:    imageURL,
		FileName:    fileName,
		FileFormat:  fileFormat,
		FileSize:    fileSize,
		Ordinal:     ordinal,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
