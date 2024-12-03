package helpers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UploadConfig struct {
	MaxSizeBytes     int64
	AllowedMimeTypes []string
	UploadBasePath   string
}

var (
	DefaultImageUploadConfig = UploadConfig{
		MaxSizeBytes: 5 * 1024 * 1024, // 5MB
		AllowedMimeTypes: []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/webp",
		},
		UploadBasePath: "./uploads/",
	}

	DefaultDocumentUploadConfig = UploadConfig{
		MaxSizeBytes: 10 * 1024 * 1024, // 10MB
		AllowedMimeTypes: []string{
			"application/pdf",
			"application/msword",
			"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			"text/plain",
		},
		UploadBasePath: "./uploads/documents/",
	}
)

func UploadFile(c *gin.Context, fileHeader *multipart.FileHeader, uploadType string, configs ...UploadConfig) (string, error) {
	config := DefaultImageUploadConfig
	if len(configs) > 0 {
		config = configs[0]
	}

	if fileHeader.Size > config.MaxSizeBytes {
		return "", fmt.Errorf("file size exceeds maximum limit of %d MB", config.MaxSizeBytes/(1024*1024))
	}

	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil {
		return "", err
	}
	mimeType := http.DetectContentType(buffer)

	mimeTypeAllowed := false
	for _, allowedType := range config.AllowedMimeTypes {
		if mimeType == allowedType {
			mimeTypeAllowed = true
			break
		}
	}
	if !mimeTypeAllowed {
		return "", fmt.Errorf("invalid file type. Allowed types: %v", config.AllowedMimeTypes)
	}

	ext := filepath.Ext(fileHeader.Filename)

	uploadPath := filepath.Join(config.UploadBasePath, uploadType)
	if err := os.MkdirAll(uploadPath, os.ModePerm); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	fullFilepath := filepath.Join(uploadPath, filename)

	if err := c.SaveUploadedFile(fileHeader, fullFilepath); err != nil {
		return "", err
	}

	return fullFilepath, nil
}

func DeleteFile(filePath string) error {
	return os.Remove(filePath)
}
