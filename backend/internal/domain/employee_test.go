package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestEmployee_ValidateFullname(t *testing.T) {
	e := &Employee{}

	// Test valid fullname
	e.Fullname = "John Doe"
	err := e.ValidateFullname()
	assert.NoError(t, err)

	// Test empty fullname
	e.Fullname = ""
	err = e.ValidateFullname()
	assert.Error(t, err)
}

func TestEmployee_ValidateCCCD(t *testing.T) {
	e := &Employee{}

	// Test valid CCCD
	e.CCCD = "123456789"
	err := e.ValidateCCCD()
	assert.NoError(t, err)

	// Test empty CCCD
	e.CCCD = ""
	err = e.ValidateCCCD()
	assert.Error(t, err)

	// Test invalid characters in CCCD
	e.CCCD = "12345!6789"
	err = e.ValidateCCCD()
	assert.Error(t, err)
}

func TestEmployee_ValidateEmail(t *testing.T) {
	e := &Employee{}

	// Test valid email
	email := "test@example.com"
	e.Email = &email
	err := e.ValidateEmail()
	assert.NoError(t, err)

	// Test invalid email
	invalidEmail := "invalid-email"
	e.Email = &invalidEmail
	err = e.ValidateEmail()
	assert.Error(t, err)
}

func TestEmployee_ValidateMobile(t *testing.T) {
	e := &Employee{}

	// Test valid mobile
	e.Mobile = "0123456789"
	err := e.ValidateMobile()
	assert.NoError(t, err)

	// Test invalid mobile characters
	e.Mobile = "123abc456"
	err = e.ValidateMobile()
	assert.Error(t, err)
}

func TestEmployee_ValidateDates(t *testing.T) {
	e := &Employee{}

	// Test valid dates
	dateOfBirth := time.Now().AddDate(-30, 0, 0) // 30 years ago
	e.DateOfBirth = &dateOfBirth
	err := e.ValidateDates()
	assert.NoError(t, err)

	// Test future date of birth
	futureDate := time.Now().AddDate(1, 0, 0) // next year
	e.DateOfBirth = &futureDate
	err = e.ValidateDates()
	assert.Error(t, err)
}

func TestEmployee_HasBankingInfo(t *testing.T) {
	bankID := uint(1)
	e := &Employee{
		BankID:            &bankID,
		BankAccountName:   "Test Name",
		BankAccountNumber: "123456789",
	}

	hasInfo := e.HasBankingInfo()
	assert.True(t, hasInfo)

	e.BankAccountNumber = ""
	hasInfo = e.HasBankingInfo()
	assert.False(t, hasInfo)
}

func TestEmployee_GetAge(t *testing.T) {
	dateOfBirth := time.Now().AddDate(-30, 0, 0) // 30 years ago
	e := &Employee{
		DateOfBirth: &dateOfBirth,
	}

	age := e.GetAge()
	assert.Equal(t, 30, age)
}

func TestEmployee_CanBeAssignedToProject(t *testing.T) {
	bankID := uint(1)
	dateOfBirth := time.Now().AddDate(-25, 0, 0)
	e := &Employee{
		DateOfBirth:       &dateOfBirth,
		BankID:            &bankID,
		BankAccountName:   "Test Name",
		BankAccountNumber: "123456789",
	}

	canAssign := e.CanBeAssignedToProject()
	assert.True(t, canAssign)

	// Test without banking info
	e.BankID = nil
	canAssign = e.CanBeAssignedToProject()
	assert.False(t, canAssign)

	// Test soft-deleted employee should not be assignable even with banking info
	e.BankID = &bankID
	e.DeletedAt = gorm.DeletedAt{Time: time.Now(), Valid: true}
	canAssign = e.CanBeAssignedToProject()
	assert.False(t, canAssign)
}

func TestEmployee_ValidateBankingInfo(t *testing.T) {
	bankID := uint(1)
	e := &Employee{
		BankID:            &bankID,
		BankAccountName:   "Test Name",
		BankAccountNumber: "123456789",
	}

	err := e.ValidateBankingInfo()
	assert.NoError(t, err)

	// Test with empty bank account number
	e.BankAccountNumber = ""
	err = e.ValidateBankingInfo()
	assert.Error(t, err)
}

func TestEmployee_IsValid(t *testing.T) {
	bankID := uint(1)
	email := "test@example.com"
	dateOfBirth := time.Now().AddDate(-25, 0, 0)

	// Test valid employee
	e := &Employee{
		Fullname:          "John Doe",
		CCCD:              "123456789",
		Email:             &email,
		Mobile:            "0123456789",
		BankID:            &bankID,
		BankAccountName:   "Test Name",
		BankAccountNumber: "123456789",
		DateOfBirth:       &dateOfBirth,
	}
	err := e.IsValid()
	assert.NoError(t, err)

	// Test invalid fullname
	e2 := &Employee{
		Fullname: "",
		CCCD:     "123456789",
	}
	err = e2.IsValid()
	assert.Error(t, err)

	// Test invalid CCCD
	e3 := &Employee{
		Fullname: "John Doe",
		CCCD:     "",
	}
	err = e3.IsValid()
	assert.Error(t, err)
}

func TestEmployee_FormattedFullname(t *testing.T) {
	tests := []struct {
		name     string
		fullname string
		want     string
	}{
		{
			name:     "lowercase name",
			fullname: "john doe",
			want:     "John Doe",
		},
		{
			name:     "uppercase name",
			fullname: "JOHN DOE",
			want:     "John Doe",
		},
		{
			name:     "mixed case name",
			fullname: "JoHn DoE",
			want:     "John Doe",
		},
		{
			name:     "vietnamese name",
			fullname: "nguyễn văn a",
			want:     "Nguyễn Văn A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Employee{Fullname: tt.fullname}
			got := e.FormattedFullname()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEmployee_GetAuditEntityType(t *testing.T) {
	e := &Employee{}
	entityType := e.GetAuditEntityType()
	assert.Equal(t, "employee", entityType)
}

func TestEmployee_GetAuditEntityID(t *testing.T) {
	e := &Employee{ID: 123}
	entityID := e.GetAuditEntityID()
	assert.Equal(t, uint(123), entityID)
}
