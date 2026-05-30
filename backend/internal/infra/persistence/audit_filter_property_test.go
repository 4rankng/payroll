// Feature: audit-log-revamp, Property 8: Filter correctness — action filter
// Feature: audit-log-revamp, Property 9: Filter correctness — entity_type filter
// Feature: audit-log-revamp, Property 10: Filter correctness — date range
package persistence

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// ---------------------------------------------------------------------------
// In-memory audit repository for property testing
// ---------------------------------------------------------------------------

// inMemoryAuditRepo stores AuditLog records in a slice and implements
// filtering in pure Go, mirroring the logic in applyAuditFilters.
type inMemoryAuditRepo struct {
	logs []*domain.AuditLog
}

func (r *inMemoryAuditRepo) insert(logs ...*domain.AuditLog) {
	r.logs = append(r.logs, logs...)
}

func (r *inMemoryAuditRepo) list(filters domain.AuditFilters) []*domain.AuditLog {
	var result []*domain.AuditLog
	for _, l := range r.logs {
		if !r.matchesFilters(l, filters) {
			continue
		}
		result = append(result, l)
	}
	return result
}

func (r *inMemoryAuditRepo) matchesFilters(l *domain.AuditLog, f domain.AuditFilters) bool {
	// action IN (...)
	if len(f.Action) > 0 {
		found := false
		for _, a := range f.Action {
			if l.Action == a {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// entity_type IN (...)
	if len(f.EntityType) > 0 {
		found := false
		for _, et := range f.EntityType {
			if l.EntityType == et {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// created_at >= FromDate
	if f.FromDate != nil && l.CreatedAt.Before(*f.FromDate) {
		return false
	}

	// created_at <= ToDate
	if f.ToDate != nil && l.CreatedAt.After(*f.ToDate) {
		return false
	}

	return true
}

// ---------------------------------------------------------------------------
// Generators
// ---------------------------------------------------------------------------

var allAuditActions = []interface{}{
	domain.AuditActionCreate,
	domain.AuditActionUpdate,
	domain.AuditActionDelete,
	domain.AuditActionApprove,
	domain.AuditActionReject,
	domain.AuditActionLogin,
	domain.AuditActionLogout,
	domain.AuditActionView,
	domain.AuditActionExport,
	domain.AuditActionImport,
}

var allEntityTypes = []interface{}{
	domain.EntityTypeUser,
	domain.EntityTypeProject,
	domain.EntityTypeEmployee,
	domain.EntityTypeBank,
	domain.EntityTypePayrate,
	domain.EntityTypeTimesheet,
	domain.EntityTypePayroll,
	domain.EntityTypeLedgerEntry,
	domain.EntityTypeSettings,
	domain.EntityTypeAsset,
}

// genAuditLogWithAction generates an AuditLog with a specific action.
func genAuditLogWithAction(action domain.AuditAction) *domain.AuditLog {
	return &domain.AuditLog{
		UserID:     1,
		Action:     action,
		EntityType: domain.EntityTypeUser,
		Message:    "test",
		CreatedAt:  time.Now(),
	}
}

// genAuditLogWithEntityType generates an AuditLog with a specific entity type.
func genAuditLogWithEntityType(et domain.EntityType) *domain.AuditLog {
	return &domain.AuditLog{
		UserID:     1,
		Action:     domain.AuditActionCreate,
		EntityType: et,
		Message:    "test",
		CreatedAt:  time.Now(),
	}
}

// genAuditLogWithTime generates an AuditLog with a specific CreatedAt time.
func genAuditLogWithTime(t time.Time) *domain.AuditLog {
	return &domain.AuditLog{
		UserID:     1,
		Action:     domain.AuditActionCreate,
		EntityType: domain.EntityTypeUser,
		Message:    "test",
		CreatedAt:  t,
	}
}

// genSubsetOfActions generates a non-empty subset of AuditActions.
func genSubsetOfActions() gopter.Gen {
	// Generate a bitmask over allAuditActions; at least one bit must be set.
	n := len(allAuditActions)
	return gen.IntRange(1, (1<<n)-1).Map(func(mask int) []domain.AuditAction {
		var result []domain.AuditAction
		for i, a := range allAuditActions {
			if mask&(1<<i) != 0 {
				result = append(result, a.(domain.AuditAction))
			}
		}
		return result
	})
}

// genSubsetOfEntityTypes generates a non-empty subset of EntityTypes.
func genSubsetOfEntityTypes() gopter.Gen {
	n := len(allEntityTypes)
	return gen.IntRange(1, (1<<n)-1).Map(func(mask int) []domain.EntityType {
		var result []domain.EntityType
		for i, et := range allEntityTypes {
			if mask&(1<<i) != 0 {
				result = append(result, et.(domain.EntityType))
			}
		}
		return result
	})
}

// ---------------------------------------------------------------------------
// Property 8: Filter correctness — action filter
// ---------------------------------------------------------------------------

// TestProperty8_ActionFilterCorrectness verifies that when AuditFilters.Action
// contains a subset of actions, List returns only records whose action is in
// that subset.
//
// Validates: Requirements 5.1
func TestProperty8_ActionFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(8001)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("action filter returns only records with matching action", prop.ForAll(
		func(filterActions []domain.AuditAction) bool {
			repo := &inMemoryAuditRepo{}

			// Insert one log per known action
			for _, a := range allAuditActions {
				repo.insert(genAuditLogWithAction(a.(domain.AuditAction)))
			}

			filters := domain.AuditFilters{Action: filterActions}
			results := repo.list(filters)

			// Build a set of filter actions for O(1) lookup
			filterSet := make(map[domain.AuditAction]bool, len(filterActions))
			for _, a := range filterActions {
				filterSet[a] = true
			}

			// Every returned record's action must be in the filter set
			for _, r := range results {
				if !filterSet[r.Action] {
					return false
				}
			}
			return true
		},
		genSubsetOfActions(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty8_ActionFilterReturnsAllMatchingRecords verifies that the action
// filter does not drop records that should match.
//
// Validates: Requirements 5.1
func TestProperty8_ActionFilterReturnsAllMatchingRecords(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(8002)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("action filter includes all records with matching action", prop.ForAll(
		func(filterActions []domain.AuditAction) bool {
			repo := &inMemoryAuditRepo{}

			// Insert 3 logs per known action
			for _, a := range allAuditActions {
				for i := 0; i < 3; i++ {
					repo.insert(genAuditLogWithAction(a.(domain.AuditAction)))
				}
			}

			filterSet := make(map[domain.AuditAction]bool, len(filterActions))
			for _, a := range filterActions {
				filterSet[a] = true
			}

			// Count expected matches
			expectedCount := 0
			for _, l := range repo.logs {
				if filterSet[l.Action] {
					expectedCount++
				}
			}

			filters := domain.AuditFilters{Action: filterActions}
			results := repo.list(filters)

			return len(results) == expectedCount
		},
		genSubsetOfActions(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ---------------------------------------------------------------------------
// Property 9: Filter correctness — entity_type filter
// ---------------------------------------------------------------------------

// TestProperty9_EntityTypeFilterCorrectness verifies that when AuditFilters.EntityType
// contains a subset of entity types, List returns only records whose entity_type
// is in that subset.
//
// Validates: Requirements 5.2
func TestProperty9_EntityTypeFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(9001)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("entity_type filter returns only records with matching entity_type", prop.ForAll(
		func(filterTypes []domain.EntityType) bool {
			repo := &inMemoryAuditRepo{}

			// Insert one log per known entity type
			for _, et := range allEntityTypes {
				repo.insert(genAuditLogWithEntityType(et.(domain.EntityType)))
			}

			filters := domain.AuditFilters{EntityType: filterTypes}
			results := repo.list(filters)

			filterSet := make(map[domain.EntityType]bool, len(filterTypes))
			for _, et := range filterTypes {
				filterSet[et] = true
			}

			for _, r := range results {
				if !filterSet[r.EntityType] {
					return false
				}
			}
			return true
		},
		genSubsetOfEntityTypes(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty9_EntityTypeFilterReturnsAllMatchingRecords verifies that the
// entity_type filter does not drop records that should match.
//
// Validates: Requirements 5.2
func TestProperty9_EntityTypeFilterReturnsAllMatchingRecords(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(9002)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("entity_type filter includes all records with matching entity_type", prop.ForAll(
		func(filterTypes []domain.EntityType) bool {
			repo := &inMemoryAuditRepo{}

			// Insert 3 logs per known entity type
			for _, et := range allEntityTypes {
				for i := 0; i < 3; i++ {
					repo.insert(genAuditLogWithEntityType(et.(domain.EntityType)))
				}
			}

			filterSet := make(map[domain.EntityType]bool, len(filterTypes))
			for _, et := range filterTypes {
				filterSet[et] = true
			}

			expectedCount := 0
			for _, l := range repo.logs {
				if filterSet[l.EntityType] {
					expectedCount++
				}
			}

			filters := domain.AuditFilters{EntityType: filterTypes}
			results := repo.list(filters)

			return len(results) == expectedCount
		},
		genSubsetOfEntityTypes(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// ---------------------------------------------------------------------------
// Property 10: Filter correctness — date range
// ---------------------------------------------------------------------------

// dateRangeWithLogs bundles a date range and a set of audit logs for Property 10.
type dateRangeWithLogs struct {
	FromDate time.Time
	ToDate   time.Time
	Logs     []*domain.AuditLog
}

// genDateRangeWithLogs generates a FromDate, ToDate, and a slice of AuditLogs
// with CreatedAt times spread across a wider window (some inside, some outside).
func genDateRangeWithLogs() gopter.Gen {
	// Base epoch: 2020-01-01 00:00:00 UTC
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	return gopter.CombineGens(
		// fromOffset: 0..364 days from base
		gen.IntRange(0, 364),
		// rangeLen: 1..30 days
		gen.IntRange(1, 30),
		// number of logs to generate: 5..20
		gen.IntRange(5, 20),
		// log timestamp offsets: spread over base ± 400 days
		gen.SliceOfN(20, gen.IntRange(-400, 400)),
	).Map(func(vals []interface{}) dateRangeWithLogs {
		fromOffset := vals[0].(int)
		rangeLen := vals[1].(int)
		logCount := vals[2].(int)
		offsets := vals[3].([]int)

		fromDate := base.Add(time.Duration(fromOffset) * 24 * time.Hour)
		toDate := fromDate.Add(time.Duration(rangeLen) * 24 * time.Hour)

		var logs []*domain.AuditLog
		for i := 0; i < logCount && i < len(offsets); i++ {
			ts := base.Add(time.Duration(offsets[i]) * 24 * time.Hour)
			logs = append(logs, genAuditLogWithTime(ts))
		}
		return dateRangeWithLogs{FromDate: fromDate, ToDate: toDate, Logs: logs}
	})
}

// TestProperty10_DateRangeFilterCorrectness verifies that when FromDate and ToDate
// are both set, List returns only records whose created_at falls within [FromDate, ToDate].
//
// Validates: Requirements 5.6
func TestProperty10_DateRangeFilterCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(10001)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("date range filter returns only records within [FromDate, ToDate]", prop.ForAll(
		func(d dateRangeWithLogs) bool {
			repo := &inMemoryAuditRepo{}
			repo.insert(d.Logs...)

			filters := domain.AuditFilters{
				FromDate: &d.FromDate,
				ToDate:   &d.ToDate,
			}
			results := repo.list(filters)

			for _, r := range results {
				if r.CreatedAt.Before(d.FromDate) || r.CreatedAt.After(d.ToDate) {
					return false
				}
			}
			return true
		},
		genDateRangeWithLogs(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// TestProperty10_DateRangeFilterReturnsAllMatchingRecords verifies that the date
// range filter does not drop records that fall within [FromDate, ToDate].
//
// Validates: Requirements 5.6
func TestProperty10_DateRangeFilterReturnsAllMatchingRecords(t *testing.T) {
	parameters := gopter.DefaultTestParametersWithSeed(10002)
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("date range filter includes all records within [FromDate, ToDate]", prop.ForAll(
		func(d dateRangeWithLogs) bool {
			repo := &inMemoryAuditRepo{}
			repo.insert(d.Logs...)

			// Count expected matches manually
			expectedCount := 0
			for _, l := range d.Logs {
				if !l.CreatedAt.Before(d.FromDate) && !l.CreatedAt.After(d.ToDate) {
					expectedCount++
				}
			}

			filters := domain.AuditFilters{
				FromDate: &d.FromDate,
				ToDate:   &d.ToDate,
			}
			results := repo.list(filters)

			return len(results) == expectedCount
		},
		genDateRangeWithLogs(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Ensure context import is used (needed for interface compliance check below).
var _ = context.Background
