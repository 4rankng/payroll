package storage

import (
	"api-server/internal/pkg/clock"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type FileStorage interface {
	Store(file multipart.File, header *multipart.FileHeader, uploadType string) (*StoredFile, error)
	StoreBytes(data []byte, filename string, uploadType string) (*StoredFile, error)
	Delete(filePath string) error
	GetFilePath(filename string) string
	Exists(filePath string) bool
}

type LocalFileStorage struct {
	basePath string
	baseURL  string
}

type StoredFile struct {
	Filename         string
	OriginalFilename string
	FilePath         string
	FileSize         int64
	ContentType      string
}

func NewLocalFileStorage(basePath, baseURL string) FileStorage {
	return &LocalFileStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// resolveSafePath joins basePath with rel and verifies the cleaned result stays
// within basePath, returning an error if the relative path escapes the storage
// root (e.g. via "../" segments). This is the path-traversal confinement
// boundary for all file operations; it complements the upload_type allowlist
// enforced in the asset service.
func (fs *LocalFileStorage) resolveSafePath(rel string) (string, error) {
	cleanBase := filepath.Clean(fs.basePath)
	full := filepath.Clean(filepath.Join(cleanBase, rel))
	sep := string(filepath.Separator)
	if full != cleanBase && !strings.HasPrefix(full, cleanBase+sep) {
		return "", fmt.Errorf("file path escapes storage root")
	}
	return full, nil
}

func (fs *LocalFileStorage) Store(file multipart.File, header *multipart.FileHeader, uploadType string) (*StoredFile, error) {
	// Generate unique filename
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Create directory structure based on upload type and date
	now := clock.Now()
	relativePath := filepath.Join(uploadType, fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	fullDir, err := fs.resolveSafePath(relativePath)
	if err != nil {
		return nil, fmt.Errorf("invalid upload path: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Full file path
	fullPath := filepath.Join(fullDir, filename)
	filePath := filepath.Join(relativePath, filename)

	// Create the file
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			fmt.Printf("Failed to close file: %v\n", closeErr)
		}
	}()

	// Copy file content
	if _, err := io.Copy(dst, file); err != nil {
		// Clean up on error
		if removeErr := os.Remove(fullPath); removeErr != nil {
			fmt.Printf("Failed to clean up file on copy error: %v\n", removeErr)
		}
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Get file size
	fileInfo, err := dst.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &StoredFile{
		Filename:         filename,
		OriginalFilename: header.Filename,
		FilePath:         filePath,
		FileSize:         fileInfo.Size(),
		ContentType:      header.Header.Get("Content-Type"),
	}, nil
}

func (fs *LocalFileStorage) StoreBytes(data []byte, filename string, uploadType string) (*StoredFile, error) {
	ext := filepath.Ext(filename)
	storedFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	now := clock.Now()
	relativePath := filepath.Join(uploadType, fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	fullDir, err := fs.resolveSafePath(relativePath)
	if err != nil {
		return nil, fmt.Errorf("invalid upload path: %w", err)
	}

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fullPath := filepath.Join(fullDir, storedFilename)
	filePath := filepath.Join(relativePath, storedFilename)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	return &StoredFile{
		Filename:         storedFilename,
		OriginalFilename: filename,
		FilePath:         filePath,
		FileSize:         int64(len(data)),
		ContentType:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	}, nil
}

func (fs *LocalFileStorage) Delete(filePath string) error {
	fullPath, err := fs.resolveSafePath(filePath)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (fs *LocalFileStorage) GetFilePath(filename string) string {
	full, err := fs.resolveSafePath(filename)
	if err != nil {
		// Escape attempt: return empty so callers cannot stat/serve a path
		// outside the storage root.
		return ""
	}
	return full
}

func (fs *LocalFileStorage) Exists(filePath string) bool {
	fullPath, err := fs.resolveSafePath(filePath)
	if err != nil {
		return false
	}
	_, err = os.Stat(fullPath)
	return !os.IsNotExist(err)
}

// ValidateFileType validates file type based on extension and content type
func ValidateFileType(header *multipart.FileHeader) error {
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".pdf":  true,
		".doc":  true,
		".docx": true,
		".xls":  true,
		".xlsx": true,
		".txt":  true,
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		return fmt.Errorf("file type %s not allowed", ext)
	}

	return nil
}

// ValidateFileSize validates file size
func ValidateFileSize(header *multipart.FileHeader, maxSize int64) error {
	if header.Size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", header.Size, maxSize)
	}
	return nil
}

// SanitizeFilename removes potentially dangerous characters from filename
func SanitizeFilename(filename string) string {
	// Remove path separators and other dangerous characters
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")
	filename = strings.ReplaceAll(filename, " ", "_")
	return filename
}
