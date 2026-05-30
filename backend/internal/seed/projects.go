package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	mathrand "math/rand/v2"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"

	"gorm.io/gorm"
)

func (s *Seeder) seedProjects(ctx context.Context, db *gorm.DB) error {
	projectRepo := persistence.NewProjectRepository(&persistence.Database{DB: db})

	var adminUser domain.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	// Track existing project codes to avoid duplicates
	existingCodes := make(map[string]bool)
	var existingProjects []domain.Project
	if err := db.Find(&existingProjects).Error; err != nil {
		return fmt.Errorf("check existing projects: %w", err)
	}

	for _, proj := range existingProjects {
		existingCodes[proj.Code] = true
	}

	// Generate realistic project profiles
	numProjects := 18
	for i := 0; i < numProjects; i++ {
		projectProfile := GenerateRealisticProjectProfile(existingCodes)

		// Convert status from string to domain enum
		var status domain.ProjectStatus
		switch projectProfile.Status {
		case "draft":
			status = domain.ProjectStatusDraft
		case "active":
			status = domain.ProjectStatusRunning
		case "completed":
			status = domain.ProjectStatusCompleted
		case "cancelled":
			status = domain.ProjectStatusCancelled
		default:
			status = domain.ProjectStatusRunning
		}

		// Generate realistic start and end dates based on project status
		var startDate, endDate time.Time
		now := clock.Now()

		switch status {
		case domain.ProjectStatusDraft:
			// Future projects
			startDate = now.AddDate(0, mathrand.IntN(6)+1, mathrand.IntN(30))
			endDate = startDate.AddDate(0, projectProfile.ExpectedDuration, 0)
		case domain.ProjectStatusRunning:
			// Currently running projects
			monthsRunning := mathrand.IntN(projectProfile.ExpectedDuration)
			startDate = now.AddDate(0, -monthsRunning, -mathrand.IntN(30))
			remainingMonths := projectProfile.ExpectedDuration - monthsRunning
			endDate = now.AddDate(0, remainingMonths, mathrand.IntN(30))
		case domain.ProjectStatusCompleted:
			// Completed projects
			endDate = now.AddDate(0, -mathrand.IntN(12)-1, -mathrand.IntN(30))
			startDate = endDate.AddDate(0, -projectProfile.ExpectedDuration, 0)
		case domain.ProjectStatusCancelled:
			// Cancelled projects (started but didn't finish)
			cancelMonthsAgo := mathrand.IntN(8) + 1
			endDate = now.AddDate(0, -cancelMonthsAgo, -mathrand.IntN(30))
			monthsBeforeCancel := mathrand.IntN(projectProfile.ExpectedDuration/2) + 1
			startDate = endDate.AddDate(0, -monthsBeforeCancel, 0)
		}

		// Calculate realistic financial amounts based on project scope
		var totalPayout, pendingPayable, pendingReceivable, totalReceived float64

		switch status {
		case domain.ProjectStatusCompleted:
			// Completed projects should have most payments settled
			totalPayout = projectProfile.Budget * 0.85       // 85% of budget paid out
			pendingPayable = projectProfile.Budget * 0.05    // 5% still pending to employees
			totalReceived = projectProfile.Budget * 0.95     // 95% received from client
			pendingReceivable = projectProfile.Budget * 0.05 // 5% still pending from client
		case domain.ProjectStatusRunning:
			// Running projects have ongoing payments
			progressRatio := mathrand.Float64()*0.6 + 0.2 // 20-80% progress
			totalPayout = projectProfile.Budget * progressRatio * 0.8
			pendingPayable = projectProfile.Budget * 0.15
			totalReceived = projectProfile.Budget * progressRatio * 0.7
			pendingReceivable = projectProfile.Budget * 0.3
		case domain.ProjectStatusCancelled:
			// Cancelled projects have limited financial activity
			cancelRatio := mathrand.Float64() * 0.4 // up to 40% of original budget
			totalPayout = projectProfile.Budget * cancelRatio * 0.6
			pendingPayable = projectProfile.Budget * cancelRatio * 0.2
			totalReceived = projectProfile.Budget * cancelRatio * 0.5
			pendingReceivable = 0 // typically no more payments expected
		default: // Draft
			// No financial activity yet
			totalPayout, pendingPayable, totalReceived, pendingReceivable = 0, 0, 0, 0
		}

		project := &domain.Project{
			ClientName:           projectProfile.ClientName,
			Name:                 projectProfile.Name,
			Code:                 projectProfile.Code,
			StartDate:            &startDate,
			EndDate:              &endDate,
			ProjectStatus:        status,
			TotalPayoutVND:       totalPayout,
			PendingPayableVND:    pendingPayable,
			PendingReceivableVND: pendingReceivable,
			TotalReceivedVND:     totalReceived,
			CreatedBy:            adminUser.ID,
		}

		if err := projectRepo.Create(ctx, project); err != nil {
			return fmt.Errorf("create project %s: %w", project.Name, err)
		}
	}

	fmt.Printf("    Created %d realistic projects across various industries with proper financial tracking\n", numProjects)
	return nil
}
