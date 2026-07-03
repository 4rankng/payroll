package admin

import (
	"bytes"
	"context"
	"log/slog"
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

type fakeFailedAttemptRepo struct {
	domain.AttendanceFailedAttemptRepository
	rows []*domain.AttendanceFailedAttempt
}

func (f *fakeFailedAttemptRepo) List(_ context.Context, _ domain.FailedAttemptFilters) ([]*domain.AttendanceFailedAttempt, error) {
	return f.rows, nil
}

func (f *fakeFailedAttemptRepo) Count(_ context.Context, _ domain.FailedAttemptFilters) (int64, error) {
	return int64(len(f.rows)), nil
}

type fakeProjectRepo struct {
	domain.ProjectRepository
	projects map[uint]*domain.Project
}

func (f *fakeProjectRepo) GetByIDs(_ context.Context, ids []uint) (map[uint]*domain.Project, error) {
	out := make(map[uint]*domain.Project, len(ids))
	for _, id := range ids {
		if p := f.projects[id]; p != nil {
			out[id] = p
		}
	}
	return out, nil
}

type fakeAttendanceRepo struct {
	domain.AttendanceRepository
	byDay   map[string]*domain.Attendance
	created []*domain.Attendance
	nextID  uint
}

func (f *fakeAttendanceRepo) GetByEmployeeAndDate(_ context.Context, _ uint, date time.Time) (*domain.Attendance, error) {
	return f.byDay[date.Format("2006-01-02")], nil
}

func TestNearestCheckpointForAttempt(t *testing.T) {
	lat := 10.0001
	lng := 106.0001
	project := &domain.Project{
		GeofenceRadiusMeters: 100,
		GeofenceGates: []domain.GeofenceGate{
			{Name: "Cổng xa", Lat: 10.01, Lng: 106.01},
			{Name: "Cổng chính", Lat: 10.0002, Lng: 106.0002},
		},
	}

	got := nearestCheckpointForAttempt(&domain.AttendanceFailedAttempt{
		Lat: &lat,
		Lng: &lng,
	}, project)

	if got == nil {
		t.Fatal("nearestCheckpointForAttempt() returned nil")
	}
	if got.name != "Cổng chính" {
		t.Fatalf("nearestCheckpointForAttempt() name = %q, want %q", got.name, "Cổng chính")
	}
	if got.geofenceRadiusMeters != 100 {
		t.Fatalf("nearestCheckpointForAttempt() radius = %d, want %d", got.geofenceRadiusMeters, 100)
	}
	if got.distanceMeters <= 0 || got.distanceMeters >= 20 {
		t.Fatalf("nearestCheckpointForAttempt() distance = %f, want within 0..20m", got.distanceMeters)
	}
}

func TestNearestCheckpointForAttemptMissingData(t *testing.T) {
	lat := 10.0001
	lng := 106.0001

	tests := []struct {
		name    string
		attempt *domain.AttendanceFailedAttempt
		project *domain.Project
	}{
		{
			name:    "missing longitude",
			attempt: &domain.AttendanceFailedAttempt{Lat: &lat},
			project: &domain.Project{GeofenceGates: []domain.GeofenceGate{{Name: "Cổng chính", Lat: 10, Lng: 106}}},
		},
		{
			name:    "missing project",
			attempt: &domain.AttendanceFailedAttempt{Lat: &lat, Lng: &lng},
			project: nil,
		},
		{
			name:    "missing gates",
			attempt: &domain.AttendanceFailedAttempt{Lat: &lat, Lng: &lng},
			project: &domain.Project{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nearestCheckpointForAttempt(tt.attempt, tt.project); got != nil {
				t.Fatalf("nearestCheckpointForAttempt() = %#v, want nil", got)
			}
		})
	}
}

func TestMapAdminAttendanceResponseRejectedIncludesRejectTimeAndNearestGate(t *testing.T) {
	loc := clock.DefaultLocation
	reason := "Đã hết hạn tan ca"
	rejectedAt := time.Date(2026, 7, 2, 21, 0, 0, 0, loc)
	att := &domain.Attendance{
		ID:                 7,
		EmployeeID:         123,
		ProjectID:          55,
		Date:               time.Date(2026, 7, 2, 0, 0, 0, 0, loc),
		CheckInTime:        time.Date(2026, 7, 2, 20, 40, 0, 0, loc),
		CheckInLat:         10.0001,
		CheckInLng:         106.0001,
		CheckInGate:        "Cổng A",
		SalaryRejectReason: &reason,
		UpdatedAt:          rejectedAt,
		Employee:           domain.Employee{Fullname: "Đỗ Thị Thoa"},
		Project: domain.Project{
			Name:                 "LGD",
			GeofenceRadiusMeters: 100,
			GeofenceGates: []domain.GeofenceGate{
				{Name: "Cổng xa", Lat: 10.01, Lng: 106.01},
				{Name: "Cổng A", Lat: 10.0002, Lng: 106.0002},
			},
		},
	}

	got := mapAdminAttendanceResponse(att, rejectedAt)

	if got.Status != string(domain.AttendanceStatusRejected) {
		t.Fatalf("status = %q, want rejected", got.Status)
	}
	if got.RejectedAt == nil || !got.RejectedAt.Equal(rejectedAt) {
		t.Fatalf("rejected_at = %v, want %v", got.RejectedAt, rejectedAt)
	}
	if got.NearestCheckpointName == nil || *got.NearestCheckpointName != "Cổng A" {
		t.Fatalf("nearest checkpoint = %v, want Cổng A", got.NearestCheckpointName)
	}
	if got.NearestCheckpointDistanceMeters == nil || *got.NearestCheckpointDistanceMeters <= 0 {
		t.Fatalf("nearest distance = %v, want > 0", got.NearestCheckpointDistanceMeters)
	}
	if got.GeofenceRadiusMeters == nil || *got.GeofenceRadiusMeters != 100 {
		t.Fatalf("geofence radius = %v, want 100", got.GeofenceRadiusMeters)
	}
	if got.CheckInAccuracy != nil {
		t.Fatalf("check_in_accuracy = %v, want nil", got.CheckInAccuracy)
	}
	if got.NearestCheckpointLat == nil || got.NearestCheckpointLng == nil {
		t.Fatalf("nearest checkpoint coordinates = (%v, %v), want non-nil", got.NearestCheckpointLat, got.NearestCheckpointLng)
	}
}

func TestListFailedAttemptsInfersCheckoutProjectForLegacyRows(t *testing.T) {
	loc := clock.DefaultLocation
	createdAt := time.Date(2026, 7, 2, 23, 35, 0, 0, loc)
	today := time.Date(2026, 7, 2, 0, 0, 0, 0, loc)
	lat := 10.0001
	lng := 106.0001

	failedRepo := &fakeFailedAttemptRepo{rows: []*domain.AttendanceFailedAttempt{{
		ID:             1,
		EmployeeID:     123,
		AttemptType:    "check_out",
		ReasonCategory: "check_out_window",
		ProjectID:      0,
		Lat:            &lat,
		Lng:            &lng,
		CreatedAt:      createdAt,
		Employee:       domain.Employee{Fullname: "Đỗ Thị Thoa"},
	}}}
	project := &domain.Project{
		ID:                   55,
		GeofenceRadiusMeters: 100,
		GeofenceGates: []domain.GeofenceGate{
			{Name: "Cổng chính", Lat: 10.0002, Lng: 106.0002},
		},
	}
	svc := attendanceSvc.NewAttendanceService(
		&fakeAttendanceRepo{byDay: map[string]*domain.Attendance{
			today.Format("2006-01-02"): {
				ID:         7,
				EmployeeID: 123,
				ProjectID:  55,
				Date:       today,
			},
		}},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		clock.NewFake(createdAt),
	)
	handler := &AttendanceHandler{
		attendanceService: svc,
		failedAttemptRepo: failedRepo,
		projectRepo:       &fakeProjectRepo{projects: map[uint]*domain.Project{55: project}},
	}

	got, total, err := handler.listFailedAttempts(context.Background(), domain.FailedAttemptFilters{})
	if err != nil {
		t.Fatalf("listFailedAttempts returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(got) != 1 {
		t.Fatalf("len(response) = %d, want 1", len(got))
	}
	if got[0].NearestCheckpointName == nil || *got[0].NearestCheckpointName != "Cổng chính" {
		t.Fatalf("nearest checkpoint = %v, want Cổng chính", got[0].NearestCheckpointName)
	}
	if got[0].NearestCheckpointDistanceMeters == nil {
		t.Fatal("expected nearest checkpoint distance to be populated")
	}
	if got[0].NearestCheckpointLat == nil || got[0].NearestCheckpointLng == nil {
		t.Fatal("expected nearest checkpoint coordinates to be populated")
	}
}

// --- admin override (record check-in from a device-GPS failure) ---

func (f *fakeFailedAttemptRepo) GetByID(_ context.Context, id uint) (*domain.AttendanceFailedAttempt, error) {
	for _, r := range f.rows {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, domain.NewNotFoundError("không tìm thấy lần thử thất bại")
}

// Update is a no-op: the handler mutates the in-memory row pointer that lives
// in f.rows, so assertions read f.rows directly after the call.
func (f *fakeFailedAttemptRepo) Update(_ context.Context, _ *domain.AttendanceFailedAttempt) error {
	return nil
}

func (f *fakeAttendanceRepo) Create(_ context.Context, attendance *domain.Attendance) error {
	if f.nextID == 0 {
		f.nextID = 100
	}
	f.nextID++
	attendance.ID = f.nextID
	f.created = append(f.created, attendance)
	return nil
}

func (f *fakeProjectRepo) GetByID(_ context.Context, id uint) (*domain.Project, error) {
	if p := f.projects[id]; p != nil {
		return p, nil
	}
	return nil, domain.NewNotFoundError("project not found")
}

type fakeTransactionManager struct{}

func (fakeTransactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (fakeTransactionManager) WithTransactionResult(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	return fn(ctx)
}

func buildOverrideHandler(t *testing.T, failedRows []*domain.AttendanceFailedAttempt, byDay map[string]*domain.Attendance) (*AttendanceHandler, *fakeFailedAttemptRepo, *fakeAttendanceRepo) {
	t.Helper()
	now := time.Date(2026, 7, 3, 9, 0, 0, 0, clock.DefaultLocation)
	attRepo := &fakeAttendanceRepo{byDay: byDay, nextID: 100}
	projectRepo := &fakeProjectRepo{projects: map[uint]*domain.Project{55: {ID: 55, IsFlexible: true}}}
	svc := attendanceSvc.NewAttendanceService(
		attRepo,
		nil, // projectEmployeeRepo unused: the attempt carries project_id 55
		projectRepo,
		nil,
		nil,
		fakeTransactionManager{},
		nil,
		clock.NewFake(now),
	)
	failedRepo := &fakeFailedAttemptRepo{rows: failedRows}
	handler := &AttendanceHandler{
		attendanceService: svc,
		failedAttemptRepo: failedRepo,
		projectRepo:       projectRepo,
		clk:               clock.NewFake(now),
		logger:            slog.Default(),
	}
	return handler, failedRepo, attRepo
}

func callOverride(handler *AttendanceHandler, idStr, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: idStr}}
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(constants.CtxUserID, uint(42))
	handler.AdminOverrideFailedAttempt(c)
	return w
}

func TestAdminOverrideFailedAttempt_GPSFailureCreatesCheckIn(t *testing.T) {
	loc := clock.DefaultLocation
	rows := []*domain.AttendanceFailedAttempt{{
		ID:             1,
		EmployeeID:     7,
		AttemptType:    "check_in",
		ReasonCategory: "gps_timeout",
		ProjectID:      55,
		CreatedAt:      time.Date(2026, 7, 3, 8, 58, 0, 0, loc),
	}}
	handler, failedRepo, attRepo := buildOverrideHandler(t, rows, nil)

	w := callOverride(handler, "1", `{"reason":"Nhân viên ở cổng, điện thoại hỏng GPS"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if !failedRepo.rows[0].IsResolved() {
		t.Fatal("failed attempt should be marked resolved")
	}
	if failedRepo.rows[0].ResolvedBy == nil || *failedRepo.rows[0].ResolvedBy != 42 {
		t.Fatalf("resolved_by = %v, want 42", failedRepo.rows[0].ResolvedBy)
	}
	if len(attRepo.created) != 1 {
		t.Fatalf("created %d attendances, want 1", len(attRepo.created))
	}
	got := attRepo.created[0]
	if got.EmployeeID != 7 || got.ProjectID != 55 {
		t.Fatalf("created attendance employee/project = %d/%d, want 7/55", got.EmployeeID, got.ProjectID)
	}
	if got.CheckInGate == "" {
		t.Fatal("created attendance should carry a manual-origin gate marker")
	}
	// Check-in time reflects when the employee was actually at the gate.
	wantCI := time.Date(2026, 7, 3, 8, 58, 0, 0, loc)
	if !got.CheckInTime.Equal(wantCI) {
		t.Fatalf("check_in_time = %v, want %v", got.CheckInTime, wantCI)
	}
}

func TestAdminOverrideFailedAttempt_GeofenceOutsideIsRejected(t *testing.T) {
	rows := []*domain.AttendanceFailedAttempt{{
		ID: 1, EmployeeID: 7, AttemptType: "check_in",
		ReasonCategory: "geofence_outside", ProjectID: 55,
		CreatedAt: time.Date(2026, 7, 3, 8, 58, 0, 0, clock.DefaultLocation),
	}}
	handler, failedRepo, attRepo := buildOverrideHandler(t, rows, nil)

	w := callOverride(handler, "1", `{"reason":"should not be allowed"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for geofence_outside; body=%s", w.Code, w.Body.String())
	}
	if failedRepo.rows[0].IsResolved() {
		t.Fatal("geofence_outside attempt must not be resolved")
	}
	if len(attRepo.created) != 0 {
		t.Fatalf("created %d attendances, want 0 for rejected override", len(attRepo.created))
	}
}

func TestAdminOverrideFailedAttempt_CheckoutAttemptIsRejected(t *testing.T) {
	rows := []*domain.AttendanceFailedAttempt{{
		ID: 1, EmployeeID: 7, AttemptType: "check_out",
		ReasonCategory: "gps_timeout", ProjectID: 55,
		CreatedAt: time.Date(2026, 7, 3, 8, 58, 0, 0, clock.DefaultLocation),
	}}
	handler, _, attRepo := buildOverrideHandler(t, rows, nil)

	w := callOverride(handler, "1", `{"reason":"x"}`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for check_out attempt", w.Code)
	}
	if len(attRepo.created) != 0 {
		t.Fatalf("created %d attendances, want 0", len(attRepo.created))
	}
}

func TestAdminOverrideFailedAttempt_AlreadyResolvedIsConflict(t *testing.T) {
	resolvedAt := time.Date(2026, 7, 3, 9, 0, 0, 0, clock.DefaultLocation)
	resolvedBy := uint(5)
	rows := []*domain.AttendanceFailedAttempt{{
		ID: 1, EmployeeID: 7, AttemptType: "check_in", ReasonCategory: "gps_unavailable",
		ProjectID: 55, ResolvedAt: &resolvedAt, ResolvedBy: &resolvedBy,
		CreatedAt: time.Date(2026, 7, 3, 8, 58, 0, 0, clock.DefaultLocation),
	}}
	handler, _, attRepo := buildOverrideHandler(t, rows, nil)

	w := callOverride(handler, "1", `{"reason":"again"}`)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 for already-resolved", w.Code)
	}
	if len(attRepo.created) != 0 {
		t.Fatalf("created %d attendances, want 0 for double-override", len(attRepo.created))
	}
}

func TestAdminOverrideFailedAttempt_NotFound(t *testing.T) {
	handler, _, attRepo := buildOverrideHandler(t, nil, nil)

	w := callOverride(handler, "999", `{"reason":"x"}`)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for missing attempt", w.Code)
	}
	if len(attRepo.created) != 0 {
		t.Fatalf("created %d attendances, want 0", len(attRepo.created))
	}
}

func TestAdminOverrideFailedAttempt_IdempotentWhenAttendanceAlreadyExists(t *testing.T) {
	loc := clock.DefaultLocation
	rows := []*domain.AttendanceFailedAttempt{{
		ID: 1, EmployeeID: 7, AttemptType: "check_in", ReasonCategory: "gps_timeout",
		ProjectID: 55, CreatedAt: time.Date(2026, 7, 3, 8, 58, 0, 0, loc),
	}}
	existing := &domain.Attendance{ID: 33, EmployeeID: 7, ProjectID: 55, Date: time.Date(2026, 7, 3, 0, 0, 0, 0, loc)}
	handler, failedRepo, attRepo := buildOverrideHandler(t, rows, map[string]*domain.Attendance{
		"2026-07-03": existing,
	})

	w := callOverride(handler, "1", `{"reason":"override"}`)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if len(attRepo.created) != 0 {
		t.Fatalf("created %d attendances, want 0 (should reuse existing)", len(attRepo.created))
	}
	if !failedRepo.rows[0].IsResolved() {
		t.Fatal("attempt should still be marked resolved even when reusing attendance")
	}
}
