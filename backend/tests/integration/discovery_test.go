package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAllPagesFindsRecordsBeyondFirstPage(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages = append(pages, r.URL.Query().Get("page"))
		require.Equal(t, "200", r.URL.Query().Get("pageSize"))
		require.Equal(t, "active", r.URL.Query().Get("status"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"success","data":[{"id":%s}],"pagination":{"page":%s,"totalPages":2}}`, r.URL.Query().Get("page"), r.URL.Query().Get("page"))
	}))
	defer server.Close()

	projects, err := loadAllPages[ProjectResponse](NewAPIClient(server.URL), "/projects?status=active")
	require.NoError(t, err)
	require.Equal(t, []string{"1", "2"}, pages)
	require.Len(t, projects, 2)
	require.Equal(t, uint(2), projects[1].ID)
}

func TestLoadAllPagesRejectsBrokenPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":[],"pagination":{"page":1,"totalPages":2}}`))
	}))
	defer server.Close()
	_, err := loadAllPages[ProjectResponse](NewAPIClient(server.URL), "/projects")
	require.ErrorContains(t, err, "inconsistent pagination")
}

func TestLoadAllPagesRejectsHTTPErrorWithoutErrorEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"database unavailable"}`))
	}))
	defer server.Close()
	_, err := loadAllPages[ProjectResponse](NewAPIClient(server.URL), "/projects")
	require.ErrorContains(t, err, "HTTP 500")
}

func TestTimesheetFlowDoesNotDereferenceMissingDiscovery(t *testing.T) {
	for _, data := range []*TestData{{}, {WeeklyProject: &ProjectResponse{ID: 1}}} {
		runTimesheetExtendedTests(NewAPIClient("http://127.0.0.1:1"), data, NewReporter(), &TestConfig{})
	}
}

func TestImportedAdvanceDiscoveryExcludesSelfCheckInPolicy(t *testing.T) {
	require.False(t, hasImportedAdvanceAssignment([]EmployeeProjectInfo{{PaymentSchedule: "flexible", CheckInEnabled: true}}))
	require.False(t, hasImportedAdvanceAssignment([]EmployeeProjectInfo{{PaymentSchedule: "weekly"}}))
	require.True(t, hasImportedAdvanceAssignment([]EmployeeProjectInfo{{PaymentSchedule: "flexible"}}))
}
