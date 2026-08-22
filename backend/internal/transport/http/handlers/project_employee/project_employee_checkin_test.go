package project_employee

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	projectservice "api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type checkInProjectRepo struct {
	domain.ProjectRepository
	projects []*domain.Project
	filters  domain.ProjectFilters
}

func (f *checkInProjectRepo) GetByID(_ context.Context, id uint) (*domain.Project, error) {
	for _, project := range f.projects {
		if project.ID == id {
			return project, nil
		}
	}
	return &domain.Project{ID: id, CreatedBy: 999}, nil
}

func (f *checkInProjectRepo) List(_ context.Context, filters domain.ProjectFilters) ([]*domain.Project, error) {
	f.filters = filters
	return f.projects, nil
}

type checkInProjectUserRepo struct {
	domain.ProjectUserRepository
	hasAccess bool
}

func (f *checkInProjectUserRepo) HasAccess(_ context.Context, _, _ uint) (bool, error) {
	return f.hasAccess, nil
}

func newCheckInHandlerForTest(projectRepo *checkInProjectRepo, projectUserRepo *checkInProjectUserRepo) *Handler {
	permissionService := projectservice.NewProjectPermissionService(
		projectRepo, projectUserRepo, nil, nil, nil, nil,
	)
	return &Handler{
		projectService:           &projectservice.ProjectService{ProjectRepo: projectRepo},
		projectPermissionService: permissionService,
		logger:                   slog.Default(),
	}
}

func checkInGinContext(role string, userID uint) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/", nil)
	ctx.Set(constants.CtxUserRole, role)
	ctx.Set(constants.CtxUserID, userID)
	return ctx, recorder
}

func TestRequireCheckInConfigurationAccessRejectsStandardPartner(t *testing.T) {
	handler := newCheckInHandlerForTest(&checkInProjectRepo{}, &checkInProjectUserRepo{})
	ctx, recorder := checkInGinContext(string(domain.RolePartner), 42)

	require.False(t, handler.requireCheckInConfigurationAccess(ctx, 7))
	require.Equal(t, 403, recorder.Code)
}

func TestRequireCheckInConfigurationAccessScopesAdvancePartnerToModifiableProject(t *testing.T) {
	projectRepo := &checkInProjectRepo{projects: []*domain.Project{{ID: 7, CreatedBy: 999}}}
	handler := newCheckInHandlerForTest(projectRepo, &checkInProjectUserRepo{hasAccess: false})
	ctx, recorder := checkInGinContext(string(domain.RoleAdvPartner), 42)

	require.False(t, handler.requireCheckInConfigurationAccess(ctx, 7))
	require.Equal(t, 403, recorder.Code)

	handler = newCheckInHandlerForTest(projectRepo, &checkInProjectUserRepo{hasAccess: true})
	ctx, _ = checkInGinContext(string(domain.RoleAdvPartner), 42)
	require.True(t, handler.requireCheckInConfigurationAccess(ctx, 7))
}

func TestListCheckInConfigurableProjectsReturnsMinimalModifiableProjection(t *testing.T) {
	projectRepo := &checkInProjectRepo{projects: []*domain.Project{
		{ID: 7, Name: "LGD", Code: "LGD", ProjectStatus: domain.ProjectStatusRunning, IsFlexible: true, GeofenceRadiusMeters: 500},
		{ID: 8, Name: "Fixed", Code: "FIX", ProjectStatus: domain.ProjectStatusRunning, IsFlexible: false},
	}}
	handler := newCheckInHandlerForTest(projectRepo, &checkInProjectUserRepo{})
	ctx, recorder := checkInGinContext(string(domain.RoleAdvPartner), 42)

	handler.ListCheckInConfigurableProjects(ctx)

	require.Equal(t, 200, recorder.Code)
	require.NotNil(t, projectRepo.filters.ModifiableBy)
	require.Equal(t, uint(42), *projectRepo.filters.ModifiableBy)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, map[string]any{
		"id":          float64(7),
		"name":        "LGD",
		"code":        "LGD",
		"status":      "active",
		"is_flexible": true,
	}, body.Data[0])
}
