package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ImageService handles image upload operations with MinIO/S3 and pure Go image libraries
type ImageService struct {
	s3Client    *minio.Client
	bucketName  string
	cdnURL      string // Optional CDN URL for serving images
	enableWebP  bool
	webPQuality int
	maxFileSize int64
}

// ImageConfig holds configuration for the image service
type ImageConfig struct {
	AWSEndpoint string // Optional: for S3-compatible services (e.g., MinIO)
	AWSRegion   string
	AWSAccessID string
	AWSSecret   string
	BucketName  string
	CDNURL      string
	EnableWebP  bool
	WebPQuality int
	MaxFileSize int64 // in bytes
}

// imageFormat represents the supported image formats
type imageFormat string

const (
	formatJPEG imageFormat = "jpeg"
	formatPNG  imageFormat = "png"
	formatWEBP imageFormat = "webp"
	formatGIF  imageFormat = "gif"
)

// NewImageService creates a new image service with MinIO/S3
func NewImageService(cfg *ImageConfig) (*ImageService, error) {
	if cfg == nil {
		return nil, errors.New("image config cannot be nil")
	}

	// Set defaults
	if cfg.WebPQuality == 0 {
		cfg.WebPQuality = 85 // Good balance between quality and size
	}
	if cfg.MaxFileSize == 0 {
		cfg.MaxFileSize = 5 * 1024 * 1024 // 5MB
	}

	// Determine if using SSL (HTTPS) based on endpoint
	useSSL := true
	endpoint := cfg.AWSEndpoint

	if endpoint != "" {
		// Check if endpoint specifies HTTP
		if strings.HasPrefix(endpoint, "http://") {
			useSSL = false
			endpoint = strings.TrimPrefix(endpoint, "http://")
		} else if strings.HasPrefix(endpoint, "https://") {
			endpoint = strings.TrimPrefix(endpoint, "https://")
		}
		// Remove any trailing slashes
		endpoint = strings.TrimSuffix(endpoint, "/")
	} else {
		// Default to AWS S3 endpoint
		endpoint = fmt.Sprintf("s3.%s.amazonaws.com", cfg.AWSRegion)
	}

	// Create MinIO client
	s3Client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AWSAccessID, cfg.AWSSecret, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &ImageService{
		s3Client:    s3Client,
		bucketName:  cfg.BucketName,
		cdnURL:      cfg.CDNURL,
		enableWebP:  cfg.EnableWebP,
		webPQuality: cfg.WebPQuality,
		maxFileSize: cfg.MaxFileSize,
	}, nil
}

// UploadResult represents the result of an image upload
type UploadResult struct {
	URL        string    `json:"url"`
	FileName   string    `json:"file_name"`
	FileFormat string    `json:"file_format"`
	Size       int64     `json:"size"`
	Width      int       `json:"width"`
	Height     int       `json:"height"`
	UploadedAt time.Time `json:"uploaded_at"`
	PublicURL  string    `json:"public_url"`
	Etag       string    `json:"etag,omitempty"`
}

// UploadFromBase64 uploads an image from base64 encoded data
func (s *ImageService) UploadFromBase64(ctx context.Context, base64Data, fileName string) (*UploadResult, error) {
	imageData, fileFormat, err := s.extractAndValidateImage(base64Data)
	if err != nil {
		return nil, err
	}

	return s.processAndUpload(ctx, imageData, fileFormat, fileName)
}

// UploadFromMultipart uploads an image from multipart form data
func (s *ImageService) UploadFromMultipart(ctx context.Context, fileHeader *multipart.FileHeader) (*UploadResult, error) {
	if fileHeader == nil {
		return nil, errors.New("file header cannot be nil")
	}

	// Check file size
	if fileHeader.Size > s.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds limit %d", fileHeader.Size, s.maxFileSize)
	}

	// Open file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file content
	imageData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect file format
	fileFormat := http.DetectContentType(imageData)

	return s.processAndUpload(ctx, imageData, fileFormat, fileHeader.Filename)
}

