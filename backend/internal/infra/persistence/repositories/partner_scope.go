package repositories

// PartnerScope is the precomputed set of IDs that determine which timesheet rows
// are visible to a given partner user. It is the materialised equivalent of the
// partnerAccessCondition() SQL fragment:
//
//   - EmployeeIDs covers the three employee-scoped access paths:
//     (a) employees.created_by = partnerID
//     (b) project_employees.created_by = partnerID  (DISTINCT employee_id)
//     (c) employee_users.user_id = partnerID        (employee_id)
//   - ProjectIDs covers the project-scoped access path:
//     (d) project_users.user_id = partnerID         (project_id)
//
// Once materialised, the per-row access predicate collapses from 3 correlated
// EXISTS subqueries into a simple `employee_id IN (...) OR project_id IN (...)`,
// which the optimiser serves via index lookups.
//
// Both slices are sorted ascending and deduplicated so callers can use them
// directly in `IN (?)` clauses without further normalisation.
type PartnerScope struct {
	PartnerID   uint
	EmployeeIDs []uint
	ProjectIDs  []uint
}

// Empty reports whether the scope grants no access at all. Callers can use this
// to short-circuit queries that would otherwise return zero rows after applying
// an empty `IN ()` clause (which some drivers reject).
func (s *PartnerScope) Empty() bool {
	return len(s.EmployeeIDs) == 0 && len(s.ProjectIDs) == 0
}
