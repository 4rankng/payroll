package employee

import (
	"context"
	"strconv"
	"strings"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

func (s *EmployeeService) GetEmployee(ctx context.Context, id uint) (*domain.Employee, error) {
	return s.EmployeeRepo.GetByID(ctx, id)
}

func (s *EmployeeService) GetEmployeeForUpdate(ctx context.Context, id uint) (*domain.Employee, error) {
	return s.EmployeeRepo.GetByIDForUpdate(ctx, id)
}

// EmployeeHasAccess reports whether actorUserID has project-scope access to
// the given employee (the actor is assigned to a project that employee belongs
// to). Used by the adv_partner scoped update path to enforce ownership before
// any mutation. Reuses ProjectEmployeeRepo.HasAccessViaProject — the same join
// backing the scoped list/detail access checks (see lesson_partner_403_employee_detail).
func (s *EmployeeService) EmployeeHasAccess(ctx context.Context, employeeID, actorUserID uint) (bool, error) {
	return s.ProjectEmployeeRepo.HasAccessViaProject(ctx, employeeID, actorUserID)
}

// GetUserByID retrieves a user by ID - helper method for employee operations
func (s *EmployeeService) GetUserByID(ctx context.Context, userID uint) (*domain.User, error) {
	return s.UserRepo.GetByID(ctx, userID)
}

func (s *EmployeeService) ListEmployees(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	return s.EmployeeRepo.List(ctx, filters)
}

func (s *EmployeeService) ListEmployeesWithProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProject, error) {
	return s.EmployeeRepo.ListWithProjects(ctx, filters)
}

func (s *EmployeeService) ListAccessibleIDs(ctx context.Context, userID uint) ([]uint, error) {
	return s.EmployeeRepo.ListAccessibleIDs(ctx, userID)
}

func (s *EmployeeService) ListEmployeesWithAllProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	// Apply a short-lived microcache for high-traffic employee list endpoints.
	if filters.Limit > 0 {
		cacheKey := s.generateEmployeeListCacheKey("all_projects", filters)

		var cached []*domain.EmployeeWithProjects
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}

		result, err := s.EmployeeRepo.ListWithAllProjects(ctx, filters)
		if err != nil {
			return nil, err
		}

		if len(result) > 0 {
			_ = s.cache.Set(ctx, cacheKey, result, constants.EmployeeListCacheTTL)
		}

		return result, nil
	}

	return s.EmployeeRepo.ListWithAllProjects(ctx, filters)
}

func (s *EmployeeService) CountEmployees(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	// Mirror the list microcache: counts are recomputed from the same accessible
	// set as the list and change at the same cadence, so a short TTL dedupes the
	// partner fan-out without going stale on permission changes.
	//
	// The count key deliberately ignores Limit/Offset/Sort — two list queries
	// that differ only in pagination share the same total count, so a single
	// cache entry serves both.
	if filters.Limit > 0 {
		cacheKey := s.generateEmployeeCountCacheKey(filters)

		var cached int64
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
			return cached, nil
		}

		count, err := s.EmployeeRepo.Count(ctx, filters)
		if err != nil {
			return 0, err
		}

		_ = s.cache.Set(ctx, cacheKey, count, constants.EmployeeListCacheTTL)
		return count, nil
	}

	return s.EmployeeRepo.Count(ctx, filters)
}

// generateEmployeeCountCacheKey mirrors generateEmployeeListCacheKey but omits
// Limit/Offset/SortBy/SortOrder, since the total count is invariant under
// pagination and ordering.
func (s *EmployeeService) generateEmployeeCountCacheKey(filters domain.EmployeeFilters) string {
	var b strings.Builder
	b.WriteString("employees:count")

	if filters.CreatedBy != nil {
		b.WriteString(":created_by:")
		b.WriteString(strconv.FormatUint(uint64(*filters.CreatedBy), 10))
	}
	if filters.ProjectID != nil {
		b.WriteString(":project_id:")
		b.WriteString(strconv.FormatUint(uint64(*filters.ProjectID), 10))
	}
	if len(filters.ProjectIDs) > 0 {
		b.WriteString(":project_ids:")
		for i, id := range filters.ProjectIDs {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.FormatUint(uint64(id), 10))
		}
	}
	if filters.AccessibleBy != nil {
		b.WriteString(":accessible_by:")
		b.WriteString(strconv.FormatUint(uint64(*filters.AccessibleBy), 10))
	}
	if filters.Search != "" {
		b.WriteString(":search:")
		b.WriteString(strings.ToLower(strings.TrimSpace(filters.Search)))
	}
	if filters.Status != "" {
		b.WriteString(":status:")
		b.WriteString(filters.Status)
	}
	if filters.FromDate != nil {
		b.WriteString(":from:")
		b.WriteString(filters.FromDate.Format("2006-01-02"))
	}
	if filters.ToDate != nil {
		b.WriteString(":to:")
		b.WriteString(filters.ToDate.Format("2006-01-02"))
	}

	return b.String()
}

func (s *EmployeeService) GetEmployeeByCCCD(ctx context.Context, cccd string) (*domain.Employee, error) {
	return s.EmployeeRepo.GetByCCCD(ctx, cccd)
}

// SearchEmployees delegates to domain service for normalized Vietnamese search
func (s *EmployeeService) SearchEmployees(ctx context.Context, query string, limit int) ([]*domain.EmployeeWithProject, error) {
	return s.EmployeeDomainService.SearchEmployees(ctx, query, limit)
}

// generateEmployeeListCacheKey builds a deterministic cache key for employee list queries.
func (s *EmployeeService) generateEmployeeListCacheKey(operation string, filters domain.EmployeeFilters) string {
	var b strings.Builder
	b.WriteString("employees:")
	b.WriteString(operation)

	if filters.CreatedBy != nil {
		b.WriteString(":created_by:")
		b.WriteString(strconv.FormatUint(uint64(*filters.CreatedBy), 10))
	}
	if filters.ProjectID != nil {
		b.WriteString(":project_id:")
		b.WriteString(strconv.FormatUint(uint64(*filters.ProjectID), 10))
	}
	if len(filters.ProjectIDs) > 0 {
		b.WriteString(":project_ids:")
		for i, id := range filters.ProjectIDs {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(strconv.FormatUint(uint64(id), 10))
		}
	}
	if filters.AccessibleBy != nil {
		b.WriteString(":accessible_by:")
		b.WriteString(strconv.FormatUint(uint64(*filters.AccessibleBy), 10))
	}
	if filters.Search != "" {
		b.WriteString(":search:")
		b.WriteString(strings.ToLower(strings.TrimSpace(filters.Search)))
	}
	if filters.Status != "" {
		b.WriteString(":status:")
		b.WriteString(filters.Status)
	}
	if filters.FromDate != nil {
		b.WriteString(":from:")
		b.WriteString(filters.FromDate.Format("2006-01-02"))
	}
	if filters.ToDate != nil {
		b.WriteString(":to:")
		b.WriteString(filters.ToDate.Format("2006-01-02"))
	}
	if filters.Limit > 0 {
		b.WriteString(":limit:")
		b.WriteString(strconv.Itoa(filters.Limit))
	}
	if filters.Offset > 0 {
		b.WriteString(":offset:")
		b.WriteString(strconv.Itoa(filters.Offset))
	}
	if filters.SortBy != "" {
		b.WriteString(":sort_by:")
		b.WriteString(filters.SortBy)
	}
	if filters.SortOrder != "" {
		b.WriteString(":sort_order:")
		b.WriteString(filters.SortOrder)
	}

	return b.String()
}
