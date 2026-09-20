package main

import (
	"fmt"
	"net/url"
	"strconv"
)

func hasImportedAdvanceAssignment(projects []EmployeeProjectInfo) bool {
	hasImported := false
	for _, project := range projects {
		if project.PaymentSchedule == "flexible" {
			if project.CheckInEnabled {
				return false
			}
			hasImported = true
		}
	}
	return hasImported
}

// Reserve the maximum signed BIGINT for negative API scenarios. Small magic
// IDs such as 999999 can be real records in a populated local test database.
const nonexistentID = 9223372036854775807

func loadAllPages[T any](client *APIClient, path string) ([]T, error) {
	endpoint, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	query := endpoint.Query()
	query.Set("pageSize", "200")
	var records []T
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		var batch []T
		response, err := client.GetInto(endpoint.String(), &batch)
		if err != nil {
			return nil, fmt.Errorf("discover %s page %d: %w", path, page, err)
		}
		records = append(records, batch...)
		if response.Pagination == nil || page >= response.Pagination.TotalPages {
			return records, nil
		}
		if len(batch) == 0 || response.Pagination.Page != page {
			return nil, fmt.Errorf("discover %s: inconsistent pagination at page %d", path, page)
		}
	}
}

// hasRegisteredDisbursementProvider reports whether this server process has a
// disbursement provider wired. It is the authoritative availability signal: the
// payout settings flag lives in the database and can be on in a deployment that
// registered no provider (a local run, for example), where the poller never
// starts and the bulk-transfer endpoints legitimately refuse to work.
func hasRegisteredDisbursementProvider(client *APIClient) bool {
	var settings struct {
		RegisteredProviders []string `json:"registered_providers"`
	}
	if _, err := client.GetInto("/api/v1/admin/settings/disbursement", &settings); err != nil {
		// Unknown is treated as available so a broken probe cannot silently
		// retire the provider-dependent flows.
		return true
	}
	return len(settings.RegisteredProviders) > 0
}

// partnerWeeklyProject returns a weekly project the given partner can actually
// upload into. Discovery picks the weekly project globally while partners come
// from config, so the two are not guaranteed to overlap; a partner upload to a
// project outside their scope is refused by design. Prefer the discovered
// project when it is visible, otherwise fall back to any weekly project the
// partner sees, and report false when there is none (caller skips).
func partnerWeeklyProject(client *APIClient, preferred *ProjectResponse) (*ProjectResponse, bool) {
	projects, err := loadAllPages[ProjectResponse](client, "/api/v1/projects")
	if err != nil {
		return preferred, preferred != nil
	}
	if preferred != nil {
		for i := range projects {
			if projects[i].ID == preferred.ID {
				return &projects[i], true
			}
		}
	}
	for i := range projects {
		if projects[i].WeeklySalaryEmployeeCount > 0 {
			return &projects[i], true
		}
	}
	return nil, false
}
