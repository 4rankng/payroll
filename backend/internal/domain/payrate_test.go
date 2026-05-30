package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPayrate_ValidatePayrateConfig(t *testing.T) {
	pr := &Payrate{
		ProjectID: 1,
		FromDate:  time.Now().AddDate(0, 0, -1),                 // yesterday
		ToDate:    &[]time.Time{time.Now().AddDate(0, 1, 0)}[0], // next month
		Payrate:   PayrateConfiguration(`{"regular": {"rate": 100000}}`),
	}

	err := pr.ValidatePayrateConfig()
	assert.NoError(t, err)

	// Test with empty configuration
	pr.Payrate = PayrateConfiguration("")
	err = pr.ValidatePayrateConfig()
	assert.Error(t, err)
}

func TestPayrate_IsValid(t *testing.T) {
	now := time.Now()
	toDate := now.AddDate(0, 1, 0) // next month
	pr := &Payrate{
		ProjectID: 1,
		FromDate:  now.AddDate(0, 0, -1), // yesterday
		ToDate:    &toDate,
		Payrate:   PayrateConfiguration(`{"regular": {"rate": 100000}}`),
	}

	err := pr.IsValid()
	assert.NoError(t, err)

	// Test with invalid ProjectID
	pr.ProjectID = 0
	err = pr.IsValid()
	assert.Error(t, err)
}

func TestPayrate_IsActiveOnDate(t *testing.T) {
	now := time.Now()
	toDate := now.AddDate(0, 1, 0) // next month
	pr := &Payrate{
		FromDate: now.AddDate(0, -1, 0), // 1 month ago
		ToDate:   &toDate,
	}

	// Should be active today
	assert.True(t, pr.IsActiveOnDate(now))

	// Should not be active in the future (after ValidTo)
	futureDate := now.AddDate(0, 2, 0) // 2 months from now
	assert.False(t, pr.IsActiveOnDate(futureDate))

	// Should not be active in the past (before ValidFrom)
	pastDate := now.AddDate(0, -2, 0) // 2 months ago
	assert.False(t, pr.IsActiveOnDate(pastDate))
}

func TestPayrate_GetRateValue(t *testing.T) {
	config := `{"regular": 100000, "overtime": 150000}`
	pr := &Payrate{
		Payrate: PayrateConfiguration(config),
	}

	// Test getting regular rate
	rate, err := pr.GetRateValue("regular")
	assert.NoError(t, err)
	assert.Equal(t, 100000, rate)

	// Test getting overtime rate
	rate, err = pr.GetRateValue("overtime")
	assert.NoError(t, err)
	assert.Equal(t, 150000, rate)

	// Test getting non-existent rate
	rate, err = pr.GetRateValue("nonexistent")
	assert.Error(t, err)
	assert.Equal(t, 0, rate)
}

func TestPayrate_GetAllRates(t *testing.T) {
	config := `{"regular": 100000, "overtime": 150000}`
	pr := &Payrate{
		Payrate: PayrateConfiguration(config),
	}

	rates := pr.GetAllRates()
	assert.Equal(t, PayrateConfiguration(config), rates)
}

func TestPayrate_CanOverlapWith(t *testing.T) {
	now := time.Now()
	toDate1 := now.AddDate(0, 1, 0) // next month
	toDate2 := now.AddDate(0, 3, 0) // 3 months from now
	pr1 := &Payrate{
		FromDate:  now.AddDate(0, -1, 0), // 1 month ago
		ToDate:    &toDate1,
		ProjectID: 1,
	}

	pr2 := &Payrate{
		FromDate:  now.AddDate(0, 2, 0), // 2 months from now
		ToDate:    &toDate2,
		ProjectID: 1,
	}

	// Should not overlap since dates don't overlap
	assert.True(t, pr1.CanOverlapWith(pr2))

	// Make pr2's date range overlap with pr1
	newFromDate := now.AddDate(0, 0, 15) // middle of pr1's range
	pr2.FromDate = newFromDate
	assert.False(t, pr1.CanOverlapWith(pr2))

	// Different projects should be able to overlap
	pr2.ProjectID = 2
	assert.True(t, pr1.CanOverlapWith(pr2))
}

func TestPayrate_DateRangeOverlapsWith(t *testing.T) {
	now := time.Now()
	toDate1 := now.AddDate(0, 1, 0) // next month
	toDate2 := now.AddDate(0, 3, 0) // 3 months from now
	pr1 := &Payrate{
		FromDate: now.AddDate(0, -1, 0), // 1 month ago
		ToDate:   &toDate1,
	}

	pr2 := &Payrate{
		FromDate: now.AddDate(0, 2, 0), // 2 months from now
		ToDate:   &toDate2,
	}

	// Should not overlap
	assert.False(t, pr1.DateRangeOverlapsWith(pr2))

	// Make pr2's date range overlap with pr1
	newFromDate := now.AddDate(0, 0, 15) // middle of pr1's range
	pr2.FromDate = newFromDate
	assert.True(t, pr1.DateRangeOverlapsWith(pr2))
}
