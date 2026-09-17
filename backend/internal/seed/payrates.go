package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"encoding/json"
	"fmt"
	mathrand "math/rand/v2"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence"

	"gorm.io/gorm"
)

func (s *Seeder) seedPayrates(ctx context.Context, db *gorm.DB) error {
	payrateRepo := persistence.NewPayrateRepository(&persistence.Database{DB: db})

	var adminUser domain.User
	if err := db.Where("username = ?", "admin").First(&adminUser).Error; err != nil {
		return fmt.Errorf("find admin user: %w", err)
	}

	var projects []domain.Project
	if err := db.Find(&projects).Error; err != nil {
		return fmt.Errorf("find projects: %w", err)
	}

	// Get unique positions used in project assignments
	var projectEmployees []domain.ProjectEmployee
	if err := db.Find(&projectEmployees).Error; err != nil {
		return fmt.Errorf("find project employees: %w", err)
	}

	// Group project employees by project to determine position distribution per project
	projectPositions := make(map[uint]map[string]bool)
	for _, pe := range projectEmployees {
		if projectPositions[pe.ProjectID] == nil {
			projectPositions[pe.ProjectID] = make(map[string]bool)
		}
		projectPositions[pe.ProjectID][pe.Position] = true
	}

	payrateCount := 0
	for _, project := range projects {
		// Determine base rates for this project based on industry and budget
		baseMultiplier := 1.0

		// Adjust rates based on project characteristics
		if project.TotalReceivedVND > 5000000000 { // > 5 billion VND = high-budget project
			baseMultiplier = 1.3
		} else if project.TotalReceivedVND > 2000000000 { // > 2 billion VND = medium budget
			baseMultiplier = 1.15
		}

		// Create realistic payrate structure for each position used in the project
		positions := projectPositions[project.ID]
		if len(positions) == 0 {
			// Fallback: use standard rates if no employees assigned yet
			positions = map[string]bool{"phổ thông": true}
		}

		for position := range positions {
			// Get base rates for this position
			baseRates := GenerateRealisticPayrates(position, int(float64(vietnamesePositions[position].BaseRate)*baseMultiplier))

			// Add some project-specific variation (±10%)
			payrateConfig := make(map[string]int)
			for payType, baseRate := range baseRates {
				variation := 0.9 + mathrand.Float64()*0.2 // 90%-110%
				payrateConfig[payType] = int(float64(baseRate) * variation)
			}

			payrateJSON, _ := json.Marshal(payrateConfig)

			// Set effective date
			fromDate := clock.Now().AddDate(0, -6, 0)
			if project.StartDate != nil {
				fromDate = *project.StartDate
				// Don't set dates too far in the past
				if project.StartDate.Before(clock.Now().AddDate(-2, 0, 0)) {
					fromDate = clock.Now().AddDate(-2, 0, 0)
				}
			}

			payrate := &domain.Payrate{
				ProjectID: project.ID,
				Payrate:   domain.PayrateConfiguration(payrateJSON),
				FromDate:  fromDate,
				CreatedBy: adminUser.ID,
			}

			if err := payrateRepo.Create(ctx, payrate); err != nil {
				return fmt.Errorf("create payrate for project %s position %s: %w", project.Name, position, err)
			}
			payrateCount++
		}

		// If no specific positions, create a general payrate
		if len(positions) == 0 {
			baseRates := GenerateRealisticPayrates("phổ thông", int(float64(25000)*baseMultiplier))
			payrateJSON, _ := json.Marshal(baseRates)

			fromDate := clock.Now().AddDate(0, -6, 0)
			if project.StartDate != nil {
				fromDate = *project.StartDate
			}

			payrate := &domain.Payrate{
				ProjectID: project.ID,
				Payrate:   domain.PayrateConfiguration(payrateJSON),
				FromDate:  fromDate,
				CreatedBy: adminUser.ID,
			}

			if err := payrateRepo.Create(ctx, payrate); err != nil {
				return fmt.Errorf("create default payrate for project %s: %w", project.Name, err)
			}
			payrateCount++
		}
	}

	fmt.Printf("    Created %d position-based payrate configurations following Vietnamese labor standards\n", payrateCount)
	return nil
}
