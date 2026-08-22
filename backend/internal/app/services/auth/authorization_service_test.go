package auth

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type routeDef struct {
	method string
	path   string
}

// resolveConfigPaths returns absolute paths to the casbin model + policy
// files in repo configs/, regardless of the package dir the test runs in.
func resolveRepoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller for config path")
	}
	return filepath.Join(filepath.Dir(here), "..", "..", "..", "..")
}

func resolveConfigPaths(t *testing.T) (model, policy string) {
	t.Helper()
	repoRoot := resolveRepoRoot(t)
	return filepath.Join(repoRoot, "configs", "casbin_model.conf"),
		filepath.Join(repoRoot, "configs", "casbin_policy.csv")
}

func newTestAuthorizationService(t *testing.T) *AuthorizationService {
	t.Helper()
	model, policy := resolveConfigPaths(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc, err := NewAuthorizationService(model, policy, nil, logger)
	if err != nil {
		t.Fatalf("NewAuthorizationService: %v", err)
	}
	return svc
}

func joinRoute(parts ...string) string {
	var out []string
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			out = append(out, part)
		}
	}
	return "/" + strings.Join(out, "/")
}

func sampleRoutePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") {
			segments[i] = "1"
		}
	}
	return strings.Join(segments, "/")
}

func isEmployeeSelfServicePath(path string) bool {
	if path == "/api/v1/me" || strings.HasPrefix(path, "/api/v1/me/") {
		return true
	}
	return path == "/api/v1/mobile/attendance" || strings.HasPrefix(path, "/api/v1/mobile/attendance/")
}

func discoverEmployeeSelfServiceRoutes(t *testing.T) []routeDef {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(resolveRepoRoot(t), "internal", "app", "bootstrap", "routes*.go"))
	if err != nil {
		t.Fatalf("glob bootstrap route files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no bootstrap route files found")
	}

	groupRe := regexp.MustCompile(`^\s*(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"\)`)
	routeRe := regexp.MustCompile(`^\s*(\w+)\.(GET|POST|PUT|PATCH|DELETE)\("([^"]*)"`)

	seen := make(map[string]bool)
	var routes []routeDef
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read route file %s: %v", file, err)
		}

		groups := map[string]string{
			"protected": "",
			"v1":        "",
		}
		for _, line := range strings.Split(string(content), "\n") {
			if match := groupRe.FindStringSubmatch(line); match != nil {
				name, parent, suffix := match[1], match[2], match[3]
				if prefix, ok := groups[parent]; ok {
					groups[name] = joinRoute(prefix, suffix)
				}
				continue
			}

			match := routeRe.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			groupName, method, routePath := match[1], match[2], match[3]
			prefix, ok := groups[groupName]
			if !ok {
				continue
			}

			fullPath := sampleRoutePath(joinRoute("/api/v1", prefix, routePath))
			if !isEmployeeSelfServicePath(fullPath) {
				continue
			}

			key := method + " " + fullPath
			if seen[key] {
				continue
			}
			seen[key] = true
			routes = append(routes, routeDef{method: method, path: fullPath})
		}
	}

	return routes
}

// TestAdvPartnerRole_AllowList asserts every endpoint the adv_partner role is
// allowed to reach: auth endpoints and advance payment read/import/export.
func TestAdvPartnerRole_AllowList(t *testing.T) {
	svc := newTestAuthorizationService(t)

	allow := []struct {
		path   string
		method string
	}{
		// Auth
		{"/api/v1/auth/me", "GET"},
		{"/api/v1/auth/logout", "POST"},
		{"/api/v1/auth/change-password", "POST"},

		// Advance payment reads
		{"/api/v1/advance-payments", "GET"},
		{"/api/v1/advance-payments/summary", "GET"},
		{"/api/v1/advance-payments/available-months", "GET"},
		{"/api/v1/advance-payments/employees", "GET"},
		{"/api/v1/advance-payments/employees/export", "GET"},
		{"/api/v1/advance-payments/export", "GET"},
		{"/api/v1/advance-payments/reconciliation/export", "GET"},

		// Advance payment imports
		{"/api/v1/advance-payments/import", "POST"},
		{"/api/v1/advance-payments/import/1", "GET"},
		{"/api/v1/advance-payments/import-employee-list", "POST"},

		// Users - read only (broad PUT removed; adv_partner edits route through
		// /adv-partner/users/* which enforces project-scope ownership).
		{"/api/v1/users", "GET"},
		{"/api/v1/users/summary", "GET"},
		{"/api/v1/users/1", "GET"},

		// Employees - read
		{"/api/v1/employees", "GET"},
		{"/api/v1/employees/1", "GET"},

		// Check-in configuration workspace (project scope is enforced by handlers)
		{"/api/v1/projects/checkin-configurable", "GET"},
		{"/api/v1/projects/7/employees/checkin-configuration", "GET"},
		{"/api/v1/projects/7/employees/101/checkin-enabled", "PATCH"},
		{"/api/v1/projects/7/employees/checkin-enabled/disable-inactive", "PATCH"},
		{"/api/v1/projects/7/employees/checkin-enabled/disable-pending", "PATCH"},

		// Notifications (wildcard)
		{"/api/v1/notifications", "GET"},
		{"/api/v1/notifications/unread/count", "GET"},
		{"/api/v1/notifications/1/read", "PUT"},
		{"/api/v1/notifications/read-all", "PUT"},

		// Push notifications
		{"/api/v1/push/subscribe", "POST"},
		{"/api/v1/push/unsubscribe", "POST"},
	}
	for _, tc := range allow {
		assert.True(t,
			svc.CanAccess("adv_partner", tc.path, tc.method),
			"adv_partner should be allowed %s %s", tc.method, tc.path,
		)
	}
}

