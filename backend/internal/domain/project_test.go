package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProject_ValidateName(t *testing.T) {
	p := &Project{}

	// Test valid name
	p.Name = "Test Project"
	err := p.ValidateName()
	assert.NoError(t, err)

	// Test empty name
	p.Name = ""
	err = p.ValidateName()
	assert.Error(t, err)
}

func TestProject_ValidateDates(t *testing.T) {
	startDate := time.Now().AddDate(0, 0, 1) // tomorrow
	endDate := time.Now().AddDate(0, 1, 1)   // next month
	p := &Project{
		StartDate: &startDate,
		EndDate:   &endDate,
	}

	err := p.ValidateDates()
	assert.NoError(t, err)

	// Test start date after end date
	startDate = time.Now().AddDate(0, 1, 1) // next month
	endDate = time.Now().AddDate(0, 0, 1)   // tomorrow
	p.StartDate = &startDate
	p.EndDate = &endDate
	err = p.ValidateDates()
	assert.Error(t, err)
}

func TestProject_ValidateStatus(t *testing.T) {
	p := &Project{}

	// Test valid status
	p.ProjectStatus = ProjectStatusDraft
	err := p.ValidateStatus()
	assert.NoError(t, err)

	p.ProjectStatus = ProjectStatusRunning
	err = p.ValidateStatus()
	assert.NoError(t, err)

	p.ProjectStatus = ProjectStatusCompleted
	err = p.ValidateStatus()
	assert.NoError(t, err)

	p.ProjectStatus = ProjectStatusCancelled
	err = p.ValidateStatus()
	assert.NoError(t, err)

	// Test invalid status
	p.ProjectStatus = "invalid"
	err = p.ValidateStatus()
	assert.Error(t, err)
}

func TestProject_IsRunning(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusRunning,
	}

	assert.True(t, p.IsRunning())

	p.ProjectStatus = ProjectStatusDraft
	assert.False(t, p.IsRunning())
}

func TestProject_IsDraft(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusDraft,
	}

	assert.True(t, p.IsDraft())

	p.ProjectStatus = ProjectStatusRunning
	assert.False(t, p.IsDraft())
}

func TestProject_IsCompleted(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusCompleted,
	}

	assert.True(t, p.IsCompleted())

	p.ProjectStatus = ProjectStatusRunning
	assert.False(t, p.IsCompleted())
}

func TestProject_IsCancelled(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusCancelled,
	}

	assert.True(t, p.IsCancelled())

	p.ProjectStatus = ProjectStatusRunning
	assert.False(t, p.IsCancelled())
}

func TestProject_IsPaused(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusPaused,
	}

	assert.True(t, p.IsPaused())

	p.ProjectStatus = ProjectStatusRunning
	assert.False(t, p.IsPaused())
}

func TestProject_CanTransitionTo(t *testing.T) {
	p := &Project{
		ProjectStatus: ProjectStatusDraft,
	}

	// From draft, can transition to running
	assert.True(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.False(t, p.CanTransitionTo(ProjectStatusCompleted))

	// From running, can transition to various states
	p.ProjectStatus = ProjectStatusRunning
	assert.True(t, p.CanTransitionTo(ProjectStatusPaused))
	assert.True(t, p.CanTransitionTo(ProjectStatusCompleted))
	assert.True(t, p.CanTransitionTo(ProjectStatusCancelled))
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))

	// From completed, no transitions allowed
	p.ProjectStatus = ProjectStatusCompleted
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))
	assert.False(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.False(t, p.CanTransitionTo(ProjectStatusPaused))
	assert.False(t, p.CanTransitionTo(ProjectStatusCancelled))

	// From cancelled, no transitions allowed
	p.ProjectStatus = ProjectStatusCancelled
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))
	assert.False(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.False(t, p.CanTransitionTo(ProjectStatusCompleted))
	assert.False(t, p.CanTransitionTo(ProjectStatusPaused))

	// From paused, can transition back to running or complete/cancel
	p.ProjectStatus = ProjectStatusPaused
	assert.True(t, p.CanTransitionTo(ProjectStatusRunning))
	assert.True(t, p.CanTransitionTo(ProjectStatusCompleted))
	assert.True(t, p.CanTransitionTo(ProjectStatusCancelled))
	assert.False(t, p.CanTransitionTo(ProjectStatusDraft))
}

func TestProject_GetTotalExpenses(t *testing.T) {
	p := &Project{
		TotalPayoutVND:    50000.00,
		PendingPayableVND: 30000.00,
	}

	expectedExpenses := 80000.00 // 50000 + 30000
	actualExpenses := p.GetTotalExpenses()
	assert.Equal(t, expectedExpenses, actualExpenses)
}

func TestProject_GetProfitMargin(t *testing.T) {
	p := &Project{
		TotalPayoutVND:       50000.00,
		PendingPayableVND:    30000.00,
		TotalReceivedVND:     100000.00,
		PendingReceivableVND: 20000.00,
	}

	// Total revenue = 100000 + 20000 = 120000
	// Total expenses = 50000 + 30000 = 80000
	// Profit = 120000 - 80000 = 40000
	// Margin = (40000 / 120000) * 100 = 33.33%
	expectedMargin := 33.33333333333333
	actualMargin := p.GetProfitMargin()
	assert.Equal(t, expectedMargin, actualMargin)

	// Test with zero revenue (to avoid division by zero)
	p.TotalReceivedVND = 0.0
	p.PendingReceivableVND = 0.0
	actualMargin = p.GetProfitMargin()
	assert.Equal(t, 0.0, actualMargin)
}

