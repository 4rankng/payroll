package handlers

import (
	"bytes"
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"api-server/internal/app/services/user"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

// exportUserRepo is a domain.UserRepository that serves a fixed user page for
// the export path and panics on anything else.
type exportUserRepo struct {
	rows  []*domain.User
	total int64
}

func (r *exportUserRepo) ListWithFilters(_ context.Context, _ *domain.UserRole, _ string, _ []uint, _ bool, _ string, _ string, _, _ int) ([]*domain.User, int64, error) {
	return r.rows, r.total, nil
}

func (r *exportUserRepo) Create(context.Context, *domain.User) error { panic("not impl") }
func (r *exportUserRepo) GetByID(context.Context, uint) (*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetByIDs(context.Context, []uint) (map[uint]*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetByUsername(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetByUsernameIncludingDeleted(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetUsernamesByPrefix(context.Context, string) ([]string, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetByCCCD(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetByMobile(context.Context, string) (*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) ExistsByUsername(context.Context, string) (bool, error) {
	panic("not impl")
}
func (r *exportUserRepo) Update(context.Context, *domain.User) error { panic("not impl") }
func (r *exportUserRepo) UpdateLastLogin(context.Context, uint, time.Time) error {
	panic("not impl")
}
func (r *exportUserRepo) UpdatePasswordAndInvalidateSessions(context.Context, uint, string, time.Time) error {
	panic("not impl")
}
func (r *exportUserRepo) UpdateOTPLockout(context.Context, uint, int, *time.Time) error {
	panic("not impl")
}
func (r *exportUserRepo) UpdateTokensInvalidBefore(context.Context, uint, time.Time) error {
	panic("not impl")
}
func (r *exportUserRepo) Delete(context.Context, uint) error  { panic("not impl") }
func (r *exportUserRepo) Restore(context.Context, uint) error { panic("not impl") }
func (r *exportUserRepo) List(context.Context, int, int) ([]*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) ListByRole(context.Context, domain.UserRole) ([]*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) Count(context.Context) (int64, error) { panic("not impl") }
func (r *exportUserRepo) CountByRole(context.Context, domain.UserRole) (int64, error) {
	panic("not impl")
}
func (r *exportUserRepo) CountRecentLogins(context.Context, time.Time) (int64, error) {
	panic("not impl")
}
func (r *exportUserRepo) CountActiveEmployeesBySchedule(context.Context, time.Time, time.Time) (int, int, int, error) {
	panic("not impl")
}
func (r *exportUserRepo) CountActiveEmployeesByProject(context.Context, time.Time, time.Time) ([]*domain.ProjectActiveCount, error) {
	panic("not impl")
}
func (r *exportUserRepo) GetActiveEmployeesBySchedule(context.Context, time.Time, time.Time, string) ([]*domain.ActiveEmployeeUser, error) {
	panic("not impl")
}
func (r *exportUserRepo) FindUsersWithNullLastLogin(context.Context, int, int) ([]*domain.User, error) {
	panic("not impl")
}
func (r *exportUserRepo) CountUsersWithNullLastLogin(context.Context) (int64, error) {
	panic("not impl")
}

// auditRecorder captures LogFileExport calls for assertions.
type auditRecorder struct {
	dataType    string
	recordCount int
	fileName    string
	calls       int
}

func (a *auditRecorder) LogFileExport(_ context.Context, dataType string, recordCount int, fileName string) error {
	a.dataType = dataType
	a.recordCount = recordCount
	a.fileName = fileName
	a.calls++
	return nil
}

// TestExportUsers pins the user-list export path: the styled workbook carries
// the table's columns (role label + activity status included), the download
// follows the repository filename convention, and the export is audited.
func TestExportUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	createdAt := time.Date(2026, 9, 27, 9, 9, 45, 0, time.UTC)
	lastLogin := time.Date(2026, 10, 12, 14, 41, 4, 0, time.UTC)
	opsEmail := "ops@tinghire.example"
	newEmail := "new@example.com"
	rows := []*domain.User{
		{
			ID:        3,
			Username:  "tinghire.bot",
			Fullname:  "TingHire",
			Email:     &opsEmail,
			Role:      domain.RolePartner,
			LastLogin: &lastLogin,
			CreatedAt: createdAt,
		},
		{
			ID:        4,
			Username:  "new.user",
			Fullname:  "Người Mới",
			Email:     &newEmail,
			Role:      domain.RoleEmployee,
			CreatedAt: createdAt,
		},
	}
	repo := &exportUserRepo{rows: rows, total: 2}
	svc := user.NewUserService(repo, nil, nil, "secret", "salt", clock.NewFake(createdAt))
	audit := &auditRecorder{}
	h := NewUserHandler(svc, nil, audit)

	r := gin.New()
	r.GET("/users/export", h.ExportUsers)
	req := httptest.NewRequest("GET", "/users/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, 200, rec.Code)
	disp := rec.Header().Get("Content-Disposition")
	assert.Contains(t, disp, "attachment; filename=users_export_")
	assert.Contains(t, disp, ".xlsx")

	f, err := excelize.OpenReader(bytes.NewReader(rec.Body.Bytes()))
	assert.NoError(t, err)
	defer func() { _ = f.Close() }()

	sheet, err := f.GetRows("Users")
	assert.NoError(t, err)
	assert.Equal(t,
		[]string{"STT", "Họ và tên", "Username", "Email", "Vai trò", "Hoạt động cuối", "Ngày tạo"},
		sheet[0],
	)

	assert.Equal(t, "1", sheet[1][0])
	assert.Equal(t, "TingHire", sheet[1][1])
	assert.Equal(t, "tinghire.bot", sheet[1][2])
	assert.Equal(t, "ops@tinghire.example", sheet[1][3])
	assert.Equal(t, "Quản lý", sheet[1][4])
	assert.Contains(t, sheet[1][5], "12/10/2026")

	assert.Equal(t, "Người Mới", sheet[2][1])
	assert.Equal(t, "Nhân viên", sheet[2][4])
	assert.Equal(t, "Chưa đăng nhập", sheet[2][5])

	assert.Eventually(t, func() bool { return audit.calls == 1 }, time.Second, 5*time.Millisecond)
	assert.Equal(t, "users", audit.dataType)
	assert.Equal(t, 2, audit.recordCount)
	assert.Contains(t, audit.fileName, "users_export_")
}
