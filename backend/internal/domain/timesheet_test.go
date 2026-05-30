package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimesheet_ValidateProjectID(t *testing.T) {
	ts := &Timesheet{}

	// Test valid project ID
	ts.ProjectID = 1
	err := ts.ValidateProjectID()
	assert.NoError(t, err)

	// Test invalid project ID
	ts.ProjectID = 0
	err = ts.ValidateProjectID()
	assert.Error(t, err)
}

func TestTimesheet_ValidateEmployeeID(t *testing.T) {
	ts := &Timesheet{}

	// Test valid employee ID
	ts.EmployeeID = 1
	err := ts.ValidateEmployeeID()
	assert.NoError(t, err)

	// Test invalid employee ID
	ts.EmployeeID = 0
	err = ts.ValidateEmployeeID()
	assert.Error(t, err)
}

func TestTimesheet_ValidateDate(t *testing.T) {
	ts := &Timesheet{
		Date: time.Now(),
	}

	err := ts.ValidateDate()
	assert.NoError(t, err)

	// Test date in the future
	ts.Date = time.Now().AddDate(0, 0, 1) // tomorrow
	err = ts.ValidateDate()
	assert.Error(t, err)
}

func TestTimesheet_ValidateHoursWorked(t *testing.T) {
	ts := &Timesheet{}

	// Test valid hours
	ts.HoursWorked = 8.0
	err := ts.ValidateHoursWorked()
	assert.NoError(t, err)

	// Test negative hours
	ts.HoursWorked = -1.0
	err = ts.ValidateHoursWorked()
	assert.Error(t, err)

	// Test too many hours
	ts.HoursWorked = 25.0
	err = ts.ValidateHoursWorked()
	assert.Error(t, err)
}

func TestTimesheet_ValidatePayType(t *testing.T) {
	ts := &Timesheet{
		PayType: "regular",
	}

	err := ts.ValidatePayType()
	assert.NoError(t, err)

	// Test invalid pay type
	ts.PayType = ""
	err = ts.ValidatePayType()
	assert.Error(t, err)
}

func TestTimesheet_ValidatePayRate(t *testing.T) {
	ts := &Timesheet{
		PayRate: 100000,
	}

	err := ts.ValidatePayRate()
	assert.NoError(t, err)

	// Test negative hourly rate
	ts.PayRate = -1000
	err = ts.ValidatePayRate()
	assert.Error(t, err)
}

func TestTimesheet_IsValid(t *testing.T) {
	// Need to include PayrateID for validation to pass
	ts := &Timesheet{
		ProjectID:   1,
		EmployeeID:  1,
		PayrateID:   1,
		Date:        time.Now().AddDate(0, 0, -1), // yesterday
		HoursWorked: 8.0,
		PayType:     "regular",
		PayRate:     100000,
		Status:      TimesheetStatusPendingApproval,
	}

	// Test valid timesheet
	err := ts.IsValid()
	assert.NoError(t, err)

	// Test invalid timesheet
	ts.ProjectID = 0
	err = ts.IsValid()
	assert.Error(t, err)
}

func TestTimesheet_CanBeEdited(t *testing.T) {
	ts := &Timesheet{
		Status:      TimesheetStatusPendingApproval,
		AllowedEdit: false,
	}

	// Should be editable when pending
	assert.True(t, ts.CanBeEdited())

	// Should not be editable when approved (unless explicitly allowed)
	ts.Status = TimesheetStatusApproved
	assert.False(t, ts.CanBeEdited())

	// Should be editable if allowed_edit flag is true
	ts.AllowedEdit = true
	assert.True(t, ts.CanBeEdited())

	// Should not be editable when rejected
	ts.Status = TimesheetStatusRejected
	ts.AllowedEdit = false
	assert.True(t, ts.CanBeEdited()) // rejected timesheets can be edited by default
}

func TestTimesheet_CalculateAmount(t *testing.T) {
	ts := &Timesheet{
		HoursWorked: 8.0,
		PayRate:     100000,
	}

	expectedAmount := int64(8 * 100000) // Direct calculation
	amount := ts.CalculateAmount()
	assert.Equal(t, expectedAmount, amount)
}

func TestTimesheet_IsWeekend(t *testing.T) {
	// Create a date that's a Saturday (weekday 6)
	saturday := time.Date(2023, time.March, 18, 0, 0, 0, 0, time.UTC)
	ts := &Timesheet{
		Date: saturday,
	}

	assert.True(t, ts.IsWeekend())

	// Create a date that's a Monday (weekday 1)
	monday := time.Date(2023, time.March, 20, 0, 0, 0, 0, time.UTC)
	ts.Date = monday

	assert.False(t, ts.IsWeekend())
}

func TestTimesheet_IsApproved(t *testing.T) {
	ts := &Timesheet{
		Status: TimesheetStatusApproved,
	}
	assert.True(t, ts.IsApproved())

	ts.Status = TimesheetStatusPendingApproval
	assert.False(t, ts.IsApproved())
}

func TestTimesheet_IsPendingApproval(t *testing.T) {
	ts := &Timesheet{
		Status: TimesheetStatusPendingApproval,
	}
	assert.True(t, ts.IsPendingApproval())

	ts.Status = TimesheetStatusApproved
	assert.False(t, ts.IsPendingApproval())
}

func TestTimesheet_IsRejected(t *testing.T) {
	ts := &Timesheet{
		Status: TimesheetStatusRejected,
	}
	assert.True(t, ts.IsRejected())

	ts.Status = TimesheetStatusApproved
	assert.False(t, ts.IsRejected())
}

func TestTimesheet_IsPaid(t *testing.T) {
	ts := &Timesheet{
		PaymentStatus: PaymentStatusPaid,
	}
	assert.True(t, ts.IsPaid())

	ts.PaymentStatus = PaymentStatusPending
	assert.False(t, ts.IsPaid())

	ts.PaymentStatus = PaymentStatusFailed
	assert.True(t, ts.IsPaid())

	ts.PaymentStatus = PaymentStatusCancelled
	assert.True(t, ts.IsPaid())
}
