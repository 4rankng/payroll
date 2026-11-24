package project

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// Owner ruling 2026-11-24: "request to manage project without approval" —
// claiming is granting, and self-release is instant. These tests pin the
// self-service semantics on top of the PAYR-30 guard matrix.

type recordingProjectEventBus struct {
	domain.EventBus
	events []domain.DomainEvent
}

func (b *recordingProjectEventBus) Publish(_ context.Context, events ...domain.DomainEvent) error {
	b.events = append(b.events, events...)
	return nil
}
func (*recordingProjectEventBus) Subscribe(string, domain.EventHandler) {}
func (*recordingProjectEventBus) SubscribeAll(domain.EventHandler)      {}

func newSelfServiceService(activeManagers []uint, ownerRevoked bool, bus *recordingProjectEventBus) (*ProjectPermissionService, *stubProjectUserRepo) {
	created := &domain.Project{ID: testProjectID, CreatedBy: ptCreatorID}
	if ownerRevoked {
		now := time.Now()
		created.OwnerRevokedAt = &now
	}
	puRepo := &stubProjectUserRepo{active: map[uint]bool{}}
	for _, id := range activeManagers {
		puRepo.active[id] = true
	}
	svc := NewProjectPermissionService(
		&stubProjectRepo{project: created},
		puRepo,
		&stubProjectEmployeeRepo{},
		nil,
		&stubUserRepo{users: map[uint]*domain.User{
			ptCreatorID: {ID: ptCreatorID, Role: domain.RolePartner},
			ptManagerID: {ID: ptManagerID, Role: domain.RolePartner},
			ptTargetID:  {ID: ptTargetID, Fullname: "Quản lý C", Role: domain.RolePartner},
		}},
		bus,
	)
	return svc, puRepo
}

func TestRequestProjectAccessHappyPathCreatesRow(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService(nil, false, bus)

	alreadyManaged, err := svc.RequestProjectAccess(context.Background(), testProjectID, ptTargetID, string(domain.RolePartner))
	require.NoError(t, err)
	require.False(t, alreadyManaged)

	require.Len(t, puRepo.created, 1, "project_users row must be created")
	require.Equal(t, ptTargetID, puRepo.created[0].UserID)
	require.Equal(t, ptTargetID, puRepo.created[0].GrantedBy, "self-grant: requester is both grantee and granter")

	require.Len(t, bus.events, 1, "ProjectAccessGrantedEvent must publish for audit")
}

func TestRequestProjectAccessIdempotentForGrantHolder(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService([]uint{ptTargetID}, false, bus)

	alreadyManaged, err := svc.RequestProjectAccess(context.Background(), testProjectID, ptTargetID, string(domain.RolePartner))
	require.NoError(t, err)
	require.True(t, alreadyManaged)
	require.Empty(t, puRepo.created, "no duplicate row")
	require.Empty(t, bus.events)
}

func TestRequestProjectAccessCreatorIsNoOp(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService(nil, false, bus)

	alreadyManaged, err := svc.RequestProjectAccess(context.Background(), testProjectID, ptCreatorID, string(domain.RolePartner))
	require.NoError(t, err)
	require.True(t, alreadyManaged)
	require.Empty(t, puRepo.created, "creator needs no row")
}

func TestRequestProjectAccessRevokedCreatorGetsPlainRow(t *testing.T) {
	// A creator whose owner privilege an admin stripped may re-claim as a
	// plain manager — the no-op shortcut must not swallow them.
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService(nil, true, bus)

	alreadyManaged, err := svc.RequestProjectAccess(context.Background(), testProjectID, ptCreatorID, string(domain.RolePartner))
	require.NoError(t, err)
	require.False(t, alreadyManaged)
	require.Len(t, puRepo.created, 1)
}

func TestRequestProjectAccessRejectsNonPartner(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService(nil, false, bus)

	_, err := svc.RequestProjectAccess(context.Background(), testProjectID, admUserID, string(domain.RoleAdmin))
	require.Error(t, err, "admins do not claim via self-service")
	require.Empty(t, puRepo.created)
}

func TestRequestProjectReleaseDropsGrantAndAudits(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService([]uint{ptTargetID}, false, bus)

	alreadyReleased, err := svc.RequestProjectRelease(context.Background(), testProjectID, ptTargetID, string(domain.RolePartner))
	require.NoError(t, err)
	require.False(t, alreadyReleased)
	require.Equal(t, []uint{ptTargetID}, puRepo.deleted)
	require.Len(t, bus.events, 1, "ProjectAccessRevokedEvent must publish for audit")
}

func TestRequestProjectReleaseIdempotentWithoutGrant(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService(nil, false, bus)

	alreadyReleased, err := svc.RequestProjectRelease(context.Background(), testProjectID, ptTargetID, string(domain.RolePartner))
	require.NoError(t, err)
	require.True(t, alreadyReleased)
	require.Empty(t, puRepo.deleted)
	require.Empty(t, bus.events)
}

func TestRequestProjectReleaseCreatorForbidden(t *testing.T) {
	// The intrinsic creator privilege is only strippable by an admin (PAYR-30);
	// a creator "leaving" their own project would orphan it.
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService(nil, false, bus)

	_, err := svc.RequestProjectRelease(context.Background(), testProjectID, ptCreatorID, string(domain.RolePartner))
	require.Error(t, err)
	require.Empty(t, puRepo.deleted)
}

func TestRequestProjectReleaseRejectsNonPartner(t *testing.T) {
	bus := &recordingProjectEventBus{}
	svc, puRepo := newSelfServiceService([]uint{ptTargetID}, false, bus)

	_, err := svc.RequestProjectRelease(context.Background(), testProjectID, ptTargetID, string(domain.RoleAdmin))
	require.Error(t, err, "admins revoke via the explicit path, not self-release")
	require.Empty(t, puRepo.deleted)
}
