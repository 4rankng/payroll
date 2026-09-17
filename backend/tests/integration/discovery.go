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