// processAndUpload processes the image (convert if needed) and uploads to S3.
// The fileName parameter is the user's original name (kept for metadata only);
// a random UUID-based name is used for the S3 key.
func (s *ImageService) processAndUpload(ctx context.Context, imageData []byte, fileFormat, fileName string) (*UploadResult, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	processedData := imageData
	outputFormat := s.formatFromMime(fileFormat)

	// Generate random UUID-based filename (never use user-supplied names for storage)
	s3Key := uuid.New().String() + s.extensionFromFormat(outputFormat)
	_ = fileName // original name preserved by caller

	etag, s3url, err := s.uploadToS3(ctx, processedData, s3Key, s.contentTypeFromFormat(outputFormat))
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	publicURL := s.getPublicURL(s3Key)

	return &UploadResult{
		URL:        s3url,
		FileName:   s3Key,
		FileFormat: s.mimeFromFormat(outputFormat),
		Size:       int64(len(processedData)),
		Width:      width,
		Height:     height,
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
		Etag:       etag,
	}, nil
}

// uploadToS3 uploads to image data to S3
func (s *ImageService) uploadToS3(ctx context.Context, imageData []byte, fileName, contentType string) (string, string, error) {
	key := fmt.Sprintf("payments/%s", fileName)

	uploadInfo, err := s.s3Client.PutObject(ctx, s.bucketName, key, bytes.NewReader(imageData), int64(len(imageData)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload object: %w", err)
	}

	s3url := fmt.Sprintf("s3://%s/%s", s.bucketName, key)
	return uploadInfo.ETag, s3url, nil
}

// getPublicURL returns the public URL for uploaded image
func (s *ImageService) getPublicURL(fileName string) string {
	if s.cdnURL != "" {
		return fmt.Sprintf("%s/payments/%s", strings.TrimSuffix(s.cdnURL, "/"), fileName)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/payments/%s", s.bucketName, fileName)
}

// extractAndValidateImage extracts image data from base64 and validates it
func (s *ImageService) extractAndValidateImage(base64Data string) ([]byte, string, error) {
	if base64Data == "" {
		return nil, "", errors.New("base64 data cannot be empty")
	}

	var imageData []byte
	var fileFormat string

	if strings.HasPrefix(base64Data, "data:") {
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) != 2 {
			return nil, "", errors.New("invalid base64 format")
		}
		mimePart := strings.TrimPrefix(parts[0], "data:")
		mimePart = strings.TrimSuffix(mimePart, ";base64")
		fileFormat = mimePart

		var err error
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64: %w", err)
		}
	} else {
		var err error
		imageData, err = base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64: %w", err)
		}
		fileFormat = http.DetectContentType(imageData)
	}

	if !s.isValidImageFormat(fileFormat) {
		return nil, "", errors.New("invalid image format, only JPEG, PNG, WEBP, and GIF are supported")
	}

	if int64(len(imageData)) > s.maxFileSize {
		return nil, "", fmt.Errorf("image size %d exceeds limit %d", len(imageData), s.maxFileSize)
	}

	return imageData, fileFormat, nil
}

// isValidImageFormat checks if the mime type is a valid image format
func (s *ImageService) isValidImageFormat(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/pjpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

// formatFromMime converts MIME type to imageFormat
func (s *ImageService) formatFromMime(mimeType string) imageFormat {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/pjpeg":
		return formatJPEG
	case "image/png":
		return formatPNG
	case "image/webp":
		return formatWEBP
	case "image/gif":
		return formatGIF
	default:
		return formatJPEG
	}
}

