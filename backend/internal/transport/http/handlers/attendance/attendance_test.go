package attendance

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	attendanceSvc "api-server/internal/app/services/attendance"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"

	"github.com/gin-gonic/gin"
)

type fakeCheckoutTransactionManager struct {
	domain.TransactionManager
}

func (f *fakeCheckoutTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (f *fakeCheckoutTransactionManager) WithTransactionResult(ctx context.Context, fn func(context.Context) (any, error)) (any, error) {
	return fn(ctx)
}

type fakeCheckoutAttendanceRepo struct {
	domain.AttendanceRepository
	byDay map[string]*domain.Attendance
}

func (f *fakeCheckoutAttendanceRepo) GetByEmployeeAndDate(_ context.Context, _ uint, date time.Time) (*domain.Attendance, error) {
	return f.byDay[date.Format("2006-01-02")], nil
}

type fakeCheckoutEmployeeRepo struct {
	domain.EmployeeRepository
	employee *domain.Employee
}

func (f *fakeCheckoutEmployeeRepo) GetByUserID(_ context.Context, _ uint) (*domain.Employee, error) {
	return f.employee, nil
}

type fakeCheckoutFailedAttemptRepo struct {
	domain.AttendanceFailedAttemptRepository
	created chan *domain.AttendanceFailedAttempt
}

func (f *fakeCheckoutFailedAttemptRepo) Create(_ context.Context, attempt *domain.AttendanceFailedAttempt) error {
	f.created <- attempt
	return nil
}

type fakeCheckoutProjectEmployeeRepo struct {
	domain.ProjectEmployeeRepository
	assignment *domain.ProjectEmployee
}

func (f *fakeCheckoutProjectEmployeeRepo) GetActiveAssignmentByProjectAndEmployee(_ context.Context, _, _ uint) (*domain.ProjectEmployee, error) {
	return f.assignment, nil
}

type fakeCheckoutPayrateRepo struct {
	domain.PayrateRepository
	payrate *domain.Payrate
}

func (f *fakeCheckoutPayrateRepo) GetActiveByProjectAndDate(_ context.Context, _ uint, _ time.Time) (*domain.Payrate, error) {
	return f.payrate, nil
}

func TestCheckOutFailedAttemptUsesAttendanceProjectID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	loc := clock.DefaultLocation
	now := time.Date(2026, 7, 2, 23, 35, 0, 0, loc)
	today := time.Date(2026, 7, 2, 0, 0, 0, 0, loc)

	attendanceRepo := &fakeCheckoutAttendanceRepo{byDay: map[string]*domain.Attendance{
		today.Format("2006-01-02"): {
			ID:          7,
			EmployeeID:  123,
			ProjectID:   55,
			Date:        today,
			CheckInTime: time.Date(2026, 7, 2, 8, 0, 0, 0, loc),
			CheckInGate: "Cổng chính",
		},
	}}
	failedRepo := &fakeCheckoutFailedAttemptRepo{created: make(chan *domain.AttendanceFailedAttempt, 1)}
	svc := attendanceSvc.NewAttendanceService(
		attendanceRepo,
		&fakeCheckoutProjectEmployeeRepo{assignment: &domain.ProjectEmployee{Position: "Công nhân", CheckInEnabled: true}},
		nil,
		&fakeCheckoutPayrateRepo{payrate: &domain.Payrate{Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`)}},
		nil,
		&fakeCheckoutTransactionManager{},
		nil,
		clock.NewFake(now),
	)
	handler := NewHandler(
		svc,
		&fakeCheckoutEmployeeRepo{employee: &domain.Employee{ID: 123}},
		failedRepo,
		clock.NewFake(now),
		nil,
	)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Set(constants.CtxUserID, uint(99))
	body := bytes.NewBufferString(`{"lat":10.0001,"lng":106.0001,"accuracy":5}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mobile/attendance/check-out", body)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	handler.CheckOut(c)

	select {
	case got := <-failedRepo.created:
		if got.ProjectID != 55 {
			t.Fatalf("failed attempt project_id = %d, want 55", got.ProjectID)
		}
		if got.Lat == nil || got.Lng == nil {
			t.Fatal("expected failed attempt to preserve checkout coordinates")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for failed attempt log")
	}
}
