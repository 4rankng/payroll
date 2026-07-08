package admin

import (
	"context"
	"testing"
	"time"

	attendanceSvc "api-server/internal/app/services/attendance"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
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

func TestMapAdminAttendanceResponseIncludesCheckoutNearestGate(t *testing.T) {
	loc := clock.DefaultLocation
	now := time.Date(2026, 7, 2, 21, 0, 0, 0, loc)
	checkOutTime := time.Date(2026, 7, 2, 20, 55, 0, 0, loc)
	checkOutLat := 10.0201
	checkOutLng := 106.0201
	checkOutGate := "Cổng B"
	att := &domain.Attendance{
		ID:           8,
		EmployeeID:   123,
		ProjectID:    55,
		Date:         time.Date(2026, 7, 2, 0, 0, 0, 0, loc),
		CheckInTime:  time.Date(2026, 7, 2, 8, 0, 0, 0, loc),
		CheckInLat:   10.0001,
		CheckInLng:   106.0001,
		CheckInGate:  "Cổng A",
		CheckOutTime: &checkOutTime,
		CheckOutLat:  &checkOutLat,
		CheckOutLng:  &checkOutLng,
		CheckOutGate: &checkOutGate,
		Employee:     domain.Employee{Fullname: "Đỗ Thị Thoa"},
		Project: domain.Project{
			Name:                 "LGD",
			GeofenceRadiusMeters: 100,
			GeofenceGates: []domain.GeofenceGate{
				{Name: "Cổng A", Lat: 10.0002, Lng: 106.0002},
				{Name: "Cổng B", Lat: 10.0202, Lng: 106.0202},
			},
		},
	}

	got := mapAdminAttendanceResponse(att, now)

	if got.CheckOutNearestCheckpointName == nil || *got.CheckOutNearestCheckpointName != "Cổng B" {
		t.Fatalf("checkout nearest checkpoint = %v, want Cổng B", got.CheckOutNearestCheckpointName)
	}
	if got.CheckOutNearestCheckpointDistanceMeters == nil || *got.CheckOutNearestCheckpointDistanceMeters <= 0 {
		t.Fatalf("checkout nearest distance = %v, want > 0", got.CheckOutNearestCheckpointDistanceMeters)
	}
	if got.CheckOutNearestCheckpointLat == nil || got.CheckOutNearestCheckpointLng == nil {
		t.Fatalf("checkout nearest checkpoint coordinates = (%v, %v), want non-nil", got.CheckOutNearestCheckpointLat, got.CheckOutNearestCheckpointLng)
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
