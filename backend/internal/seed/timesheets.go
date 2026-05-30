package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	"math"
	mathrand "math/rand/v2"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"

	"gorm.io/gorm"
)

// isVietnameseHoliday checks if a date is a major Vietnamese public holiday
func isVietnameseHoliday(date time.Time) bool {
	holidays := GenerateVietnameseHolidays(date.Year())
	for _, holiday := range holidays {
		if holiday.YearDay() == date.YearDay() && holiday.Year() == date.Year() {
			return true
		}
	}
	return false
}

// EmployeeAttendanceProfile represents individual employee work patterns
type EmployeeAttendanceProfile struct {
	EmployeeID       uint
	Position         string
	AttendanceRate   float64
	IsReliable       bool
	OvertimeTendency float64 // 0.0-1.0, likelihood to work overtime
	WeekendWorkRate  float64 // 0.0-1.0, likelihood to work weekends
}

func (s *Seeder) seedTimesheets(ctx context.Context, db *gorm.DB) error {
	projectEmployeeRepo := persistence.NewProjectEmployeeRepository(&persistence.Database{DB: db})
	timesheetRepo := persistence.NewTimesheetRepository(&persistence.Database{DB: db}, projectEmployeeRepo)

	var adminUser domain.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	var projectEmployees []domain.ProjectEmployee
	if err := db.Find(&projectEmployees).Error; err != nil {
		return fmt.Errorf("find project employees: %w", err)
	}

	var payrates []domain.Payrate
	if err := db.Find(&payrates).Error; err != nil {
		return fmt.Errorf("find payrates: %w", err)
	}

	// Create employee attendance profiles based on positions
	employeeProfiles := make(map[uint]*EmployeeAttendanceProfile)
	for _, pe := range projectEmployees {
		if _, exists := employeeProfiles[pe.EmployeeID]; !exists {
			profile := &EmployeeAttendanceProfile{
				EmployeeID: pe.EmployeeID,
				Position:   pe.Position,
			}

			// Set attendance characteristics based on position
			switch pe.Position {
			case "thực tập":
				profile.AttendanceRate = 0.75 + mathrand.Float64()*0.15  // 75-90%
				profile.IsReliable = mathrand.Float64() < 0.6            // 60% reliable
				profile.OvertimeTendency = 0.1 + mathrand.Float64()*0.2  // 10-30%
				profile.WeekendWorkRate = 0.05 + mathrand.Float64()*0.15 // 5-20%
			case "phổ thông":
				profile.AttendanceRate = 0.80 + mathrand.Float64()*0.15   // 80-95%
				profile.IsReliable = mathrand.Float64() < 0.75            // 75% reliable
				profile.OvertimeTendency = 0.15 + mathrand.Float64()*0.25 // 15-40%
				profile.WeekendWorkRate = 0.10 + mathrand.Float64()*0.20  // 10-30%
			case "có kinh nghiệm":
				profile.AttendanceRate = 0.85 + mathrand.Float64()*0.12   // 85-97%
				profile.IsReliable = mathrand.Float64() < 0.85            // 85% reliable
				profile.OvertimeTendency = 0.20 + mathrand.Float64()*0.30 // 20-50%
				profile.WeekendWorkRate = 0.15 + mathrand.Float64()*0.25  // 15-40%
			case "chuyên môn":
				profile.AttendanceRate = 0.88 + mathrand.Float64()*0.10   // 88-98%
				profile.IsReliable = mathrand.Float64() < 0.90            // 90% reliable
				profile.OvertimeTendency = 0.25 + mathrand.Float64()*0.35 // 25-60%
				profile.WeekendWorkRate = 0.20 + mathrand.Float64()*0.30  // 20-50%
			case "tổ trưởng", "giám sát":
				profile.AttendanceRate = 0.90 + mathrand.Float64()*0.08   // 90-98%
				profile.IsReliable = mathrand.Float64() < 0.95            // 95% reliable
				profile.OvertimeTendency = 0.30 + mathrand.Float64()*0.40 // 30-70%
				profile.WeekendWorkRate = 0.25 + mathrand.Float64()*0.35  // 25-60%
			case "kỹ thuật":
				profile.AttendanceRate = 0.92 + mathrand.Float64()*0.06   // 92-98%
				profile.IsReliable = mathrand.Float64() < 0.95            // 95% reliable
				profile.OvertimeTendency = 0.35 + mathrand.Float64()*0.45 // 35-80%
				profile.WeekendWorkRate = 0.30 + mathrand.Float64()*0.40  // 30-70%
			default:
				profile.AttendanceRate = 0.80 + mathrand.Float64()*0.15
				profile.IsReliable = mathrand.Float64() < 0.75
				profile.OvertimeTendency = 0.15 + mathrand.Float64()*0.25
				profile.WeekendWorkRate = 0.10 + mathrand.Float64()*0.20
			}

			employeeProfiles[pe.EmployeeID] = profile
		}
	}

	projectPayrates := make(map[uint]*domain.Payrate)
	for i, payrate := range payrates {
		projectPayrates[payrate.ProjectID] = &payrates[i]
	}

	// Generate timesheets for the past year
	startDate := clock.Now().AddDate(-1, 0, 0)
	endDate := clock.Now().AddDate(0, 0, -1)

	timesheetCount := 0
	for _, projectEmployee := range projectEmployees {
		payrate, exists := projectPayrates[projectEmployee.ProjectID]
		if !exists {
			continue
		}

		var payrateConfig map[string]int
		if err := json.Unmarshal([]byte(payrate.PayrateJSON), &payrateConfig); err != nil {
			continue
		}

		employeeProfile := employeeProfiles[projectEmployee.EmployeeID]

		// Adjust time range based on project employee assignment dates
		actualStartDate := startDate
		if projectEmployee.StartDate.After(startDate) {
			actualStartDate = projectEmployee.StartDate
		}

		actualEndDate := endDate
		if projectEmployee.LastDate != nil && projectEmployee.LastDate.Before(endDate) {
			actualEndDate = *projectEmployee.LastDate
		}

		for current := actualStartDate; current.Before(actualEndDate) || current.Equal(actualEndDate); current = current.AddDate(0, 0, 1) {
			// Skip Sundays (Vietnamese work week is Monday-Saturday)
			if current.Weekday() == time.Sunday {
				continue
			}

			// Seasonal and individual attendance patterns
			baseAttendanceRate := employeeProfile.AttendanceRate

			// Seasonal adjustments
			month := int(current.Month())
			seasonalMultiplier := 1.0
			switch month {
			case 1, 2: // Tet season - reduced attendance
				seasonalMultiplier = 0.7
			case 6, 7, 8: // Summer vacation season
				seasonalMultiplier = 0.85
			case 9, 10, 11: // Busy manufacturing season
				seasonalMultiplier = 1.1
			case 12: // Year-end - moderate reduction
				seasonalMultiplier = 0.9
			}

			adjustedAttendanceRate := baseAttendanceRate * seasonalMultiplier

			// Random attendance check
			if mathrand.Float64() > adjustedAttendanceRate {
				continue
			}

			// Public holidays - very low work probability
			if isVietnameseHoliday(current) {
				if mathrand.Float64() > 0.15 { // Only 15% chance of work on holidays
					continue
				}
			}

			var payType string
			var hoursWorked float64

			// Saturday work
			if current.Weekday() == time.Saturday {
				if mathrand.Float64() > employeeProfile.WeekendWorkRate {
					continue
				}
				payType = "weekend"
				hoursWorked = 4 + mathrand.Float64()*4 // 4-8 hours on Saturday
			} else {
				// Weekday work
				baseHours := 8.0

				// Overtime decision based on individual tendency and seasonal factors
				overtimeChance := employeeProfile.OvertimeTendency
				if month >= 9 && month <= 11 { // Q4 busy season
					overtimeChance *= 1.5
				}

				if mathrand.Float64() < overtimeChance {
					payType = "overtime"
					overtimeHours := 1 + mathrand.Float64()*3 // 1-4 hours overtime
					hoursWorked = baseHours + overtimeHours
				} else {
					payType = "normal"
					// Slight variation in normal working hours
					variation := (mathrand.Float64() - 0.5) * 1.0 // ±0.5 hours
					hoursWorked = baseHours + variation
					if hoursWorked < 6.0 {
						hoursWorked = 6.0 // Minimum 6 hours for normal day
					}
				}
			}

			// Special holiday pay type
			if isVietnameseHoliday(current) {
				payType = "holiday"
			}

			hoursWorked = math.Round(hoursWorked*100) / 100 // Round to 2 decimal places

			payrateAmount, exists := payrateConfig[payType]
			if !exists {
				payrateAmount = payrateConfig["normal"]
			}

			// Realistic timesheet status distribution
			var timesheetStatus domain.TimesheetStatus
			var lastApprovedBy *uint
			var approvedAt *time.Time

			// Status depends on reliability and recency
			daysAgo := int(time.Since(current).Hours() / 24)
			statusRand := mathrand.Float64()

			if employeeProfile.IsReliable {
				switch {
				case daysAgo > 30: // Old entries are mostly approved
					if statusRand < 0.90 {
						timesheetStatus = domain.TimesheetStatusApproved
						lastApprovedBy = &adminUser.ID
						approvedTime := current.Add(time.Duration(1+mathrand.IntN(48)) * time.Hour)
						approvedAt = &approvedTime
					} else {
						timesheetStatus = domain.TimesheetStatusRejected
					}
				case daysAgo > 7: // Recent entries may still be pending
					if statusRand < 0.75 {
						timesheetStatus = domain.TimesheetStatusApproved
						lastApprovedBy = &adminUser.ID
						approvedTime := current.Add(time.Duration(1+mathrand.IntN(120)) * time.Hour)
						approvedAt = &approvedTime
					} else if statusRand < 0.90 {
						timesheetStatus = domain.TimesheetStatusPendingApproval
					} else {
						timesheetStatus = domain.TimesheetStatusRejected
					}
				default: // Very recent entries
					if statusRand < 0.40 {
						timesheetStatus = domain.TimesheetStatusApproved
						lastApprovedBy = &adminUser.ID
						approvedTime := current.Add(time.Duration(1+mathrand.IntN(48)) * time.Hour)
						approvedAt = &approvedTime
					} else if statusRand < 0.80 {
						timesheetStatus = domain.TimesheetStatusPendingApproval
					} else {
						timesheetStatus = domain.TimesheetStatusPendingApproval
					}
				}
			} else {
				// Less reliable employees have more rejections and pending entries
				switch {
				case statusRand < 0.60:
					timesheetStatus = domain.TimesheetStatusApproved
					lastApprovedBy = &adminUser.ID
					approvedTime := current.Add(time.Duration(1+mathrand.IntN(72)) * time.Hour)
					approvedAt = &approvedTime
				case statusRand < 0.75:
					timesheetStatus = domain.TimesheetStatusPendingApproval
				case statusRand < 0.90:
					timesheetStatus = domain.TimesheetStatusRejected
				default:
					timesheetStatus = domain.TimesheetStatusPendingApproval
				}
			}

			timesheet := &domain.Timesheet{
				ProjectID:     projectEmployee.ProjectID,
				EmployeeID:    projectEmployee.EmployeeID,
				PayrateID:     payrate.ID,
				Date:          current,
				HoursWorked:   hoursWorked,
				PayType:       payType,
				PayRate:       int64(payrateAmount),
				Status:        timesheetStatus,
				PaymentStatus: domain.PaymentStatusPending,
				CreatedBy:     adminUser.ID,
				ApprovedBy:    lastApprovedBy,
				ApprovedAt:    approvedAt,
			}

			if err := timesheetRepo.Create(ctx, timesheet); err != nil {
				return fmt.Errorf("create timesheet for employee %d on %s: %w", projectEmployee.EmployeeID, current.Format("2006-01-02"), err)
			}
			timesheetCount++
		}
	}

	fmt.Printf("    Created %d realistic timesheets with individual attendance patterns and Vietnamese work culture\n", timesheetCount)
	return nil
}
