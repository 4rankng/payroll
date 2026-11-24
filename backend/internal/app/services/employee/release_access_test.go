package employee

import (
	"context"
	"testing"

	"api-server/internal/domain"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// Owner ruling 2026-11-24: self-release of employee management is instant —
// no approval step, idempotent, audited.

// fakeEmployeeReleaseRepo is an in-memory EmployeeUserRepository covering the
// release path (Get + soft delete).
type fakeEmployeeReleaseRepo struct {
	domain.EmployeeUserRepository
	rows    map[[2]uint]*domain.EmployeeUser
	deleted [][2]uint
}

func (r *fakeEmployeeReleaseRepo) GetByEmployeeAndUser(_ context.Context, employeeID, userID uint) (*domain.EmployeeUser, error) {
	return r.rows[[2]uint{employeeID, userID}], nil
}

func (r *fakeEmployeeReleaseRepo) Delete(_ context.Context, employeeID, userID uint) error {
	key := [2]uint{employeeID, userID}
	delete(r.rows, key)
	r.deleted = append(r.deleted, key)
	return nil
}

func TestRequestEmployeeReleaseDropsRelationAndAudits(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := &fakeEmployeeReleaseRepo{rows: map[[2]uint]*domain.EmployeeUser{
		{31, 44}: {EmployeeID: 31, UserID: 44, GrantedBy: 44},
	}}
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)
	userRepo.EXPECT().GetByID(gomock.Any(), uint(44)).Return(&domain.User{ID: 44, Fullname: "Quản lý B", Role: domain.RolePartner}, nil)

	alreadyReleased, err := service.RequestEmployeeRelease(context.Background(), 31, 44, string(domain.RolePartner))
	require.NoError(t, err)
	require.False(t, alreadyReleased)

	require.Equal(t, [][2]uint{{31, 44}}, employeeUserRepo.deleted, "relation row must be soft-deleted")
	require.Len(t, events.events, 1, "EmployeeAccessRevokedEvent must publish for audit")
}

func TestRequestEmployeeReleaseIdempotentWithoutRelation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := &fakeEmployeeReleaseRepo{rows: map[[2]uint]*domain.EmployeeUser{}}
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)

	alreadyReleased, err := service.RequestEmployeeRelease(context.Background(), 31, 44, string(domain.RolePartner))
	require.NoError(t, err)
	require.True(t, alreadyReleased)
	require.Empty(t, employeeUserRepo.deleted)
	require.Empty(t, events.events)
}

func TestRequestEmployeeReleaseCreatorForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := &fakeEmployeeReleaseRepo{rows: map[[2]uint]*domain.EmployeeUser{}}
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	// Partner 9 created the employee; their management is intrinsic.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)

	_, err := service.RequestEmployeeRelease(context.Background(), 31, 9, string(domain.RolePartner))
	require.Error(t, err)
	require.Empty(t, employeeUserRepo.deleted)
}

func TestRequestEmployeeReleaseRejectsNonPartner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := &fakeEmployeeReleaseRepo{rows: map[[2]uint]*domain.EmployeeUser{
		{31, 44}: {EmployeeID: 31, UserID: 44},
	}}
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)

	_, err := service.RequestEmployeeRelease(context.Background(), 31, 44, string(domain.RoleAdmin))
	require.Error(t, err, "admins revoke via the explicit path, not self-release")
	require.Empty(t, employeeUserRepo.deleted)
}
