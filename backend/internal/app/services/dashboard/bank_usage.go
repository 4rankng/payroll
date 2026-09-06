package dashboard

import (
	"context"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
)

// GetBankUsage returns bank usage breakdown, optionally filtered by project.
func (s *Service) GetBankUsage(ctx context.Context, req *dto.BankUsageRequest) (*dto.BankUsageResponse, error) {
	s.logger.Info("Getting bank usage", "project_id", req.ProjectID)

	if req.ProjectID != nil && *req.ProjectID > 0 {
		return s.getBankUsageForProject(ctx, *req.ProjectID)
	}
	return s.getBankUsageOverall(ctx)
}

func (s *Service) getBankUsageOverall(ctx context.Context) (*dto.BankUsageResponse, error) {
	rows, err := s.TimesheetAnalyticsRepo.GetBankUsageOverall(ctx)
	if err != nil {
		s.logger.Error("Failed to get overall bank usage", "error", err)
		return nil, err
	}

	var totalEmployees int
	var totalPaid int64
	var totalTransfers int
	for _, r := range rows {
		totalEmployees += r.EmployeeCount
		totalPaid += int64(r.TotalPaid)
		totalTransfers += r.TransferCount
	}

	items := make([]dto.BankUsageItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if totalEmployees > 0 {
			pct = float64(r.EmployeeCount) / float64(totalEmployees) * 100
		}
		items = append(items, dto.BankUsageItem{
			BankName:      r.BankName,
			EmployeeCount: r.EmployeeCount,
			TotalPaidVND:  int64(r.TotalPaid),
			TransferCount: r.TransferCount,
			Percentage:    roundPct(pct),
		})
	}

	return &dto.BankUsageResponse{
		Banks:          items,
		TotalEmployees: totalEmployees,
		TotalPaidVND:   totalPaid,
		TotalTransfers: totalTransfers,
	}, nil
}

func (s *Service) getBankUsageForProject(ctx context.Context, projectID uint) (*dto.BankUsageResponse, error) {
	rows, err := s.TimesheetAnalyticsRepo.GetBankUsageByProject(ctx, projectID)
	if err != nil {
		s.logger.Error("Failed to get bank usage for project", "project_id", projectID, "error", err)
		return nil, err
	}

	// Fetch project name
	project, err := s.ProjectRepo.GetByID(ctx, projectID)
	projectName := ""
	if err == nil && project != nil {
		projectName = project.Name
	}

	var totalEmployees int
	var totalPaid int64
	var totalTransfers int
	for _, r := range rows {
		totalEmployees += r.EmployeeCount
		totalPaid += int64(r.TotalPaid)
		totalTransfers += r.TransferCount
	}

	items := make([]dto.BankUsageItem, 0, len(rows))
	for _, r := range rows {
		pct := 0.0
		if totalEmployees > 0 {
			pct = float64(r.EmployeeCount) / float64(totalEmployees) * 100
		}
		items = append(items, dto.BankUsageItem{
			BankName:      r.BankName,
			EmployeeCount: r.EmployeeCount,
			TotalPaidVND:  int64(r.TotalPaid),
			TransferCount: r.TransferCount,
			Percentage:    roundPct(pct),
		})
	}

	pid := projectID
	return &dto.BankUsageResponse{
		Banks:          items,
		TotalEmployees: totalEmployees,
		TotalPaidVND:   totalPaid,
		TotalTransfers: totalTransfers,
		ProjectID:      &pid,
		ProjectName:    projectName,
	}, nil
}

// GetBankUsageAllProjects returns bank usage broken down by every active project plus overall.
func (s *Service) GetBankUsageAllProjects(ctx context.Context) (*dto.BankUsageAllProjectsResponse, error) {
	s.logger.Info("Getting bank usage for all projects")

	// Two heavy GROUP BY scans over paid timesheets; serve from a short-lived
	// cache. Cleared with the "dashboard:*" pattern by the cache invalidation
	// handler on timesheet/employee/project mutations.
	cacheKey := s.CacheService.GenerateDashboardCacheKey("bank_usage_all_projects")
	var cached dto.BankUsageAllProjectsResponse
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	projectRows, err := s.TimesheetAnalyticsRepo.GetBankUsageAllProjects(ctx)
	if err != nil {
		s.logger.Error("Failed to get bank usage all projects", "error", err)
		return nil, err
	}

	// Group by project
	type projectKey struct {
		id   uint
		name string
	}
	projectMap := make(map[projectKey][]dto.BankUsageItem)
	projectTotals := make(map[projectKey]int)
	var projectOrder []projectKey

	for _, r := range projectRows {
		key := projectKey{id: r.ProjectID, name: r.ProjectName}
		if _, exists := projectMap[key]; !exists {
			projectOrder = append(projectOrder, key)
		}
		projectMap[key] = append(projectMap[key], dto.BankUsageItem{
			BankName:      r.BankName,
			EmployeeCount: r.EmployeeCount,
			TotalPaidVND:  int64(r.TotalPaid),
			TransferCount: r.TransferCount,
		})
		projectTotals[key] += r.EmployeeCount
	}

	// Compute percentages per project
	projects := make([]dto.ProjectBankUsageItem, 0, len(projectOrder))
	for _, key := range projectOrder {
		banks := projectMap[key]
		total := projectTotals[key]
		for i := range banks {
			if total > 0 {
				banks[i].Percentage = roundPct(float64(banks[i].EmployeeCount) / float64(total) * 100)
			}
		}
		projects = append(projects, dto.ProjectBankUsageItem{
			ProjectID:      key.id,
			ProjectName:    key.name,
			Banks:          banks,
			TotalEmployees: total,
		})
	}

	// Get overall
	overall, err := s.getBankUsageOverall(ctx)
	if err != nil {
		return nil, err
	}

	response := &dto.BankUsageAllProjectsResponse{
		Projects: projects,
		Overall:  *overall,
	}

	_ = s.CacheService.Set(ctx, cacheKey, response, constants.DashboardSummaryCacheTTL)

	return response, nil
}

func roundPct(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
