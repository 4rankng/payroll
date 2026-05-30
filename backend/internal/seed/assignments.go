package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	mathrand "math/rand/v2"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"

	"gorm.io/gorm"
)

func (s *Seeder) seedProjectEmployees(ctx context.Context, db *gorm.DB) error {
	projectEmployeeRepo := persistence.NewProjectEmployeeRepository(&persistence.Database{DB: db})

	var adminUser domain.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	var projects []domain.Project
	if err := db.Where("project_status IN ?", []string{"draft", "active"}).Find(&projects).Error; err != nil {
		return fmt.Errorf("find active projects: %w", err)
	}

	var employees []domain.Employee
	if err := db.Find(&employees).Error; err != nil {
		return fmt.Errorf("find employees: %w", err)
	}

	// Use realistic Vietnamese position hierarchy - remove unused variable

	// Position distribution weights (more junior positions than senior)
	positionWeights := map[string]int{
		"thực tập":       15, // 15%
		"phổ thông":      35, // 35%
		"có kinh nghiệm": 25, // 25%
		"chuyên môn":     15, // 15%
		"tổ trưởng":      5,  // 5%
		"giám sát":       3,  // 3%
		"kỹ thuật":       2,  // 2%
	}

	// Create weighted position selector
	weightedPositions := make([]string, 0)
	for pos, weight := range positionWeights {
		for i := 0; i < weight; i++ {
			weightedPositions = append(weightedPositions, pos)
		}
	}

	// Assign employees to projects with realistic distribution
	assignmentCount := 0

	for _, project := range projects {
		// Determine team size based on project scope (5-25 employees per project)
		teamSize := 5 + mathrand.IntN(21)
		if teamSize > len(employees) {
			teamSize = len(employees)
		}

		// Randomly select employees for this project
		availableEmployees := make([]domain.Employee, len(employees))
		copy(availableEmployees, employees)

		// Shuffle employees to get random selection
		for i := len(availableEmployees) - 1; i > 0; i-- {
			j := mathrand.IntN(i + 1)
			availableEmployees[i], availableEmployees[j] = availableEmployees[j], availableEmployees[i]
		}

		for i := 0; i < teamSize && i < len(availableEmployees); i++ {
			employee := availableEmployees[i]

			// Determine start date based on project timeline
			startDate := clock.Now().AddDate(0, -2, 0)
			if project.StartDate != nil {
				startDate = *project.StartDate
				// Add some variation - employees might join slightly after project start
				if mathrand.Float64() < 0.3 { // 30% chance of joining after project start
					variation := mathrand.IntN(30) // up to 30 days later
					startDate = startDate.AddDate(0, 0, variation)
				}
			}

			// Select position using weighted distribution
			position := weightedPositions[mathrand.IntN(len(weightedPositions))]

			projectEmployee := &domain.ProjectEmployee{
				ProjectID:    project.ID,
				EmployeeID:   employee.ID,
				EmployeeName: employee.Fullname,
				EmployeeCCCD: employee.CCCD,
				EmployeeCode: fmt.Sprintf("EMP%04d", employee.ID),
				Position:     position,
				StartDate:    startDate,
				CreatedBy:    adminUser.ID,
			}

			// For completed or cancelled projects, some employees might have end dates
			if project.ProjectStatus == domain.ProjectStatusCompleted || project.ProjectStatus == domain.ProjectStatusCancelled {
				if mathrand.Float64() < 0.2 { // 20% chance employee left before project end
					daysBeforeEnd := mathrand.IntN(60) + 1 // 1-60 days before project end
					if project.EndDate != nil {
						lastDate := project.EndDate.AddDate(0, 0, -daysBeforeEnd)
						if lastDate.After(startDate) {
							projectEmployee.LastDate = &lastDate
						}
					}
				}
			}

			if err := projectEmployeeRepo.Create(ctx, projectEmployee); err != nil {
				return fmt.Errorf("create project employee assignment for %s: %w", employee.Fullname, err)
			}
			assignmentCount++
		}
	}

	fmt.Printf("    Created %d employee-project assignments with realistic position distribution\n", assignmentCount)
	return nil
}