// mimeFromFormat converts imageFormat to MIME type
func (s *ImageService) mimeFromFormat(format imageFormat) string {
	switch format {
	case formatJPEG:
		return "image/jpeg"
	case formatPNG:
		return "image/png"
	case formatWEBP:
		return "image/webp"
	case formatGIF:
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// contentTypeFromFormat returns the Content-Type header value for a format
func (s *ImageService) contentTypeFromFormat(format imageFormat) string {
	return s.mimeFromFormat(format)
}

// extensionFromFormat returns the file extension for a format
func (s *ImageService) extensionFromFormat(format imageFormat) string {
	switch format {
	case formatJPEG:
		return ".jpg"
	case formatPNG:
		return ".png"
	case formatWEBP:
		return ".webp"
	case formatGIF:
		return ".gif"
	default:
		return ".jpg"
	}
}

// ValidateBase64 validates base64 image data without uploading
func (s *ImageService) ValidateBase64(base64Data string) error {
	_, _, err := s.extractAndValidateImage(base64Data)
	return err
}

// GetImageFromBase64 extracts image bytes from base64 data
func (s *ImageService) GetImageFromBase64(base64Data string) ([]byte, string, error) {
	return s.extractAndValidateImage(base64Data)
}

// Delete deletes an image from S3 by filename
func (s *ImageService) Delete(ctx context.Context, fileName string) error {
	key := fmt.Sprintf("payments/%s", fileName)

	err := s.s3Client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// GetPresignedURL generates a presigned URL for direct upload (useful for client-side uploads)
func (s *ImageService) GetPresignedURL(ctx context.Context, fileName, contentType string, expiresIn time.Duration) (string, error) {
	key := fmt.Sprintf("payments/%s", fileName)

	presignedURL, err := s.s3Client.PresignedPutObject(ctx, s.bucketName, key, expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}

	return presignedURL.String(), nil
}

// UploadFromBase64WithPrefix uploads an image from base64 data using a
// configurable S3 key prefix (e.g. "bill-images/"). The default
// UploadFromBase64 hardcodes "payments/"; this variant avoids breaking
// existing callers.
func (s *ImageService) UploadFromBase64WithPrefix(ctx context.Context, base64Data, fileName, prefix string) (*UploadResult, error) {
	imageData, fileFormat, err := s.extractAndValidateImage(base64Data)
	if err != nil {
		return nil, err
	}

	return s.processAndUploadWithPrefix(ctx, imageData, fileFormat, fileName, prefix)
}

// processAndUploadWithPrefix is the same as processAndUpload but uses a
// configurable S3 key prefix instead of the hardcoded "payments/".
// The fileName parameter is the user's original name (kept for metadata only);
// a random UUID-based name is used for the S3 key.
func (s *ImageService) processAndUploadWithPrefix(ctx context.Context, imageData []byte, fileFormat, fileName, prefix string) (*UploadResult, error) {
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	processedData := imageData
	outputFormat := s.formatFromMime(fileFormat)

	// Generate random UUID-based filename (never use user-supplied names for storage)
	s3Key := uuid.New().String() + s.extensionFromFormat(outputFormat)
	_ = fileName // original name preserved by caller

	etag, s3url, err := s.uploadToS3WithPrefix(ctx, processedData, s3Key, s.contentTypeFromFormat(outputFormat), prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	publicURL := s.getPublicURLWithPrefix(s3Key, prefix)

	return &UploadResult{
		URL:        s3url,
		FileName:   s3Key,
		FileFormat: s.mimeFromFormat(outputFormat),
		Size:       int64(len(processedData)),
		Width:      width,
		Height:     height,
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
		Etag:       etag,
	}, nil
}

// uploadToS3WithPrefix uploads image data to S3 with a configurable key prefix.
func (s *ImageService) uploadToS3WithPrefix(ctx context.Context, imageData []byte, fileName, contentType, prefix string) (string, string, error) {
	if prefix == "" {
		prefix = "payments/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	key := fmt.Sprintf("%s%s", prefix, fileName)

	uploadInfo, err := s.s3Client.PutObject(ctx, s.bucketName, key, bytes.NewReader(imageData), int64(len(imageData)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload object: %w", err)
	}

	s3url := fmt.Sprintf("s3://%s/%s", s.bucketName, key)
	return uploadInfo.ETag, s3url, nil
}

// getPublicURLWithPrefix returns the public URL using a configurable prefix.
func (s *ImageService) getPublicURLWithPrefix(fileName, prefix string) string {
	if prefix == "" {
		prefix = "payments/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if s.cdnURL != "" {
		return fmt.Sprintf("%s/%s%s", strings.TrimSuffix(s.cdnURL, "/"), prefix, fileName)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s%s", s.bucketName, prefix, fileName)
}

// GetPresignedGetURL generates a presigned GET URL for reading an object.
// Used to give temporary read access to private S3 objects (e.g. bill images).
func (s *ImageService) GetPresignedGetURL(ctx context.Context, s3Key string, expiresIn time.Duration) (string, error) {
	reqParams := make(url.Values)
	presignedURL, err := s.s3Client.PresignedGetObject(ctx, s.bucketName, s3Key, expiresIn, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to presign GET URL: %w", err)
	}
	return presignedURL.String(), nil
}

// DeleteWithPrefix deletes an object from S3 using a configurable prefix.
func (s *ImageService) DeleteWithPrefix(ctx context.Context, fileName, prefix string) error {
	if prefix == "" {
		prefix = "payments/"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	key := fmt.Sprintf("%s%s", prefix, fileName)
	err := s.s3Client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}
