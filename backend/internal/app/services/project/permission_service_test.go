package project

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PAYR-30 guard matrix for project access grant/revoke.
//
// Permission model under test:
//   - admin grants/revokes on any project (200), unchanged for other roles;
//   - partner-admin (project creator or holder of a project_users grant) may
//     grant/revoke only within their own project (403 elsewhere);
//   - a partner-admin may only grant to partner users — granting to a
//     company-side role (admin/accountant) is an elevation attempt (403);
//   - only admin may revoke the project creator (partner-admin → 403).

const (
	ptCreatorID = uint(10) // partner, created the project
	ptManagerID = uint(20) // partner, holds a project_users grant
	ptTargetID  = uint(30) // partner, plain grant target
	admUserID   = uint(40) // admin-role user (elevation target)
	empUserID   = uint(50) // employee-role user (invalid target)

	testProjectID = uint(1)
)

// --- minimal fakes: embed the repo interface so unexpected calls panic ---

type stubProjectRepo struct {
	domain.ProjectRepository
	project *domain.Project
}

func (s *stubProjectRepo) GetByID(_ context.Context, id uint) (*domain.Project, error) {
	if s.project == nil || s.project.ID != id {
		return nil, domain.NewNotFoundError("project not found")
	}
	return s.project, nil
}

func (s *stubProjectRepo) Update(_ context.Context, project *domain.Project) error {
	s.project = project
	return nil
}

type stubProjectUserRepo struct {
	domain.ProjectUserRepository
	active  map[uint]bool // userID -> has an active project_users row
	created []*domain.ProjectUser
	deleted []uint
}

func (s *stubProjectUserRepo) HasAccess(_ context.Context, _ uint, userID uint) (bool, error) {
	return s.active[userID], nil
}

func (s *stubProjectUserRepo) GetByProjectAndUser(_ context.Context, projectID, userID uint) (*domain.ProjectUser, error) {
	if s.active[userID] {
		return &domain.ProjectUser{ProjectID: projectID, UserID: userID}, nil
	}
	return nil, nil
}

func (s *stubProjectUserRepo) Create(_ context.Context, projectUser *domain.ProjectUser) error {
	s.created = append(s.created, projectUser)
	s.active[projectUser.UserID] = true
	return nil
}

func (s *stubProjectUserRepo) Delete(_ context.Context, _ uint, userID uint) error {
	s.deleted = append(s.deleted, userID)
	delete(s.active, userID)
	return nil
}

func (s *stubProjectUserRepo) GetProjectUsers(_ context.Context, projectID uint) ([]*domain.ProjectUser, error) {
	var users []*domain.ProjectUser
	for id := range s.active {
		users = append(users, &domain.ProjectUser{ProjectID: projectID, UserID: id})
	}
	return users, nil
}

type stubProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository
}

func (s *stubProjectEmployeeRepo) GetByProject(_ context.Context, _ uint) ([]*domain.ProjectEmployee, error) {
	return nil, nil
}

type stubUserRepo struct {
	domain.UserRepository
	users map[uint]*domain.User
}

func (s *stubUserRepo) GetByID(_ context.Context, id uint) (*domain.User, error) {
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return nil, domain.NewNotFoundError("user not found")
}

func newTestPermissionService(activeManagers ...uint) (*ProjectPermissionService, *stubProjectUserRepo) {
	active := map[uint]bool{}
	for _, id := range activeManagers {
		active[id] = true
	}
	puRepo := &stubProjectUserRepo{active: active}
	svc := NewProjectPermissionService(
		&stubProjectRepo{project: &domain.Project{ID: testProjectID, CreatedBy: ptCreatorID}},
		puRepo,
		&stubProjectEmployeeRepo{},
		nil,
		&stubUserRepo{users: map[uint]*domain.User{
			ptTargetID: {ID: ptTargetID, Role: domain.RolePartner},
			admUserID:  {ID: admUserID, Role: domain.RoleAdmin},
			empUserID:  {ID: empUserID, Role: domain.RoleEmployee},
		}},
		nil,
	)
	return svc, puRepo
}

