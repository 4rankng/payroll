// Package identity answers the single question "which account owns this phone
// number?" for every entry point that accepts a mobile as a credential.
//
// A Vietnamese mobile reaches the business through ONE of two stores, and the
// two are not interchangeable:
//
//   - users.mobile — admin / partner / adv_partner accounts. The column is
//     unique, so at most one account can hold a number.
//   - employees.mobile — employees. The account (when the worker has one) is
//     reached through employees.user_id. Duplicates ARE allowed by the schema:
//     a worker can be typed twice, in two projects, by two partners.
//
// Employee numbers live ONLY in employees.mobile. They are deliberately never
// copied into users.mobile: the unique index there would reject the duplicate
// numbers that already exist in production, and a copy would be a second
// source of truth able to drift from the employee record.
//
// Login, the Zalo-OTP password reset and the create-form duplicate check used
// to each re-implement the lookup order (canonical form first, raw value as
// fallback) and could drift apart. They all call in here now.
package identity

import (
	"context"
	"strings"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/phone"
)

// notFoundMsg is the message both former call sites used, kept so error
// handling (and any log matching it) is unchanged.
const notFoundMsg = "user not found"

// Users is the narrow subset of domain.UserRepository the resolver needs.
type Users interface {
	GetByMobile(ctx context.Context, mobile string) (*domain.User, error)
	GetByID(ctx context.Context, id uint) (*domain.User, error)
}

// Employees is the narrow subset of domain.EmployeeRepository the resolver
// needs. It is ListByMobile — not GetByMobile — so the resolver always sees
// the complete match set and can refuse an ambiguous number itself instead of
// depending on the repository's duplicate policy.
type Employees interface {
	ListByMobile(ctx context.Context, mobile string) ([]*domain.Employee, error)
}

// Resolver resolves a typed phone number to an account across both stores.
type Resolver struct {
	users     Users
	employees Employees // may be nil — employee numbers then never resolve
}

// New builds a resolver over the two phone stores. employees may be nil for
// builds and fakes that do not expose the employees repository.
func New(users Users, employees Employees) *Resolver {
	return &Resolver{users: users, employees: employees}
}

// MobileCandidates lists the stored forms a typed number may appear as, in
// lookup order: the canonical Vietnamese domestic form (0XXXXXXXXX) first,
// then the raw value when it differs — a few rows were stored before
// normalization, for example with the 84 country code. A value that is not a
// parseable Vietnamese mobile yields the raw (trimmed) value alone, and an
// empty value yields nothing, so an empty input can never match a row whose
// mobile column is empty.
func MobileCandidates(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	normalized, err := phone.NormalizeVietnameseMobile(trimmed)
	if err != nil || normalized == trimmed {
		return []string{trimmed}
	}
	return []string{normalized, trimmed}
}

// EmployeesByMobile returns every employee whose stored mobile matches raw,
// taken from the first candidate form that matches anything. All matches are
// returned: duplicate numbers are allowed by the schema, so collapsing them
// here would hide an ambiguity the caller may have to refuse.
//
// A lookup error is treated as "no match" for that candidate — the same
// fail-closed behaviour the pre-resolver call sites had — so a repository
// outage never resolves to some other candidate's account.
func EmployeesByMobile(ctx context.Context, repo Employees, raw string) []*domain.Employee {
	if repo == nil {
		return nil
	}
	for _, candidate := range MobileCandidates(raw) {
		matches, err := repo.ListByMobile(ctx, candidate)
		if err != nil || len(matches) == 0 {
			continue
		}
		return matches
	}
	return nil
}

// ResolveUserByMobile maps a typed phone number to exactly one account.
//
// Lookup order (the one rule all call sites share):
//  1. users.mobile, for accounts whose role stores its number there
//     (admin / partner / adv_partner);
//  2. employees.mobile, following employees.user_id to the linked account.
//
// It fails closed:
//   - no account carries the number → not-found error. Callers keep their
//     anti-enumeration paths (login says "invalid credentials", the Zalo reset
//     returns a dummy session and sends nothing);
//   - two or more employees carry the number → conflict error. The caller MUST
//     treat it as "no account": minting a token or sending an OTP would hand
//     whoever holds the number access to an arbitrarily chosen employee.
//
// The account linked from an employee record is returned regardless of its
// role: the employee record owns the number, so whatever account it points at
// owns it too. (Every linked account is an employee account in production, so
// this only keeps login and password reset from disagreeing if that changes.)
func (r *Resolver) ResolveUserByMobile(ctx context.Context, raw string) (*domain.User, error) {
	if r.users == nil {
		return nil, domain.NewNotFoundError(notFoundMsg)
	}

	for _, candidate := range MobileCandidates(raw) {
		user, err := r.users.GetByMobile(ctx, candidate)
		if err == nil && user != nil && storesMobileInUserRow(user.Role) {
			return user, nil
		}
	}

	matches := EmployeesByMobile(ctx, r.employees, raw)
	if len(matches) > 1 {
		return nil, domain.NewConflictError(constants.MsgMobileSharedByEmployeesVN)
	}
	for _, employee := range matches {
		if employee.UserID == nil {
			continue
		}
		if user, err := r.users.GetByID(ctx, *employee.UserID); err == nil && user != nil {
			return user, nil
		}
	}

	return nil, domain.NewNotFoundError(notFoundMsg)
}

// storesMobileInUserRow reports whether the role keeps its phone number in
// users.mobile. Employees do not: their number belongs to the employee record
// (see the package comment), so a users row that also carries one — a legacy
// copy — must not shadow the authoritative employee record.
//
// Adv-partner accounts are accepted for symmetry with their role; the user
// write paths currently only allow admin/partner to set users.mobile.
func storesMobileInUserRow(role domain.UserRole) bool {
	switch role {
	case domain.RoleAdmin, domain.RolePartner, domain.RoleAdvPartner:
		return true
	default:
		return false
	}
}
