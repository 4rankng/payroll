package auth

import (
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// resolveConfigPaths returns absolute paths to the casbin model + policy
// files in repo configs/, regardless of the package dir the test runs in.
func resolveConfigPaths(t *testing.T) (model, policy string) {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller for config path")
	}
	repoRoot := filepath.Join(filepath.Dir(here), "..", "..", "..", "..")
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

		// Users - read + update
		{"/api/v1/users", "GET"},
		{"/api/v1/users/summary", "GET"},
		{"/api/v1/users/1", "GET"},
		{"/api/v1/users/1", "PUT"},

		// Employees - read
		{"/api/v1/employees", "GET"},
		{"/api/v1/employees/1", "GET"},

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
		// User management — create/delete denied
		{"/api/v1/users", "POST"},
		{"/api/v1/users/1", "DELETE"},

		// Employee data
		{"/api/v1/employees/import", "POST"},

		// Projects
		{"/api/v1/projects", "GET"},
		{"/api/v1/projects/1", "PUT"},

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
	assert.False(t, svc.CanAccess("employee", "/api/v1/employees", "GET"))

	// adv_partner can read/import advance payments, manage users, denied on admin/partner surfaces
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/advance-payments", "GET"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/advance-payments/import", "POST"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/users", "GET"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/users/1", "PUT"))
	assert.False(t, svc.CanAccess("adv_partner", "/api/v1/users", "POST"))
	assert.False(t, svc.CanAccess("adv_partner", "/api/v1/advance-payments/1/cancel", "POST"))
	assert.True(t, svc.CanAccess("adv_partner", "/api/v1/employees", "GET"))
}