// TestAdvPartnerRole_DenyList asserts the adv_partner role does NOT inherit
// admin/partner/employee permissions. Financial actions (cancel, settle,
// upload-result, send-email) and general payroll operations are denied.
func TestAdvPartnerRole_DenyList(t *testing.T) {
	svc := newTestAuthorizationService(t)

	deny := []struct {
		path   string
		method string
	}{
		// User management — create/delete denied, and broad PUT denied (edits
		// must route through /adv-partner/users/* with the ownership gate).
		{"/api/v1/users", "POST"},
		{"/api/v1/users/1", "DELETE"},
		{"/api/v1/users/1", "PUT"},

		// Employee data
		{"/api/v1/employees/import", "POST"},

		// Projects outside the bounded check-in workspace
		{"/api/v1/projects", "POST"},
		{"/api/v1/projects/1", "PUT"},
		{"/api/v1/projects/1", "DELETE"},
		{"/api/v1/projects/1/employees", "POST"},

		// Payroll, timesheets, payrates
		{"/api/v1/payrolls/histories", "GET"},
		{"/api/v1/timesheets", "GET"},
		{"/api/v1/payrates", "GET"},

		// Advance payment financial actions (denied)
		{"/api/v1/advance-payments/1/cancel", "POST"},
		{"/api/v1/advance-payments/upload-result", "POST"},
		{"/api/v1/advance-payments/reconciliation/send-email", "POST"},
		{"/api/v1/advance-payments/reconciliation/settle", "POST"},
		{"/api/v1/advance-payments/files", "GET"},
		{"/api/v1/advance-payments/files/1/download", "GET"},
		{"/api/v1/advance-payments/transfer-histories/1/download", "GET"},

		// Fee schedule (admin-only)
		{"/api/v1/admin/advance-payment-fees", "GET"},
		{"/api/v1/admin/disbursement-fees", "GET"},

		// Settings
		{"/api/v1/settings", "GET"},
		{"/api/v1/settings/1", "PUT"},

		// Audit, ledger, transactions, dashboard
		{"/api/v1/audit/logs", "GET"},
		{"/api/v1/ledger/entries", "GET"},
		{"/api/v1/transactions", "GET"},
		{"/api/v1/dashboard/summary", "GET"},

		// Self-service routes
		{"/api/v1/me", "GET"},
		{"/api/v1/me/advance-payment", "GET"},
	}
	for _, tc := range deny {
		assert.False(t,
			svc.CanAccess("adv_partner", tc.path, tc.method),
			"adv_partner should be DENIED %s %s", tc.method, tc.path,
		)
	}
}

