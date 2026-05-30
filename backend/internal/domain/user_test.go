package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestUser_ValidateEmail(t *testing.T) {
	u := &User{}

	// Test valid email
	email := "test@example.com"
	u.Email = &email
	err := u.ValidateEmail()
	assert.NoError(t, err)

	// Email is optional, so empty email should be valid
	emptyEmail := ""
	u.Email = &emptyEmail
	err = u.ValidateEmail()
	assert.NoError(t, err)
}

func TestUser_ValidateUsername(t *testing.T) {
	u := &User{}

	// Test valid username
	u.Username = "testuser"
	err := u.ValidateUsername()
	assert.NoError(t, err)

	// Test empty username
	u.Username = ""
	err = u.ValidateUsername()
	assert.Error(t, err)

	// Test username too short
	u.Username = "ab"
	err = u.ValidateUsername()
	assert.Error(t, err)
}

func TestUser_ValidateFullname(t *testing.T) {
	u := &User{}

	// Test valid fullname
	u.Fullname = "John Doe"
	err := u.ValidateFullname()
	assert.NoError(t, err)

	// Test empty fullname
	u.Fullname = ""
	err = u.ValidateFullname()
	assert.Error(t, err)
}

func TestUser_ValidateRole(t *testing.T) {
	u := &User{}

	// Test valid roles
	u.Role = RoleAdmin
	err := u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RolePartner
	err = u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RoleEmployee
	err = u.ValidateRole()
	assert.NoError(t, err)

	u.Role = RoleAdvPartner
	err = u.ValidateRole()
	assert.NoError(t, err)

	// Test invalid role
	u.Role = "invalid"
	err = u.ValidateRole()
	assert.Error(t, err)
}

func TestUser_IsAdvPartner(t *testing.T) {
	u := &User{Role: RoleAdvPartner}
	assert.True(t, u.IsAdvPartner())

	u.Role = RoleAdmin
	assert.False(t, u.IsAdvPartner())
}

func TestUser_IsActive(t *testing.T) {
	u := &User{}

	// User is active by default (not soft deleted)
	assert.True(t, u.IsActive())

	// Soft delete the user
	now := time.Now()
	u.DeletedAt = gorm.DeletedAt{Time: now, Valid: true}
	assert.False(t, u.IsActive())
}

func TestUser_IsAdmin(t *testing.T) {
	u := &User{
		Role: RoleAdmin,
	}

	assert.True(t, u.IsAdmin())

	u.Role = RolePartner
	assert.False(t, u.IsAdmin())
}

func TestUser_IsPartner(t *testing.T) {
	u := &User{
		Role: RolePartner,
	}

	assert.True(t, u.IsPartner())

	u.Role = RoleAdmin
	assert.False(t, u.IsPartner())
}

func TestUser_IsEmployee(t *testing.T) {
	u := &User{
		Role: RoleEmployee,
	}

	assert.True(t, u.IsEmployee())

	u.Role = RoleAdmin
	assert.False(t, u.IsEmployee())
}

func TestUser_GetAuditEntityType(t *testing.T) {
	u := User{}
	entityType := u.GetAuditEntityType()
	assert.Equal(t, "user", entityType)
}

func TestUser_GetAuditEntityID(t *testing.T) {
	u := User{ID: 456}
	entityID := u.GetAuditEntityID()
	assert.Equal(t, uint(456), entityID)
}
