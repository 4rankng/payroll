package domain

import (
	"strings"

	"api-server/internal/pkg/phone"
)

// Employee reachability states.
//
// These describe why an employee cannot be reached on their mobile. They are
// REPORTING-ONLY labels: the project decision is that mobile values are neither
// validated nor normalised on write (a CCCD-shaped value is the single
// exception), so the report must surface bad contact data instead of refusing
// it at input.
//
// The states are NOT mutually exclusive: an employee can be, for example, both
// duplicate_mobile and invalid_mobile (two employees sharing a junk value). The
// single exception is no_phone, which wins outright because there is nothing
// else to say about an employee with no number at all.
const (
	// EmployeeReachabilityStateNoPhone marks an employee with no mobile value on
	// file. No Zalo message can be delivered, and no Zalo account can be
	// resolved, because there is no number to resolve.
	EmployeeReachabilityStateNoPhone = "no_phone"

	// EmployeeReachabilityStateNotLinked marks an employee whose mobile is on
	// file but whose most recent salary-notification row was suppressed by Zalo
	// with the "phone has no linked Zalo account" business code. The number
	// exists; the recipient has not linked it to a Zalo account.
	EmployeeReachabilityStateNotLinked = "not_linked"

	// EmployeeReachabilityStateDuplicateMobile marks an employee whose mobile
	// value is shared with at least one other active employee. Ops cannot tell
	// whose number it is, and phone-based identity resolution is ambiguous.
	EmployeeReachabilityStateDuplicateMobile = "duplicate_mobile"

	// EmployeeReachabilityStateInvalidMobile marks an employee whose mobile
	// value is present but is not a Vietnamese mobile shape. The shape rule is
	// the canonical phone.NormalizeVietnameseMobile, deliberately reused rather
	// than re-implemented so reporting and phone-based login never disagree.
	EmployeeReachabilityStateInvalidMobile = "invalid_mobile"
)

// ZaloNoLinkedAccountProviderCode is Zalo's ZNS business error code for "Số điện
// thoại chưa liên kết tài khoản Zalo" (the phone has no linked Zalo account).
// It mirrors infra/zalo's ErrNoZaloAccount; the literal is repeated here so the
// reporting layer does not take a dependency on the Zalo client package. The
// delivery service stores this code on every suppressed notification row.
const ZaloNoLinkedAccountProviderCode = -118

// employeeReachabilityStates is the canonical order states are reported in.
// Stable ordering keeps list output and API payloads diffable.
var employeeReachabilityStates = []string{
	EmployeeReachabilityStateNoPhone,
	EmployeeReachabilityStateNotLinked,
	EmployeeReachabilityStateInvalidMobile,
	EmployeeReachabilityStateDuplicateMobile,
}

// EmployeeReachabilityStates returns the known states in canonical report order.
func EmployeeReachabilityStates() []string {
	out := make([]string, len(employeeReachabilityStates))
	copy(out, employeeReachabilityStates)
	return out
}

// IsEmployeeReachabilityState reports whether state is one of the known states,
// so callers can reject a mistyped filter instead of silently ignoring it.
func IsEmployeeReachabilityState(state string) bool {
	for _, known := range employeeReachabilityStates {
		if state == known {
			return true
		}
	}
	return false
}

// EmployeeReachabilitySignals are the raw per-employee facts the classifier
// needs. They are assembled by the persistence layer from employees,
// flexpay_salary_notifications and the mobile duplicate count.
type EmployeeReachabilitySignals struct {
	// Mobile is the raw value on file. It is reported as-is: normalisation is
	// explicitly out of scope for this report.
	Mobile string
	// ZNSSuppressedNoLinkedAccount is true when the employee's most recent
	// flexpay_salary_notifications row is suppressed with
	// ZaloNoLinkedAccountProviderCode. Older rows are ignored — an employee who
	// has since been reached is not reported as unreachable.
	ZNSSuppressedNoLinkedAccount bool
	// MobileMatches counts ACTIVE employees sharing this mobile value, including
	// this one. 2 therefore means "one other employee has the same number".
	MobileMatches int64
}