// TestExistingRoles_NotRegressed spot-checks that the adv_partner rules
// did not accidentally widen or narrow admin/partner/employee access.
func TestExistingRoles_NotRegressed(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Admin still has wildcard on /api/*
	assert.True(t, svc.CanAccess("admin", "/api/v1/users", "POST"))
	assert.True(t, svc.CanAccess("admin", "/api/v1/settings/1", "DELETE"))

	// Partner keeps employees/projects/timesheets, denied on settings write
	assert.True(t, svc.CanAccess("partner", "/api/v1/employees", "GET"))
	assert.True(t, svc.CanAccess("partner", "/api/v1/projects/1", "PUT"))
	assert.False(t, svc.CanAccess("partner", "/api/v1/settings/1", "PUT"))
	assert.False(t, svc.CanAccess("partner", "/api/v1/audit/logs", "GET"))

	// Employee keeps self-service, denied on admin surfaces
	assert.True(t, svc.CanAccess("employee", "/api/v1/me", "GET"))
	assert.True(t, svc.CanAccess("employee", "/api/v1/me/advance-payment", "GET"))
	assert.True(t, svc.CanAccess("employee", "/api/v1/me/check-in-advance", "GET"))
	assert.True(t, svc.CanAccess("employee", "/api/v1/me/check-in-advance/request", "POST"))
	assert.False(t, svc.CanAccess("employee", "/api/v1/employees", "GET"))

	// adv_partner can read/import advance payments, manage users, denied on admin/partner surfaces
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/advance-payments", "GET"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/advance-payments/import", "POST"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/users", "GET"))
	assert.False(t, svc.CanAccess("adv_partner", "/api/v1/users/1", "PUT"))
	assert.False(t, svc.CanAccess("adv_partner", "/api/v1/users", "POST"))
	assert.False(t, svc.CanAccess("adv_partner", "/api/v1/advance-payments/1/cancel", "POST"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/employees", "GET"))
}

// TestEmployeeRole_AllSelfServiceRoutesAreAllowed audits Gin route registration
// against Casbin policy for employee-owned surfaces. It catches the class of
// regression where a new self-service route is added but the employee policy is
// not updated, causing a 403 before the handler runs.
func TestEmployeeRole_AllSelfServiceRoutesAreAllowed(t *testing.T) {
	svc := newTestAuthorizationService(t)
	routes := discoverEmployeeSelfServiceRoutes(t)
	if len(routes) == 0 {
		t.Fatal("no employee self-service routes discovered")
	}

	for _, route := range routes {
		assert.True(t,
			svc.CanAccess("employee", route.path, route.method),
			"employee should be allowed %s %s", route.method, route.path,
		)
	}
}

// TestPartnerRole_EditRequests asserts the partner can list their own edit
// requests (GET /edit-requests, scoped by the handler to RequestedBy) but
// cannot read a specific request by id (the detail handler has no ownership
// check, so allowing it would be an IDOR hole) or approve/reject (admin-only).
// Regression test for the blanket deny that e7230cb added on the whole resource.
func TestPartnerRole_EditRequests(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Partner can list their own edit requests (handler scopes by RequestedBy)
	assert.True(t, svc.CanAccess("partner", "/api/v1/timesheets/edit-requests", "GET"),
		"partner should be allowed to list edit requests (handler scopes to own)")

	// Partner must NOT reach detail/approve/reject (admin-only)
	partnerDeny := []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/edit-requests/1", "GET"},
		{"/api/v1/timesheets/edit-requests/1/approve", "PUT"},
		{"/api/v1/timesheets/edit-requests/1/reject", "PUT"},
	}
	for _, tc := range partnerDeny {
		assert.False(t, svc.CanAccess("partner", tc.path, tc.method),
			"partner should be DENIED %s %s", tc.method, tc.path)
	}

	// Admin retains full access to the edit-request resource (no regression)
	adminAllow := []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/edit-requests", "GET"},
		{"/api/v1/timesheets/edit-requests/1", "GET"},
		{"/api/v1/timesheets/edit-requests/1/approve", "PUT"},
		{"/api/v1/timesheets/edit-requests/1/reject", "PUT"},
	}
	for _, tc := range adminAllow {
		assert.True(t, svc.CanAccess("admin", tc.path, tc.method),
			"admin should be allowed %s %s", tc.method, tc.path)
	}
}

// TestPartnerRole_PayrollReport asserts the partner can export the payroll
// report by project (GET /payroll/report) — the handler explicitly allows
// RolePartner and scopes results to the partner's own projects via
// filterProjectsForPartner (no cross-tenant data) — but the partner must NOT
// be able to email the report (POST /payroll/report/send-email is admin-only:
// isAdmin gate in the email handler). Regression test for the allow-list that
// e7230cb introduced, which omitted /payroll/report and broke the partner
// export UI (QuickActions / ExportSaoKeDialog / mobile admin TimesheetPage).
func TestPartnerRole_PayrollReport(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Partner can export the payroll report scoped to own projects (handler allows it)
	assert.True(t, svc.CanAccess("partner", "/api/v1/timesheets/payroll/report", "GET"),
		"partner should be allowed to export payroll report (handler scopes to own projects)")

	// Partner must NOT email the payroll report (admin-only: isAdmin handler gate)
	assert.False(t, svc.CanAccess("partner", "/api/v1/timesheets/payroll/report/send-email", "POST"),
		"partner should be DENIED POST /payroll/report/send-email (admin-only)")

	// Partner must NOT upload settlement results (admin-only financial action)
	assert.False(t, svc.CanAccess("partner", "/api/v1/timesheets/payroll/upload-settlement-result", "POST"),
		"partner should be DENIED POST /payroll/upload-settlement-result (admin-only)")

	// Admin retains full access to the payroll-report family (no regression)
	for _, tc := range []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/payroll/report", "GET"},
		{"/api/v1/timesheets/payroll/report/send-email", "POST"},
		{"/api/v1/timesheets/payroll/upload-settlement-result", "POST"},
	} {
		assert.True(t, svc.CanAccess("admin", tc.path, tc.method),
			"admin should be allowed %s %s", tc.method, tc.path)
	}
}

// TestPartnerRole_DeleteTimesheet asserts the partner can delete a single
// timesheet by id (DELETE /timesheets/:id). The DeleteTimesheet handler
// explicitly allows RolePartner and enforces that only pending_approval or
// rejected entries are deletable (IsPaid() is rejected first), so the policy
// must permit DELETE — otherwise the partner UI hits 403 before the handler's
// own status guard can run. Regression for the allow-list e7230cb introduced,
// which omitted DELETE /timesheets/:id (same gap class as /edit-requests and
// /payroll/report). Admin-only bulk actions must remain denied.
func TestPartnerRole_DeleteTimesheet(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Partner can delete a timesheet by id (handler guards: pending/rejected + not paid)
	assert.True(t, svc.CanAccess("partner", "/api/v1/timesheets/1", "DELETE"),
		"partner should be allowed DELETE /timesheets/:id (handler enforces status)")

	// The new :id allow must NOT weaken admin-only actions — explicit deny wins (deny-overrides)
	adminOnly := []struct {
		path   string
		method string
	}{
		{"/api/v1/timesheets/bulk-approve", "POST"},
		{"/api/v1/timesheets/bulk-reject", "POST"},
		{"/api/v1/timesheets/reject-unpaid", "POST"},
		{"/api/v1/timesheets/bulk-reset", "POST"},
		{"/api/v1/timesheets/approve-all", "POST"},
		{"/api/v1/timesheets/edit-requests/1", "DELETE"},
	}
	for _, tc := range adminOnly {
		assert.False(t, svc.CanAccess("partner", tc.path, tc.method),
			"partner should be DENIED %s %s (admin-only, deny-overrides)", tc.method, tc.path)
	}

	// Admin retains full delete access (no regression)
	assert.True(t, svc.CanAccess("admin", "/api/v1/timesheets/1", "DELETE"),
		"admin should be allowed DELETE /timesheets/:id")
}

// TestPartnerRole_GetTimesheet asserts the partner can read a single timesheet
// by id (GET /timesheets/:id). The list rule (GET /timesheets) does not cover :id
// under keyMatch2, so without an explicit allow the partner UI (edit-modal detail
// fetch, delete/update pre-fetch) hits 403 — which aborts the operation before the
// React Query cache invalidation runs, leaving stale data on screen. Same gap
// class as DELETE /timesheets/:id. Admin-only actions must remain denied.
func TestPartnerRole_GetTimesheet(t *testing.T) {
	svc := newTestAuthorizationService(t)

	// Partner can read a timesheet by id (read-only)
	assert.True(t, svc.CanAccess("partner", "/api/v1/timesheets/1", "GET"),
		"partner should be allowed GET /timesheets/:id")

	// The :id allow must NOT weaken admin-only actions — explicit deny wins (deny-overrides).
	// bulk-* and edit-requests/* are denied with "*" (all methods), which covers GET too.
	adminOnlyGET := []string{
		"/api/v1/timesheets/bulk-approve",
		"/api/v1/timesheets/bulk-reject",
		"/api/v1/timesheets/reject-unpaid",
		"/api/v1/timesheets/bulk-reset",
		"/api/v1/timesheets/approve-all",
		"/api/v1/timesheets/edit-requests/1",
	}
	for _, path := range adminOnlyGET {
		assert.False(t, svc.CanAccess("partner", path, "GET"),
			"partner should be DENIED GET %s (admin-only, deny-overrides)", path)
	}

	// Admin retains full read access (no regression)
	assert.True(t, svc.CanAccess("admin", "/api/v1/timesheets/1", "GET"),
		"admin should be allowed GET /timesheets/:id")
}
