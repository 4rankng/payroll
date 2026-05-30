package specs

import (
	"context"
	"time"
)

// WorkingDaysCounter abstracts counting of distinct working days
type WorkingDaysCounter interface {
	CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, startDate, endDate time.Time) (int, error)
}

// WorkingDaysSpec evaluates if an employee meets a minimum number of distinct working days
type WorkingDaysSpec struct {
	Counter   WorkingDaysCounter
	Threshold int
}

func NewWorkingDaysSpec(counter WorkingDaysCounter, threshold int) *WorkingDaysSpec {
	return &WorkingDaysSpec{Counter: counter, Threshold: threshold}
}

func (s *WorkingDaysSpec) IsEligible(ctx context.Context, employeeID, projectID uint, fromDate, toDate time.Time) (bool, error) {
	days, err := s.Counter.CountDistinctWorkingDays(ctx, employeeID, projectID, fromDate, toDate)
	if err != nil {
		return false, err
	}
	return days >= s.Threshold, nil
}
