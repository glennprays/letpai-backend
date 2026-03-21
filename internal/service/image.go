package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/bimg"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ImageService handles image upload operations with MinIO/S3 and bimg
type ImageService struct {
	s3Client     *minio.Client
	bucketName   string
	cdnURL       string // Optional CDN URL for serving images
	enableWebP   bool
	webPQuality  int
	maxFileSize  int64
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

	// Open the file
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

// processAndUpload processes the image (convert if needed) and uploads to S3
func (s *ImageService) processAndUpload(ctx context.Context, imageData []byte, fileFormat, fileName string) (*UploadResult, error) {
	// Get image info using bimg.Size function
	imgInfo, err := bimg.Size(imageData)
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}

	processedData := imageData
	outputFormat := s.formatFromMime(fileFormat)

	// Convert to WebP if enabled and source is not already WebP
	if s.enableWebP && outputFormat != bimg.WEBP {
		processedData, err = s.convertToWebP(imageData)
		if err != nil {
			// Fall back to original if WebP conversion fails
			processedData = imageData
		} else {
			outputFormat = bimg.WEBP
		}
	}

	// Generate unique filename
	if fileName == "" {
		fileName = fmt.Sprintf("proof_%d%s", time.Now().Unix(), s.extensionFromFormat(outputFormat))
	} else {
		// Clean the filename and add extension
		fileName = strings.TrimSpace(fileName)
		fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
		fileName = fmt.Sprintf("%s_%d%s", sanitizeFilename(fileName), time.Now().Unix(), s.extensionFromFormat(outputFormat))
	}

	// Upload to S3
	etag, url, err := s.uploadToS3(ctx, processedData, fileName, s.contentTypeFromFormat(outputFormat))
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Get public URL
	publicURL := s.getPublicURL(fileName)

	return &UploadResult{
		URL:        url,
		FileName:   fileName,
		FileFormat: s.mimeFromFormat(outputFormat),
		Size:       int64(len(processedData)),
		Width:      imgInfo.Width,
		Height:     imgInfo.Height,
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
		Etag:       etag,
	}, nil
}

// convertToWebP converts the image to WebP format with compression
func (s *ImageService) convertToWebP(imageData []byte) ([]byte, error) {
	options := bimg.Options{
		Quality: s.webPQuality,
		Type:    bimg.WEBP,
	}

	return bimg.NewImage(imageData).Process(options)
}

// uploadToS3 uploads the image data to S3
func (s *ImageService) uploadToS3(ctx context.Context, imageData []byte, fileName, contentType string) (string, string, error) {
	key := fmt.Sprintf("payments/%s", fileName)

	// Upload using MinIO
	uploadInfo, err := s.s3Client.PutObject(ctx, s.bucketName, key, bytes.NewReader(imageData), int64(len(imageData)), minio.PutObjectOptions{
		ContentType: contentType,
		// Note: MinIO doesn't support ACL like AWS S3
		// Public access should be configured via bucket policy
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to upload object: %w", err)
	}

	url := fmt.Sprintf("s3://%s/%s", s.bucketName, key)
	etag := uploadInfo.ETag

	return etag, url, nil
}

// getPublicURL returns the public URL for the uploaded image
func (s *ImageService) getPublicURL(fileName string) string {
	if s.cdnURL != "" {
		return fmt.Sprintf("%s/payments/%s", strings.TrimSuffix(s.cdnURL, "/"), fileName)
	}
	// Default S3 public URL format
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
		// Extract mime type and data
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) != 2 {
			return nil, "", errors.New("invalid base64 format")
		}

		// Extract mime type
		mimePart := strings.TrimPrefix(parts[0], "data:")
		mimePart = strings.TrimSuffix(mimePart, ";base64")
		fileFormat = mimePart

		// Decode base64
		var err error
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64: %w", err)
		}
	} else {
		// Raw base64 data
		var err error
		imageData, err = base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, "", fmt.Errorf("failed to decode base64: %w", err)
		}
		fileFormat = http.DetectContentType(imageData)
	}

	// Validate image type
	if !s.isValidImageFormat(fileFormat) {
		return nil, "", errors.New("invalid image format, only JPEG, PNG, WEBP, and HEIC are supported")
	}

	// Validate image size
	if int64(len(imageData)) > s.maxFileSize {
		return nil, "", fmt.Errorf("image size %d exceeds limit %d", len(imageData), s.maxFileSize)
	}

	return imageData, fileFormat, nil
}

// isValidImageFormat checks if the mime type is a valid image format
func (s *ImageService) isValidImageFormat(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/heic", "image/heif":
		return true
	default:
		return false
	}
}

// formatFromMime converts MIME type to bimg format
func (s *ImageService) formatFromMime(mimeType string) bimg.ImageType {
	switch mimeType {
	case "image/jpeg", "image/jpg":
		return bimg.JPEG
	case "image/png":
		return bimg.PNG
	case "image/webp":
		return bimg.WEBP
	case "image/heic", "image/heif":
		return bimg.HEIF
	default:
		return bimg.JPEG
	}
}

// mimeFromFormat converts bimg format to MIME type
func (s *ImageService) mimeFromFormat(format bimg.ImageType) string {
	switch format {
	case bimg.JPEG:
		return "image/jpeg"
	case bimg.PNG:
		return "image/png"
	case bimg.WEBP:
		return "image/webp"
	case bimg.HEIF:
		return "image/heic"
	default:
		return "image/jpeg"
	}
}

// contentTypeFromFormat returns the Content-Type header value for a format
func (s *ImageService) contentTypeFromFormat(format bimg.ImageType) string {
	return s.mimeFromFormat(format)
}

// extensionFromFormat returns the file extension for a format
func (s *ImageService) extensionFromFormat(format bimg.ImageType) string {
	switch format {
	case bimg.JPEG:
		return ".jpg"
	case bimg.PNG:
		return ".png"
	case bimg.WEBP:
		return ".webp"
	case bimg.HEIF:
		return ".heic"
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

// sanitizeFilename removes or replaces unsafe characters from filename
func sanitizeFilename(fileName string) string {
	// Replace spaces with underscores
	fileName = strings.ReplaceAll(fileName, " ", "_")
	// Remove any characters that aren't alphanumeric, underscore, hyphen, or dot
	var result strings.Builder
	for _, r := range fileName {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GetPresignedURL generates a presigned URL for direct upload (useful for client-side uploads)
func (s *ImageService) GetPresignedURL(ctx context.Context, fileName, contentType string, expiresIn time.Duration) (string, error) {
	key := fmt.Sprintf("payments/%s", fileName)

	// Generate presigned PUT URL for client-side upload
	presignedURL, err := s.s3Client.PresignedPutObject(ctx, s.bucketName, key, expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to presign URL: %w", err)
	}

	return presignedURL.String(), nil
}
