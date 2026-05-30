package storage

import (
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalFileStorage(t *testing.T) {
	basePath := "/tmp/test-storage"
	baseURL := "http://localhost:8080/uploads"

	storage := NewLocalFileStorage(basePath, baseURL)
	if storage == nil {
		t.Error("NewLocalFileStorage() returned nil")
	}

	localStorage, ok := storage.(*LocalFileStorage)
	if !ok {
		t.Error("NewLocalFileStorage() did not return *LocalFileStorage")
	}

	if localStorage.basePath != basePath {
		t.Errorf("basePath = %v, want %v", localStorage.basePath, basePath)
	}

	if localStorage.baseURL != baseURL {
		t.Errorf("baseURL = %v, want %v", localStorage.baseURL, baseURL)
	}
}

func TestGetFilePath(t *testing.T) {
	basePath := "/tmp/test-storage"
	storage := NewLocalFileStorage(basePath, "").(*LocalFileStorage)

	filename := "test/file.txt"
	expected := filepath.Join(basePath, filename)
	got := storage.GetFilePath(filename)

	if got != expected {
		t.Errorf("GetFilePath() = %v, want %v", got, expected)
	}
}

func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{name: "valid jpg", filename: "image.jpg", wantErr: false},
		{name: "valid jpeg", filename: "image.jpeg", wantErr: false},
		{name: "valid png", filename: "image.png", wantErr: false},
		{name: "valid gif", filename: "image.gif", wantErr: false},
		{name: "valid pdf", filename: "document.pdf", wantErr: false},
		{name: "valid doc", filename: "document.doc", wantErr: false},
		{name: "valid docx", filename: "document.docx", wantErr: false},
		{name: "valid xls", filename: "spreadsheet.xls", wantErr: false},
		{name: "valid xlsx", filename: "spreadsheet.xlsx", wantErr: false},
		{name: "valid txt", filename: "text.txt", wantErr: false},
		{name: "uppercase extension", filename: "image.JPG", wantErr: false},
		{name: "invalid exe", filename: "malware.exe", wantErr: true},
		{name: "invalid sh", filename: "script.sh", wantErr: true},
		{name: "no extension", filename: "file", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &multipart.FileHeader{
				Filename: tt.filename,
			}

			err := ValidateFileType(header)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		maxSize  int64
		wantErr  bool
	}{
		{name: "size within limit", fileSize: 1024, maxSize: 2048, wantErr: false},
		{name: "size at limit", fileSize: 2048, maxSize: 2048, wantErr: false},
		{name: "size exceeds limit", fileSize: 3000, maxSize: 2048, wantErr: true},
		{name: "zero size", fileSize: 0, maxSize: 2048, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &multipart.FileHeader{
				Size: tt.fileSize,
			}

			err := ValidateFileSize(header, tt.maxSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "normal filename",
			filename: "document.pdf",
			want:     "document.pdf",
		},
		{
			name:     "filename with forward slash",
			filename: "path/to/file.txt",
			want:     "path_to_file.txt",
		},
		{
			name:     "filename with backslash",
			filename: "path\\to\\file.txt",
			want:     "path_to_file.txt",
		},
		{
			name:     "filename with parent directory reference",
			filename: "../../../etc/passwd",
			want:     "______etc_passwd",
		},
		{
			name:     "filename with spaces",
			filename: "my document with spaces.pdf",
			want:     "my_document_with_spaces.pdf",
		},
		{
			name:     "complex filename",
			filename: "../path/with spaces/file..name.txt",
			want:     "__path_with_spaces_file_name.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeFilename(tt.filename)
			if got != tt.want {
				t.Errorf("SanitizeFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExists(t *testing.T) {
	// Create temporary directory for testing
	tempDir := os.TempDir()
	basePath := filepath.Join(tempDir, "test-storage-exists")
	defer func() { _ = os.RemoveAll(basePath) }()

	storage := NewLocalFileStorage(basePath, "").(*LocalFileStorage)

	// Create a test file
	testFilePath := "test/exists.txt"
	fullPath := filepath.Join(basePath, testFilePath)
	_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_ = f.Close()

	// Test existing file
	if !storage.Exists(testFilePath) {
		t.Error("Exists() returned false for existing file")
	}

	// Test non-existing file
	if storage.Exists("nonexistent/file.txt") {
		t.Error("Exists() returned true for non-existing file")
	}
}

func TestDelete(t *testing.T) {
	// Create temporary directory for testing
	tempDir := os.TempDir()
	basePath := filepath.Join(tempDir, "test-storage-delete")
	defer func() { _ = os.RemoveAll(basePath) }()

	storage := NewLocalFileStorage(basePath, "").(*LocalFileStorage)

	// Create a test file
	testFilePath := "test/delete-me.txt"
	fullPath := filepath.Join(basePath, testFilePath)
	_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	_ = f.Close()

	// Verify file exists
	if !storage.Exists(testFilePath) {
		t.Fatal("Test file was not created")
	}

	// Delete the file
	err = storage.Delete(testFilePath)
	if err != nil {
		t.Errorf("Delete() error = %v", err)
	}

	// Verify file is deleted
	if storage.Exists(testFilePath) {
		t.Error("File still exists after Delete()")
	}

	// Try to delete non-existing file (should not error)
	err = storage.Delete("nonexistent/file.txt")
	if err != nil {
		t.Errorf("Delete() error on non-existing file = %v", err)
	}
}

// Note: TestStore is more complex and would require a full multipart.File mock
// For now, we're focusing on the utility functions that provide good coverage
