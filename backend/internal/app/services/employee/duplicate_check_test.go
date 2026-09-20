package employee

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestCheckDuplicatesByCCCD(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{
		EmployeeRepo:        employeeRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
	}

	email := "nguyen.van.a@gmail.com"
	created := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	employee := &domain.Employee{
		ID: 31, Fullname: "Nguyễn Văn A", CCCD: "031086019743",
		Mobile: "0909123456", Email: &email, CreatedAt: created,
	}
	employee.Creator = domain.User{ID: 9, Fullname: "Quản lý A"}

	employeeRepo.EXPECT().GetByCCCD(gomock.Any(), "031086019743").Return(employee, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(31)).Return([]*domain.ProjectEmployee{
		{ProjectID: 5, LastDate: nil, Project: domain.Project{ID: 5, Name: "Sản xuất xà phòng"}},
		{ProjectID: 6, LastDate: &created, Project: domain.Project{ID: 6, Name: "Đã kết thúc"}},
	}, nil)

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{CCCD: "031086019743"})
	require.NoError(t, err)
	require.Len(t, matches, 1)

	m := matches[0]
	require.Equal(t, uint(31), m.ID)
	require.Equal(t, "Nguyễn Văn A", m.Fullname)
	require.Equal(t, "031*****9743", m.CCCDMasked)
	require.Empty(t, m.MobileMasked, "mobile must not be revealed when matched via CCCD only")
	require.Empty(t, m.EmailMasked, "email must not be revealed when matched via CCCD only")
	require.Equal(t, []string{"cccd"}, m.MatchedOn)
	require.Equal(t, []string{"Sản xuất xà phòng"}, m.CurrentProjectNames, "ended assignments excluded")
	require.Equal(t, "Quản lý A", m.CreatedByName)
}

func TestCheckDuplicatesByMobileNormalizesPlus84(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{EmployeeRepo: employeeRepo, ProjectEmployeeRepo: projectEmployeeRepo}

	employee := &domain.Employee{ID: 32, Fullname: "Trần Thị B", Mobile: "0909123456"}

	// Shared lookup order: the canonical domestic form is asked first, so the
	// typed +84 value must reach the domestic row.
	employeeRepo.EXPECT().ListByMobile(gomock.Any(), "0909123456").Return([]*domain.Employee{employee}, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(32)).Return(nil, nil)

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{Mobile: "+84909123456"})
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, "******3456", matches[0].MobileMasked)
	require.Empty(t, matches[0].CCCDMasked, "CCCD must not be revealed when matched via mobile only")
	require.Equal(t, []string{"mobile"}, matches[0].MatchedOn)
}

func TestCheckDuplicatesByMobileFallsBackToTheStoredRawForm(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{EmployeeRepo: employeeRepo, ProjectEmployeeRepo: projectEmployeeRepo}

	employee := &domain.Employee{ID: 35, Fullname: "Vũ Thị E", Mobile: "84909123456"}

	// The row was stored before normalization (84 prefix kept), so the canonical
	// form misses and the raw typed value has to find it.
	employeeRepo.EXPECT().ListByMobile(gomock.Any(), "0909123456").Return([]*domain.Employee{}, nil)
	employeeRepo.EXPECT().ListByMobile(gomock.Any(), "84909123456").Return([]*domain.Employee{employee}, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(35)).Return(nil, nil)

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{Mobile: "84909123456"})
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, uint(35), matches[0].ID)
	require.Equal(t, []string{"mobile"}, matches[0].MatchedOn)
}

func TestCheckDuplicatesReportsEveryEmployeeOnASharedMobile(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{EmployeeRepo: employeeRepo, ProjectEmployeeRepo: projectEmployeeRepo}

	first := &domain.Employee{ID: 41, Fullname: "Lò Thị Ánh Tuyết", Mobile: "0374990377"}
	second := &domain.Employee{ID: 42, Fullname: "Lò Thị Luân", Mobile: "0374990377"}

	employeeRepo.EXPECT().ListByMobile(gomock.Any(), "0374990377").Return([]*domain.Employee{first, second}, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(41)).Return(nil, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(42)).Return(nil, nil)

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{Mobile: "0374990377"})
	require.NoError(t, err)
	require.Len(t, matches, 2, "both employees sharing the number must be reported")
	require.Equal(t, uint(41), matches[0].ID)
	require.Equal(t, uint(42), matches[1].ID)
}

func TestCheckDuplicatesByEmailLowercased(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{EmployeeRepo: employeeRepo, ProjectEmployeeRepo: projectEmployeeRepo}

	email := "contact@tingting.vip"
	employee := &domain.Employee{ID: 33, Fullname: "Lê Văn C", Email: &email}

	employeeRepo.EXPECT().GetByEmail(gomock.Any(), "contact@tingting.vip").Return(employee, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(33)).Return(nil, nil)

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{Email: "  Contact@TingTing.vip "})
	require.NoError(t, err)
	require.Len(t, matches, 1)
	require.Equal(t, "c******@tingting.vip", matches[0].EmailMasked)
	require.Equal(t, []string{"email"}, matches[0].MatchedOn)
}

func TestCheckDuplicatesMergesSameEmployee(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	projectEmployeeRepo := mocks.NewMockProjectEmployeeRepository(ctrl)
	service := &EmployeeService{EmployeeRepo: employeeRepo, ProjectEmployeeRepo: projectEmployeeRepo}

	employee := &domain.Employee{ID: 34, Fullname: "Phạm D", CCCD: "031086019744", Mobile: "0909123457"}

	employeeRepo.EXPECT().GetByCCCD(gomock.Any(), "031086019744").Return(employee, nil)
	employeeRepo.EXPECT().ListByMobile(gomock.Any(), "0909123457").Return([]*domain.Employee{employee}, nil)
	projectEmployeeRepo.EXPECT().GetByEmployee(gomock.Any(), uint(34)).Return(nil, nil)

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{CCCD: "031086019744", Mobile: "0909123457"})
	require.NoError(t, err)
	require.Len(t, matches, 1, "same employee matched via two identifiers must merge into one card")
	require.Equal(t, []string{"cccd", "mobile"}, matches[0].MatchedOn)
	require.Equal(t, "031*****9744", matches[0].CCCDMasked)
	require.Equal(t, "******3457", matches[0].MobileMasked)
}

func TestCheckDuplicatesNoMatchReturnsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	employeeRepo := mocks.NewMockEmployeeRepository(ctrl)
	service := &EmployeeService{EmployeeRepo: employeeRepo}

	employeeRepo.EXPECT().GetByCCCD(gomock.Any(), "unknown").Return(nil, domain.NewNotFoundError("not found"))

	matches, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{CCCD: "unknown"})
	require.NoError(t, err)
	require.Empty(t, matches)
}

func TestCheckDuplicatesRequiresIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	service := &EmployeeService{EmployeeRepo: mocks.NewMockEmployeeRepository(ctrl)}

	_, err := service.CheckDuplicates(context.Background(), DuplicateCheckParams{})
	require.Error(t, err)
}
