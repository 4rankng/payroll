package user

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserRepository is a mock implementation of UserRepository
type MockUserRepository struct {
	mock.Mock
}

// Mock implementation - implement minimal interface methods needed for testing
func (m *MockUserRepository) GetByID(ctx interface{}, id uint) (interface{}, error) {
	args := m.Called(ctx, id)
	return args.Get(0), args.Error(1)
}

// MockAuditLogRepository is a mock implementation of AuditLogRepository
type MockAuditLogRepository struct {
	mock.Mock
}

func TestValidateEmailFormat(t *testing.T) {
	service := &UserService{}

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "Valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with subdomain",
			email:   "user@mail.example.com",
			wantErr: false,
		},
		{
			name:    "Valid email with plus",
			email:   "test+tag@example.com",
			wantErr: false,
		},
		{
			name:    "Invalid email - no @",
			email:   "testexample.com",
			wantErr: true,
		},
		{
			name:    "Invalid email - no domain",
			email:   "test@",
			wantErr: true,
		},
		{
			name:    "Invalid email - no local part",
			email:   "@example.com",
			wantErr: true,
		},
		{
			name:    "Empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "Invalid email - spaces",
			email:   "test @example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateEmailFormat(tt.email)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseIDs(t *testing.T) {
	service := &UserService{
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	tests := []struct {
		name     string
		idsStr   string
		expected []uint
	}{
		{
			name:     "Valid IDs",
			idsStr:   "1,2,3",
			expected: []uint{1, 2, 3},
		},
		{
			name:     "Single ID",
			idsStr:   "42",
			expected: []uint{42},
		},
		{
			name:     "IDs with spaces",
			idsStr:   "1, 2, 3",
			expected: []uint{1, 2, 3},
		},
		{
			name:     "Empty string",
			idsStr:   "",
			expected: nil,
		},
		{
			name:     "IDs with invalid values skipped",
			idsStr:   "1,invalid,3",
			expected: []uint{1, 3},
		},
		{
			name:     "IDs with extra commas",
			idsStr:   "1,,3",
			expected: []uint{1, 3},
		},
		{
			name:     "Mixed valid and negative numbers",
			idsStr:   "1,-2,3",
			expected: []uint{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseIDs(tt.idsStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
