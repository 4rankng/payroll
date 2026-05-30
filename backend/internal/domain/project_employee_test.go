package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProjectEmployee_ValidateProjectID(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid project ID
	pe.ProjectID = 1
	err := pe.ValidateProjectID()
	assert.NoError(t, err)

	// Test invalid project ID
	pe.ProjectID = 0
	err = pe.ValidateProjectID()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidateEmployeeID(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid employee ID
	pe.EmployeeID = 1
	err := pe.ValidateEmployeeID()
	assert.NoError(t, err)

	// Test invalid employee ID
	pe.EmployeeID = 0
	err = pe.ValidateEmployeeID()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidateDates(t *testing.T) {
	now := time.Now()
	var endDate *time.Time
	endTime := now.AddDate(0, 1, 1) // next month
	endDate = &endTime

	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, 0, 1), // tomorrow
		LastDate:  endDate,              // next month
	}

	err := pe.ValidateDates()
	assert.NoError(t, err)

	// Test start date after end date
	startDate := now.AddDate(0, 1, 2) // after the end date
	pe.StartDate = startDate
	err = pe.ValidateDates()
	assert.Error(t, err)
}

func TestProjectEmployee_ValidatePosition(t *testing.T) {
	pe := &ProjectEmployee{}

	// Test valid position
	pe.Position = "Developer"
	err := pe.ValidatePosition()
	assert.NoError(t, err)

	// Test empty position
	pe.Position = ""
	err = pe.ValidatePosition()
	assert.Error(t, err)
}

func TestProjectEmployee_IsValid(t *testing.T) {
	now := time.Now()
	var endDate *time.Time
	endTime := now.AddDate(0, 1, 1) // next month
	endDate = &endTime

	pe := &ProjectEmployee{
		ProjectID:       1,
		EmployeeID:      1,
		Position:        "Developer",
		StartDate:       now.AddDate(0, 0, 1), // tomorrow
		LastDate:        endDate,              // next month
		PaymentSchedule: "weekly",             // Default payment schedule
	}

	// Test valid project employee
	err := pe.IsValid()
	assert.NoError(t, err)

	// Test invalid project employee
	pe.ProjectID = 0
	err = pe.IsValid()
	assert.Error(t, err)
}

func TestProjectEmployee_IsCurrentlyAssigned(t *testing.T) {
	now := time.Now()

	// Test assignment that is active now
	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, 0, -5), // 5 days ago
		LastDate:  nil,                   // Currently active
	}

	assert.True(t, pe.IsCurrentlyAssigned())

	// Test assignment that ended in the past
	endDate := now.AddDate(0, 0, -1) // yesterday
	pe.LastDate = &endDate
	assert.False(t, pe.IsCurrentlyAssigned())
}

func TestProjectEmployee_HasEnded(t *testing.T) {
	now := time.Now()

	// Test assignment that has ended
	endDate := now.AddDate(0, -1, 0) // 1 month ago
	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, -2, 0), // 2 months ago
		LastDate:  &endDate,
	}

	assert.True(t, pe.HasEnded())

	// Test assignment that is still active
	pe.LastDate = nil
	assert.False(t, pe.HasEnded())
}

func TestProjectEmployee_GetAssignmentDuration(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0) // 1 month ago
	endDate := now.AddDate(0, 1, 0)    // next month

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  &endDate,
	}

	duration := pe.GetAssignmentDuration()
	expectedDuration := int(endDate.Sub(startDate).Hours() / 24) // Convert duration to days
	assert.Equal(t, expectedDuration, duration)
}

func TestProjectEmployee_CanEndAssignment(t *testing.T) {
	now := time.Now()

	// Test assignment that can be ended (it's currently active)
	pe := &ProjectEmployee{
		StartDate: now.AddDate(0, 0, -5), // 5 days ago
		LastDate:  nil,                   // Currently active
	}

	assert.True(t, pe.CanEndAssignment())

	// Test assignment that has already ended
	endDate := now.AddDate(0, 0, -1) // yesterday
	pe.LastDate = &endDate
	assert.False(t, pe.CanEndAssignment())
}

func TestProjectEmployee_EndAssignment(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  nil, // Currently active
	}

	// End the assignment
	newEndDate := now.AddDate(0, 0, -1) // yesterday
	err := pe.EndAssignment(newEndDate)
	assert.NoError(t, err)

	// Check if end date is set
	assert.True(t, pe.HasEnded())
}

func TestProjectEmployee_CanCreateTimesheet(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago
	endDate := now.AddDate(0, 1, 0)    // next month

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  &endDate,
	}

	// Should be able to create timesheet for today
	assert.True(t, pe.CanCreateTimesheet(now))

	// Should not be able to create timesheet before start date
	beforeStartDate := now.AddDate(0, 0, -10)
	assert.False(t, pe.CanCreateTimesheet(beforeStartDate))

	// Should not be able to create timesheet after end date
	afterEndDate := now.AddDate(0, 2, 0)
	assert.False(t, pe.CanCreateTimesheet(afterEndDate))

	// Test when LastDate is nil (currently active)
	pe.LastDate = nil
	assert.True(t, pe.CanCreateTimesheet(now))
}

func TestProjectEmployee_IsAssignedOnDate(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -5) // 5 days ago
	endDate := now.AddDate(0, 1, 0)    // next month

	pe := &ProjectEmployee{
		StartDate: startDate,
		LastDate:  &endDate,
	}

	// Should be assigned on today
	assert.True(t, pe.IsAssignedOnDate(now))

	// Should not be assigned before start date
	beforeStartDate := now.AddDate(0, 0, -10)
	assert.False(t, pe.IsAssignedOnDate(beforeStartDate))

	// Should not be assigned after end date
	afterEndDate := now.AddDate(0, 2, 0)
	assert.False(t, pe.IsAssignedOnDate(afterEndDate))

	// Test when LastDate is nil (currently active)
	pe.LastDate = nil
	assert.True(t, pe.IsAssignedOnDate(now))
}
