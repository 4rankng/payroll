package helpers

import (
	"net/http/httptest"
	"testing"

	"api-server/internal/constants"

	"github.com/gin-gonic/gin"
)

func TestGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupCtx  func(*gin.Context)
		expectErr bool
		expected  uint
	}{
		{
			name: "valid user ID",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserID, uint(123))
			},
			expectErr: false,
			expected:  123,
		},
		{
			name:      "user ID not found",
			setupCtx:  func(c *gin.Context) {},
			expectErr: true,
			expected:  0,
		},
		{
			name: "invalid user ID type",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserID, "not-a-uint")
			},
			expectErr: true,
			expected:  0,
		},
		{
			name: "zero user ID",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserID, uint(0))
			},
			expectErr: true,
			expected:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			result, err := GetUserID(c)

			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestGetUserRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name      string
		setupCtx  func(*gin.Context)
		expectErr bool
		expected  string
	}{
		{
			name: "valid role",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserRole, "admin")
			},
			expectErr: false,
			expected:  "admin",
		},
		{
			name:      "role not found",
			setupCtx:  func(c *gin.Context) {},
			expectErr: true,
			expected:  "",
		},
		{
			name: "invalid role type",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserRole, 123)
			},
			expectErr: true,
			expected:  "",
		},
		{
			name: "empty role",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserRole, "")
			},
			expectErr: true,
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			result, err := GetUserRole(c)

			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetUserIDOrRespond(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupCtx       func(*gin.Context)
		expectedID     uint
		expectedOK     bool
		expectResponse bool
	}{
		{
			name: "valid user ID",
			setupCtx: func(c *gin.Context) {
				c.Set(constants.CtxUserID, uint(123))
			},
			expectedID:     123,
			expectedOK:     true,
			expectResponse: false,
		},
		{
			name:           "user ID not found",
			setupCtx:       func(c *gin.Context) {},
			expectedID:     0,
			expectedOK:     false,
			expectResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			id, ok := GetUserIDOrRespond(c)

			if id != tt.expectedID {
				t.Errorf("expected ID %d, got %d", tt.expectedID, id)
			}
			if ok != tt.expectedOK {
				t.Errorf("expected ok %v, got %v", tt.expectedOK, ok)
			}
			if tt.expectResponse && w.Code == 0 {
				t.Errorf("expected response to be sent but none was")
			}
		})
	}
}

func TestMustGetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid user ID", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(constants.CtxUserID, uint(123))

		result := MustGetUserID(c)
		if result != 123 {
			t.Errorf("expected 123, got %d", result)
		}
	})

	t.Run("panics when user ID not found", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic but didn't get one")
			}
		}()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		MustGetUserID(c)
	})
}