func TestProject_IsValid(t *testing.T) {
	startDate := time.Now()
	endDate := startDate.AddDate(0, 1, 0)

	// Test valid project
	p := &Project{
		Name:          "Test Project",
		ProjectStatus: ProjectStatusRunning,
		StartDate:     &startDate,
		EndDate:       &endDate,
	}
	err := p.IsValid()
	assert.NoError(t, err)

	// Test invalid name
	p2 := &Project{
		Name: "",
	}
	err = p2.IsValid()
	assert.Error(t, err)
}

func TestProject_GetAuditEntityType(t *testing.T) {
	p := Project{}
	entityType := p.GetAuditEntityType()
	assert.Equal(t, "project", entityType)
}

func TestProject_GetAuditEntityID(t *testing.T) {
	p := Project{ID: 789}
	entityID := p.GetAuditEntityID()
	assert.Equal(t, uint(789), entityID)
}

// --- Shift names (admin display names for payrate shifts) ---

func TestIsValidShiftRange(t *testing.T) {
	cases := map[string]bool{
		"09:00-18:00": true,
		"21:00-05:00": true,
		"00:00-23:59": true,
		// invalid
		"":            false,
		"9:00-18:00":  false, // hour must be 2 digits
		"25:00-18:00": false,
		"09:60-18:00": false,
		"0900-1800":   false,
		"09:00/18:00": false,
		"09:00":       false,
	}
	for input, want := range cases {
		got := IsValidShiftRange(input)
		assert.Equalf(t, want, got, "IsValidShiftRange(%q) = %v, want %v", input, got, want)
	}
}

func TestValidateShiftNames_NoopForNonFlexible(t *testing.T) {
	p := &Project{IsFlexible: false, ShiftNames: []ShiftName{{Range: "09:00-18:00", Name: "Ca làm"}}}
	assert.NoError(t, p.ValidateShiftNames(nil))
}

func TestValidateShiftNames_EmptyListOK(t *testing.T) {
	p := &Project{IsFlexible: true}
	assert.NoError(t, p.ValidateShiftNames(nil))
	assert.NoError(t, p.ValidateShiftNames([]string{"09:00-18:00"}))
}

func TestValidateShiftNames_MaxTwentyEnforced(t *testing.T) {
	names := make([]ShiftName, 21)
	for i := range names {
		names[i] = ShiftName{Range: "09:00-18:00", Name: "Ca"}
	}
	p := &Project{IsFlexible: true, ShiftNames: names}
	err := p.ValidateShiftNames(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "20")
}

func TestValidateShiftNames_RequiredFields(t *testing.T) {
	// missing range
	p := &Project{IsFlexible: true, ShiftNames: []ShiftName{{Name: "Ca"}}}
	err := p.ValidateShiftNames(nil)
	assert.Error(t, err)

	// invalid range format
	p.ShiftNames = []ShiftName{{Range: "9-5", Name: "Ca"}}
	err = p.ValidateShiftNames(nil)
	assert.Error(t, err)

	// missing name
	p.ShiftNames = []ShiftName{{Range: "09:00-18:00"}}
	err = p.ValidateShiftNames(nil)
	assert.Error(t, err)
}

func TestValidateShiftNames_NameLength(t *testing.T) {
	long := strings.Repeat("a", 51)
	p := &Project{IsFlexible: true, ShiftNames: []ShiftName{{Range: "09:00-18:00", Name: long}}}
	err := p.ValidateShiftNames(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "50")
}

func TestValidateShiftNames_DuplicateRangeRejected(t *testing.T) {
	p := &Project{IsFlexible: true, ShiftNames: []ShiftName{
		{Range: "09:00-18:00", Name: "Ca làm"},
		{Range: "09:00-18:00", Name: "Ca ngày"},
	}}
	err := p.ValidateShiftNames(nil)
	assert.Error(t, err)
}

func TestValidateShiftNames_RangeMustMatchPayrate(t *testing.T) {
	payrateRanges := []string{"09:00-18:00", "21:00-05:00"}

	// mismatch -> error
	p := &Project{IsFlexible: true, ShiftNames: []ShiftName{{Range: "08:00-17:00", Name: "Hành chính"}}}
	err := p.ValidateShiftNames(payrateRanges)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "không khớp")

	// all match -> OK
	p.ShiftNames = []ShiftName{
		{Range: "09:00-18:00", Name: "Ca làm"},
		{Range: "21:00-05:00", Name: "Ca đêm"},
	}
	assert.NoError(t, p.ValidateShiftNames(payrateRanges))
}

func TestValidateShiftNames_OvernightRangeAllowed(t *testing.T) {
	// 21:00-05:00 crosses midnight; the regex allows it (end <= start is valid).
	p := &Project{IsFlexible: true, ShiftNames: []ShiftName{{Range: "21:00-05:00", Name: "Ca đêm"}}}
	assert.NoError(t, p.ValidateShiftNames(nil))
}