func requireDomainErrorType(t *testing.T, err error, typ string) {
	t.Helper()
	var de *domain.DomainError
	require.ErrorAs(t, err, &de)
	assert.Equal(t, typ, de.Type)
}

// newOwnerRevocationService exposes the project stub so revocation tests can
// assert on OwnerRevokedAt, and seeds creator/manager users for list checks.
func newOwnerRevocationService(ownerRevoked bool, activeManagers ...uint) (*ProjectPermissionService, *stubProjectRepo, *stubProjectUserRepo) {
	project := &domain.Project{ID: testProjectID, CreatedBy: ptCreatorID}
	if ownerRevoked {
		now := time.Now()
		project.OwnerRevokedAt = &now
	}
	projRepo := &stubProjectRepo{project: project}
	active := map[uint]bool{}
	for _, id := range activeManagers {
		active[id] = true
	}
	puRepo := &stubProjectUserRepo{active: active}
	svc := NewProjectPermissionService(
		projRepo,
		puRepo,
		&stubProjectEmployeeRepo{},
		nil,
		&stubUserRepo{users: map[uint]*domain.User{
			ptCreatorID: {ID: ptCreatorID, Role: domain.RolePartner},
			ptManagerID: {ID: ptManagerID, Role: domain.RolePartner},
			ptTargetID:  {ID: ptTargetID, Role: domain.RolePartner},
			admUserID:   {ID: admUserID, Role: domain.RoleAdmin},
		}},
		nil,
	)
	return svc, projRepo, puRepo
}

// --- grant ---

func TestGrantProjectAccess_PartnerManager_SameProject_Allowed(t *testing.T) {
	svc, puRepo := newTestPermissionService(ptManagerID)

	err := svc.GrantProjectAccess(context.Background(), testProjectID, ptTargetID, ptManagerID, string(domain.RolePartner))

	require.NoError(t, err)
	require.Len(t, puRepo.created, 1)
	assert.Equal(t, ptTargetID, puRepo.created[0].UserID)
}

func TestGrantProjectAccess_PartnerManager_OtherProject_Forbidden(t *testing.T) {
	svc, puRepo := newTestPermissionService() // no manager grant on this project

	err := svc.GrantProjectAccess(context.Background(), testProjectID, ptTargetID, ptManagerID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.created)
}

func TestGrantProjectAccess_PartnerCreator_ElevationToAdminTarget_Forbidden(t *testing.T) {
	svc, puRepo := newTestPermissionService()

	err := svc.GrantProjectAccess(context.Background(), testProjectID, admUserID, ptCreatorID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.created)
}

func TestGrantProjectAccess_PartnerCreator_ElevationToEmployeeTarget_Forbidden(t *testing.T) {
	svc, puRepo := newTestPermissionService()

	err := svc.GrantProjectAccess(context.Background(), testProjectID, empUserID, ptCreatorID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.created)
}

func TestGrantProjectAccess_Admin_AnyProject_Allowed(t *testing.T) {
	svc, puRepo := newTestPermissionService()

	err := svc.GrantProjectAccess(context.Background(), testProjectID, ptTargetID, admUserID, string(domain.RoleAdmin))

	require.NoError(t, err)
	require.Len(t, puRepo.created, 1)
}

func TestGrantProjectAccess_Admin_NonPartnerTarget_Validation(t *testing.T) {
	svc, puRepo := newTestPermissionService()

	err := svc.GrantProjectAccess(context.Background(), testProjectID, empUserID, admUserID, string(domain.RoleAdmin))

	requireDomainErrorType(t, err, "VALIDATION_ERROR")
	assert.Empty(t, puRepo.created)
}

// --- revoke ---

