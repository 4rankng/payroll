package admin

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"api-server/internal/app/services/auth"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fakeZnsRepo returns a fixed page regardless of filters — the handler test
// checks routing, auth and response shape, not query semantics (those are
// pinned by the persistence package tests).
type fakeZnsRepo struct {
	rows  []*domain.ZnsMessage
	total int64
}

func (f *fakeZnsRepo) Record(context.Context, *domain.ZnsMessage) error { return nil }

func (f *fakeZnsRepo) ListAdmin(context.Context, domain.ZnsMessageFilter) ([]*domain.ZnsMessage, int64, error) {
	return f.rows, f.total, nil
}

// buildZnsTestRouter wires the real Casbin authorization middleware (policy
// loaded from configs/) in front of the handler, with a stub auth step that
// injects the caller identity.
func buildZnsTestRouter(t *testing.T, role string, repo domain.ZnsMessageRepository) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "..")
	svc, err := auth.NewAuthorizationService(
		filepath.Join(repoRoot, "configs", "casbin_model.conf"),
		filepath.Join(repoRoot, "configs", "casbin_policy.csv"),
		nil,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	require.NoError(t, err)

	authz := middleware.NewAuthorizationMiddleware(svc, nil, nil, config.OTPConfig{})

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(constants.CtxUserID, uint(1))
		c.Set(constants.CtxUserRole, role)
		c.Next()
	})
	r.Use(authz.Authorize())
	r.GET("/api/v1/admin/zns-notifications", NewZnsMessageHandler(repo).List)
	return r
}

// TestZnsMessageList_ForbiddenForNonAdmin pins AC3: every non-admin role gets
// 403 on the tracking endpoint.
func TestZnsMessageList_ForbiddenForNonAdmin(t *testing.T) {
	for _, role := range []string{"partner", "employee", "adv_partner", "accountant"} {
		r := buildZnsTestRouter(t, role, &fakeZnsRepo{})
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/zns-notifications", nil)
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code, "role %s must be forbidden", role)
	}
}

// TestZnsMessageList_AdminGetsPage pins AC1/AC5: admin receives the tracking
// rows with correct pagination totals.
func TestZnsMessageList_AdminGetsPage(t *testing.T) {
	repo := &fakeZnsRepo{
		total: 3,
		rows: []*domain.ZnsMessage{
			{
				ID: 3, CreatedAt: time.Date(2026, 10, 28, 9, 30, 0, 0, time.Local),
				Feature: "salary_notification", TemplateID: "tpl-a", Phone: "0912345678",
				Status: domain.ZnsStatusSent, ProviderMsgID: "zns-1",
			},
			{
				ID: 2, CreatedAt: time.Date(2026, 10, 28, 8, 0, 0, 0, time.Local),
				Feature: "password_reset", TemplateID: "tpl-b", Phone: "0987654321",
				Status: domain.ZnsStatusFailed, ErrorMsg: "Phone has no linked Zalo account",
				ProviderCode: -118,
			},
		},
	}
	r := buildZnsTestRouter(t, "admin", repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/zns-notifications?page=1&pageSize=2", nil)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	require.Contains(t, body, `"totalRecords":3`)
	require.Contains(t, body, `"salary_notification"`)
	require.Contains(t, body, `"zns-1"`)
	require.Contains(t, body, "Phone has no linked Zalo account")
}
