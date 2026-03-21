package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// ImageService handles image upload operations
// For MVP v1, this is a stub implementation that simulates S3/Cloudinary upload
type ImageService struct {
	baseURL    string
	uploadPath string
}

// NewImageService creates a new image service
func NewImageService(baseURL, uploadPath string) *ImageService {
	if uploadPath == "" {
		uploadPath = "/uploads"
	}
	return &ImageService{
		baseURL:    baseURL,
		uploadPath: uploadPath,
	}
}

// UploadResult represents the result of an image upload
type UploadResult struct {
	URL         string    `json:"url"`
	FileName    string    `json:"file_name"`
	FileFormat  string    `json:"file_format"`
	Size        int64     `json:"size"`
	UploadedAt  time.Time `json:"uploaded_at"`
	PublicURL   string    `json:"public_url"`
}

// UploadFromBase64 uploads an image from base64 encoded data
func (s *ImageService) UploadFromBase64(ctx context.Context, base64Data, fileName string) (*UploadResult, error) {
	if base64Data == "" {
		return nil, errors.New("base64 data cannot be empty")
	}

	// Parse base64 data
	// Expected format: "data:image/jpeg;base64,/9j/4AAQSkZJRg..."
	var imageData []byte
	var fileFormat string

	if strings.HasPrefix(base64Data, "data:") {
		// Extract mime type and data
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) != 2 {
			return nil, errors.New("invalid base64 format")
		}

		// Extract mime type
		mimePart := strings.TrimPrefix(parts[0], "data:")
		mimePart = strings.TrimSuffix(mimePart, ";base64")
		fileFormat = mimePart

		// Decode base64
		var err error
		imageData, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
	} else {
		// Raw base64 data
		var err error
		imageData, err = base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64: %w", err)
		}
		fileFormat = http.DetectContentType(imageData)
	}

	// Validate image type
	if !s.isValidImageFormat(fileFormat) {
		return nil, errors.New("invalid image format, only JPEG, PNG, and WEBP are supported")
	}

	// Validate image size (max 5MB)
	if int64(len(imageData)) > 5*1024*1024 {
		return nil, errors.New("image size exceeds 5MB limit")
	}

	// Generate unique filename
	ext := s.extensionFromMime(fileFormat)
	if fileName == "" {
		fileName = fmt.Sprintf("proof_%d%s", time.Now().Unix(), ext)
	} else {
		// Clean the filename and add extension
		fileName = strings.TrimSpace(fileName)
		fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
		fileName = fmt.Sprintf("%s_%d%s", fileName, time.Now().Unix(), ext)
	}

	// For MVP v1, return a simulated URL
	// In production, this would upload to S3/Cloudinary
	publicURL := fmt.Sprintf("%s%s/%s", s.baseURL, s.uploadPath, fileName)

	return &UploadResult{
		URL:        publicURL,
		FileName:   fileName,
		FileFormat: fileFormat,
		Size:       int64(len(imageData)),
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
	}, nil
}

// UploadFromMultipart uploads an image from multipart form data
func (s *ImageService) UploadFromMultipart(ctx context.Context, fileHeader *multipart.FileHeader) (*UploadResult, error) {
	if fileHeader == nil {
		return nil, errors.New("file header cannot be nil")
	}

	// Open the file
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file content
	imageData := make([]byte, fileHeader.Size)
	_, err = io.ReadFull(file, imageData)
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Validate image type
	fileFormat := http.DetectContentType(imageData)
	if !s.isValidImageFormat(fileFormat) {
		return nil, errors.New("invalid image format, only JPEG, PNG, and WEBP are supported")
	}

	// Validate image size (max 5MB)
	if fileHeader.Size > 5*1024*1024 {
		return nil, errors.New("image size exceeds 5MB limit")
	}

	// Generate unique filename
	ext := path.Ext(fileHeader.Filename)
	if ext == "" {
		ext = s.extensionFromMime(fileFormat)
	}
	fileName := fmt.Sprintf("proof_%d%s", time.Now().Unix(), ext)

	// For MVP v1, return a simulated URL
	publicURL := fmt.Sprintf("%s%s/%s", s.baseURL, s.uploadPath, fileName)

	return &UploadResult{
		URL:        publicURL,
		FileName:   fileName,
		FileFormat: fileFormat,
		Size:       fileHeader.Size,
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
	}, nil
}

// UploadFromURL downloads and uploads an image from a URL
func (s *ImageService) UploadFromURL(ctx context.Context, url string) (*UploadResult, error) {
	if url == "" {
		return nil, errors.New("URL cannot be empty")
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Read image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Validate image type
	fileFormat := http.DetectContentType(imageData)
	if !s.isValidImageFormat(fileFormat) {
		return nil, errors.New("invalid image format, only JPEG, PNG, and WEBP are supported")
	}

	// Validate image size (max 5MB)
	if int64(len(imageData)) > 5*1024*1024 {
		return nil, errors.New("image size exceeds 5MB limit")
	}

	// Generate unique filename
	ext := s.extensionFromMime(fileFormat)
	fileName := fmt.Sprintf("proof_%d%s", time.Now().Unix(), ext)

	// For MVP v1, return a simulated URL
	publicURL := fmt.Sprintf("%s%s/%s", s.baseURL, s.uploadPath, fileName)

	return &UploadResult{
		URL:        publicURL,
		FileName:   fileName,
		FileFormat: fileFormat,
		Size:       int64(len(imageData)),
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
	}, nil
}

// isValidImageFormat checks if the mime type is a valid image format
func (s *ImageService) isValidImageFormat(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

// extensionFromMime returns the file extension for a given mime type
func (s *ImageService) extensionFromMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".jpg"
	}
}

// ValidateBase64 validates base64 image data without uploading
func (s *ImageService) ValidateBase64(base64Data string) error {
	if base64Data == "" {
		return errors.New("base64 data cannot be empty")
	}

	var imageData []byte

	if strings.HasPrefix(base64Data, "data:") {
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) != 2 {
			return errors.New("invalid base64 format")
		}
		_, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("invalid base64: %w", err)
		}
		imageData, _ = base64.StdEncoding.DecodeString(parts[1])
	} else {
		_, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			return fmt.Errorf("invalid base64: %w", err)
		}
		imageData, _ = base64.StdEncoding.DecodeString(base64Data)
	}

	// Validate size
	if int64(len(imageData)) > 5*1024*1024 {
		return errors.New("image size exceeds 5MB limit")
	}

	return nil
}

// GetImageFromBase64 extracts image bytes from base64 data
func (s *ImageService) GetImageFromBase64(base64Data string) ([]byte, string, error) {
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

	return imageData, fileFormat, nil
}

// StoreBytes stores image bytes and returns the URL
// This is a placeholder for actual storage implementation
func (s *ImageService) StoreBytes(ctx context.Context, imageData []byte, fileName, fileFormat string) (*UploadResult, error) {
	publicURL := fmt.Sprintf("%s%s/%s", s.baseURL, s.uploadPath, fileName)

	return &UploadResult{
		URL:        publicURL,
		FileName:   fileName,
		FileFormat: fileFormat,
		Size:       int64(len(imageData)),
		UploadedAt: time.Now(),
		PublicURL:  publicURL,
	}, nil
}

// Delete deletes an image by URL
func (s *ImageService) Delete(ctx context.Context, url string) error {
	// TODO: Implement actual deletion from S3/Cloudinary
	// For MVP v1, this is a no-op
	return nil
}