func TestRevokeProjectAccess_PartnerManager_SameProject_Allowed(t *testing.T) {
	svc, puRepo := newTestPermissionService(ptManagerID, ptTargetID)

	err := svc.RevokeProjectAccess(context.Background(), testProjectID, ptTargetID, ptManagerID, string(domain.RolePartner))

	require.NoError(t, err)
	assert.Equal(t, []uint{ptTargetID}, puRepo.deleted)
}

func TestRevokeProjectAccess_PartnerManager_OtherProject_Forbidden(t *testing.T) {
	svc, puRepo := newTestPermissionService(ptTargetID)

	err := svc.RevokeProjectAccess(context.Background(), testProjectID, ptTargetID, ptManagerID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.deleted)
}

func TestRevokeProjectAccess_PartnerCreator_CreatorTarget_Forbidden(t *testing.T) {
	svc, puRepo := newTestPermissionService()

	err := svc.RevokeProjectAccess(context.Background(), testProjectID, ptCreatorID, ptCreatorID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.deleted)
}

func TestRevokeProjectAccess_PartnerManager_CreatorTarget_Forbidden(t *testing.T) {
	svc, puRepo := newTestPermissionService(ptManagerID)

	err := svc.RevokeProjectAccess(context.Background(), testProjectID, ptCreatorID, ptManagerID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.deleted)
}

func TestRevokeProjectAccess_Admin_AnyProject_Allowed(t *testing.T) {
	svc, puRepo := newTestPermissionService(ptTargetID)

	err := svc.RevokeProjectAccess(context.Background(), testProjectID, ptTargetID, admUserID, string(domain.RoleAdmin))

	require.NoError(t, err)
	assert.Equal(t, []uint{ptTargetID}, puRepo.deleted)
}

// --- creator revocation semantics (PAYR-30) ---

func TestRevokeProjectAccess_Admin_CreatorTarget_RevokesOwnerPrivilege(t *testing.T) {
	svc, projRepo, puRepo := newOwnerRevocationService(false, ptCreatorID)

	err := svc.RevokeProjectAccess(context.Background(), testProjectID, ptCreatorID, admUserID, string(domain.RoleAdmin))

	require.NoError(t, err)
	require.NotNil(t, projRepo.project.OwnerRevokedAt, "owner privilege must be revoked")
	assert.Equal(t, []uint{ptCreatorID}, puRepo.deleted)
}

func TestGetProjectUsers_OmitsRevokedOwner(t *testing.T) {
	svc, _, _ := newOwnerRevocationService(true, ptManagerID)

	users, err := svc.GetProjectUsers(context.Background(), testProjectID)

	require.NoError(t, err)
	var ids []uint
	for _, u := range users {
		ids = append(ids, u.UserID)
	}
	assert.NotContains(t, ids, ptCreatorID)
	assert.Contains(t, ids, ptManagerID)
}

func TestCanUserAccessProject_RevokedOwner_WithoutGrant_NoAccess(t *testing.T) {
	svc, _, _ := newOwnerRevocationService(true)

	hasAccess, err := svc.CanUserAccessProject(context.Background(), testProjectID, ptCreatorID)

	require.NoError(t, err)
	assert.False(t, hasAccess)
}

func TestCanUserAccessProject_RevokedOwner_WithGrant_Access(t *testing.T) {
	svc, _, _ := newOwnerRevocationService(true, ptCreatorID)

	hasAccess, err := svc.CanUserAccessProject(context.Background(), testProjectID, ptCreatorID)

	require.NoError(t, err)
	assert.True(t, hasAccess)
}

func TestGrantProjectAccess_RevokedOwner_WithoutGrant_Forbidden(t *testing.T) {
	svc, _, puRepo := newOwnerRevocationService(true)

	err := svc.GrantProjectAccess(context.Background(), testProjectID, ptTargetID, ptCreatorID, string(domain.RolePartner))

	requireDomainErrorType(t, err, "FORBIDDEN")
	assert.Empty(t, puRepo.created)
}
