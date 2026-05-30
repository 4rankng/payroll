package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/pkg/timeutil"
)

func (s *Service) GetEmployeeActivityStats(ctx context.Context, month string) (*dto.EmployeeActivityStatsResponse, error) {
	s.logger.Info("Getting employee activity stats", "month", month)

	var since, until time.Time
	if month != "" {
		t, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, fmt.Errorf("invalid month format: %w", err)
		}
		since = timeutil.StartOfMonth(t)
		until = timeutil.EndOfMonth(t).Add(time.Nanosecond) // exclusive upper bound
	} else {
		// Default: current month
		since = timeutil.StartOfMonth(clock.Now())
		until = timeutil.EndOfMonth(clock.Now()).Add(time.Nanosecond)
	}

	weekly, monthly, flexible, err := s.UserRepo.CountActiveEmployeesBySchedule(ctx, since, until)
	if err != nil {
		s.logger.Error("Failed to count active employees by schedule", "error", err)
		return nil, err
	}

	response := &dto.EmployeeActivityStatsResponse{
		ActiveWeekly:   weekly,
		ActiveMonthly:  monthly,
		ActiveFlexible: flexible,
		ActiveTotal:    weekly + monthly + flexible,
	}

	s.logger.Info("Employee activity stats retrieved",
		"active_weekly", weekly,
		"active_monthly", monthly,
		"active_flexible", flexible,
		"active_total", response.ActiveTotal)

	return response, nil
}

func (s *Service) GetActiveEmployeesBySchedule(ctx context.Context, month, schedule string) ([]*dto.ActiveEmployeeUserResponse, error) {
	s.logger.Info("Getting active employees by schedule", "schedule", schedule, "month", month)

	var since, until time.Time
	if month != "" {
		t, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, fmt.Errorf("invalid month format: %w", err)
		}
		since = timeutil.StartOfMonth(t)
		until = timeutil.EndOfMonth(t).Add(time.Nanosecond)
	} else {
		since = timeutil.StartOfMonth(clock.Now())
		until = timeutil.EndOfMonth(clock.Now()).Add(time.Nanosecond)
	}

	users, err := s.UserRepo.GetActiveEmployeesBySchedule(ctx, since, until, schedule)
	if err != nil {
		s.logger.Error("Failed to get active employees by schedule", "error", err)
		return nil, err
	}

	result := make([]*dto.ActiveEmployeeUserResponse, len(users))
	for i, u := range users {
		var lastLogin *string
		if u.LastLogin != nil {
			t := u.LastLogin.Format("2006-01-02T15:04:05Z07:00")
			lastLogin = &t
		}
		result[i] = &dto.ActiveEmployeeUserResponse{
			UserID:     u.UserID,
			EmployeeID: u.EmployeeID,
			Fullname:   u.Fullname,
			Username:   u.Username,
			LastLogin:  lastLogin,
		}
	}
	return result, nil
}
