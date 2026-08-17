package employee

import (
	"context"
	"testing"

	"api-server/internal/domain"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// fakeEmployeeUserRepo is an in-memory EmployeeUserRepository for permission tests.
type fakeEmployeeUserRepo struct {
	domain.EmployeeUserRepository
	rows map[[2]uint]*domain.EmployeeUser
}

func newFakeEmployeeUserRepo() *fakeEmployeeUserRepo {
	return &fakeEmployeeUserRepo{rows: map[[2]uint]*domain.EmployeeUser{}}
}

func (r *fakeEmployeeUserRepo) GetByEmployeeAndUser(_ context.Context, employeeID, userID uint) (*domain.EmployeeUser, error) {
	return r.rows[[2]uint{employeeID, userID}], nil
}

func (r *fakeEmployeeUserRepo) Create(_ context.Context, employeeUser *domain.EmployeeUser) error {
	r.rows[[2]uint{employeeUser.EmployeeID, employeeUser.UserID}] = employeeUser
	return nil
}

func TestRequestEmployeeAccessHappyPathCreatesRow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	userRepo := mocks.NewMockUserRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	events := &recordingEventBus{}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		eventBus:         events,
	}

	// Employee created by partner A (user 9); partner B (user 44) claims it.
	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, Fullname: "Nguyễn Văn A", CreatedBy: 9}, nil)
	userRepo.EXPECT().GetByID(gomock.Any(), uint(44)).Return(&domain.User{ID: 44, Fullname: "Quản lý B", Role: domain.RolePartner}, nil)

	alreadyManaged, err := service.RequestEmployeeAccess(context.Background(), 31, 44, string(domain.RolePartner))
	require.NoError(t, err)
	require.False(t, alreadyManaged)

	row := employeeUserRepo.rows[[2]uint{31, 44}]
	require.NotNil(t, row, "employee_users row must be created")
	require.Equal(t, uint(44), row.UserID)
	require.Equal(t, uint(44), row.GrantedBy, "self-grant: requester is both grantee and granter")

	require.Len(t, events.events, 1, "EmployeeAccessGrantedEvent must publish for audit")
}

func TestRequestEmployeeAccessIdempotent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()
	employeeUserRepo.rows[[2]uint{31, 44}] = &domain.EmployeeUser{EmployeeID: 31, UserID: 44}

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		eventBus:         discardEventBus{},
	}

	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, CreatedBy: 9}, nil)

	alreadyManaged, err := service.RequestEmployeeAccess(context.Background(), 31, 44, string(domain.RolePartner))
	require.NoError(t, err)
	require.True(t, alreadyManaged)
	require.Len(t, employeeUserRepo.rows, 1, "no duplicate row")
}

func TestRequestEmployeeAccessByCreatorIsNoOp(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	employeeUserRepo := newFakeEmployeeUserRepo()

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		eventBus:         discardEventBus{},
	}

	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, CreatedBy: 44}, nil)

	alreadyManaged, err := service.RequestEmployeeAccess(context.Background(), 31, 44, string(domain.RolePartner))
	require.NoError(t, err)
	require.True(t, alreadyManaged)
	require.Empty(t, employeeUserRepo.rows, "creator needs no row")
}

func TestRequestEmployeeAccessRejectsNonPartner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)

	service := &EmployeePermissionService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: newFakeEmployeeUserRepo(),
		eventBus:         discardEventBus{},
	}

	employeeRepo.EXPECT().GetByID(gomock.Any(), uint(31)).Return(&domain.Employee{ID: 31, CreatedBy: 9}, nil)

	_, err := service.RequestEmployeeAccess(context.Background(), 31, 44, string(domain.RoleAdmin))
	require.Error(t, err, "admins do not claim via self-service")
}