// ClassifyReachabilityStates returns the states that apply to an employee, in
// canonical order, or nil when the employee is reachable.
func ClassifyReachabilityStates(signals EmployeeReachabilitySignals) []string {
	mobile := strings.TrimSpace(signals.Mobile)
	if mobile == "" {
		// no_phone wins outright: with no number on file there is nothing else
		// worth telling ops about this employee.
		return []string{EmployeeReachabilityStateNoPhone}
	}

	states := make([]string, 0, 3)
	if signals.ZNSSuppressedNoLinkedAccount {
		states = append(states, EmployeeReachabilityStateNotLinked)
	}
	if _, err := phone.NormalizeVietnameseMobile(mobile); err != nil {
		states = append(states, EmployeeReachabilityStateInvalidMobile)
	}
	if signals.MobileMatches > 1 {
		states = append(states, EmployeeReachabilityStateDuplicateMobile)
	}
	if len(states) == 0 {
		return nil
	}
	return states
}

// EmployeeReachability is one row of the reachability report: an employee ops
// cannot currently reach, with the payroll exposure attached so the worklist can
// be prioritised by money rather than by row order.
type EmployeeReachability struct {
	EmployeeID uint   `json:"employee_id"`
	Fullname   string `json:"fullname"`
	CCCD       string `json:"cccd"`
	// Mobile is the raw stored value, exactly as found (never normalised), so
	// ops can see the bad data they are being asked to fix.
	Mobile   string   `json:"mobile"`
	Projects []string `json:"projects"`
	// PaidAmount is the employee's lifetime paid salary (timesheets with
	// payment_status = paid, summed over paid_amount).
	PaidAmount int64 `json:"paid_amount"`
	// OutstandingAmount is the salary still awaiting operational processing,
	// using the canonical operational backlog cohort: timesheets that are
	// pending approval or approved and whose payment is pending or failed.
	OutstandingAmount int64 `json:"outstanding_amount"`
	// States lists every reason this employee is unreachable, in canonical
	// order. It is never empty for a reported row.
	States []string `json:"states"`
}

// EmployeeReachabilityFilters scopes the reachability report.
type EmployeeReachabilityFilters struct {
	// ProjectID limits the report to employees with a current assignment on the
	// project. Nil means every employee.
	ProjectID *uint
	// State limits the report to employees in that state. Empty means every
	// state. It does NOT restrict the Summary, which always covers the whole
	// project-scoped cohort so ops can size the worklist.
	State string
	// Limit and Offset paginate the reported rows only; counts always describe
	// the whole cohort.
	Limit  int
	Offset int
}

// EmployeeReachabilitySummary counts employees per state over a cohort.
//
// The per-state counters OVERLAP: an employee who is both invalid_mobile and
// duplicate_mobile is counted once in each, so the counters can sum to more
// than Employees. Employees counts each employee once.
type EmployeeReachabilitySummary struct {
	NoPhone         int64 `json:"no_phone"`
	NotLinked       int64 `json:"not_linked"`
	DuplicateMobile int64 `json:"duplicate_mobile"`
	InvalidMobile   int64 `json:"invalid_mobile"`
	// Employees is the number of distinct employees in at least one state.
	Employees int64 `json:"employees"`
}

// CountForState returns the counter for a single state, or the distinct employee
// count when state is empty. Callers validate the state first, so the fallback
// only ever means "no state filter".
func (s EmployeeReachabilitySummary) CountForState(state string) int64 {
	switch state {
	case EmployeeReachabilityStateNoPhone:
		return s.NoPhone
	case EmployeeReachabilityStateNotLinked:
		return s.NotLinked
	case EmployeeReachabilityStateDuplicateMobile:
		return s.DuplicateMobile
	case EmployeeReachabilityStateInvalidMobile:
		return s.InvalidMobile
	default:
		return s.Employees
	}
}

// UnreachableEmployeesReport is the paginated reachability report.
type UnreachableEmployeesReport struct {
	Employees []*EmployeeReachability     `json:"employees"`
	Summary   EmployeeReachabilitySummary `json:"summary"`
	// Total is the number of employees matching the requested filters, before
	// pagination. Summary.Employees is the larger cohort size when a state
	// filter is applied.
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}
