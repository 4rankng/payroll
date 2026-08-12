package attendance

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/geo"
)

const (
	attendanceGPSInaccurateCode   = "ATTENDANCE_GPS_INACCURATE"
	attendanceOutsideGeofenceCode = "ATTENDANCE_OUTSIDE_GEOFENCE"
)

type nearestCheckpointGuidance struct {
	Name           string  `json:"name"`
	Lat            float64 `json:"lat"`
	Lng            float64 `json:"lng"`
	DistanceMeters float64 `json:"distance_meters"`
}

func newLocationValidationError(code, reason, message string, checkpoint nearestCheckpointGuidance) *domain.DomainError {
	return domain.NewValidationErrorWithCode(code, message).
		WithContext("guidance_type", "location").
		WithContext("reason", reason).
		WithContext("nearest_checkpoint", checkpoint)
}

// validateGeofence checks whether a GPS reading is confidently inside a
// configured gate.
//
// A reading passes a gate when either:
//   - the whole uncertainty circle fits inside the radius
//     (dist + accuracy <= radius); or
//   - the point estimate is comfortably inside the zone (dist <= radius/2) and
//     the reported accuracy is no worse than the zone (0 < accuracy <= radius).
//
// The second path stops a zone-scale GPS reading from blocking an employee who
// is plainly at the gate: the worst-case rule alone rejects a point 50 m from a
// 150 m gate when the phone reports ±100 m, because 50 + 100 overshoots the
// radius by centimetres. Requiring accuracy <= radius keeps grossly inaccurate
// fixes (e.g. ±800 m) rejected via the uncertain branch below.
//
// accuracy == 0 (unknown, or not sent by a legacy client) skips the uncertainty
// terms — dist + 0 <= radius reduces to "point inside" — so legacy on-gate
// fixes still pass. A reading whose uncertainty cannot decide inside vs outside
// is "GPS inaccurate"; a reading outside every gate is "outside geofence" even
// when its accuracy is poor (a far reading is unambiguously outside, so the
// accuracy guard no longer short-circuits before the distance check). Returns an
// error if no gates are configured.
func (s *AttendanceService) validateGeofence(project *domain.Project, reading domain.GeoReading) (string, error) {
	gates := project.GeofenceGates
	if len(gates) == 0 {
		return "", domain.NewValidationError("Dự án chưa cấu hình vị trí chấm công. Vui lòng báo quản lý.")
	}

	radius := float64(project.GeofenceRadiusMeters)
	insideButUncertain := false
	var nearestGate domain.GeofenceGate
	nearestDistance := -1.0
	for _, gate := range gates {
		dist := geo.HaversineDistance(reading.Lat, reading.Lng, gate.Lat, gate.Lng)
		if nearestDistance < 0 || dist < nearestDistance {
			nearestGate = gate
			nearestDistance = dist
		}
		if dist > radius {
			continue
		}
		if reading.Accuracy <= 0 || dist+reading.Accuracy <= radius {
			return gate.Name, nil
		}
		// Point estimate is inside, but the uncertainty circle spills past the
		// radius. Trust it when GPS accuracy is zone-scale and the employee is
		// in the inner half of the zone — they are plainly at the gate, and a
		// few metres of GPS jitter should not block an otherwise valid checkout.
		if reading.Accuracy <= radius && dist <= radius/2 {
			return gate.Name, nil
		}
		insideButUncertain = true
	}
	checkpoint := nearestCheckpointGuidance{
		Name:           nearestGate.Name,
		Lat:            nearestGate.Lat,
		Lng:            nearestGate.Lng,
		DistanceMeters: nearestDistance,
	}
	if insideButUncertain {
		return "", newLocationValidationError(
			attendanceGPSInaccurateCode,
			"gps_inaccurate",
			"Tín hiệu GPS không đủ chính xác để chấm công. Hãy đứng ở nơi thoáng hơn, giữ điện thoại yên vài giây rồi thử lại.",
			checkpoint,
		)
	}
	return "", newLocationValidationError(
		attendanceOutsideGeofenceCode,
		"outside_geofence",
		"Bạn đang ở ngoài khu vực chấm công của dự án. Vui lòng di chuyển đến cổng hoặc khu vực đã được cấu hình.",
		checkpoint,
	)
}

// resolveProject determines the project for a check-in request.
// If projectID is provided (non-zero), it validates the project exists and is flexible.
// If projectID is 0 (omitted), it auto-detects the employee's active flexible project.
// Returns the resolved project to avoid redundant DB queries downstream.
func (s *AttendanceService) resolveProject(ctx context.Context, employeeID, projectID uint) (*domain.Project, error) {
	if projectID != 0 {
		project, err := s.projectRepo.GetByID(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get project: %w", err)
		}
		if !project.IsFlexible {
			return nil, domain.NewValidationError("Dự án không hỗ trợ chấm công linh hoạt")
		}
		return project, nil
	}

	// Auto-detect: find active flexible projects for this employee
	projects, err := s.projectEmployeeRepo.GetActiveProjectsForEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active projects: %w", err)
	}

	var flexibleProjects []*domain.Project
	for _, p := range projects {
		if p.IsFlexible {
			flexibleProjects = append(flexibleProjects, p)
		}
	}

	switch len(flexibleProjects) {
	case 0:
		return nil, domain.NewValidationError("Bạn không thuộc dự án linh hoạt nào. Vui lòng liên hệ quản lý.")
	case 1:
		return flexibleProjects[0], nil
	default:
		return nil, domain.NewValidationError("Bạn thuộc nhiều dự án linh hoạt. Vui lòng chọn dự án trước khi vào làm.")
	}
}

// ResolveProjectID returns the project id a check-in for this employee would
// resolve to, best-effort. The handler uses it to attribute a failed attempt to
// the right project when project_id was omitted (auto-detected), since the
// resolved id is otherwise trapped inside CheckIn's transaction. Returns 0 when
// the project cannot be determined — errors are swallowed because this only
// feeds best-effort forensic logging.
func (s *AttendanceService) ResolveProjectID(ctx context.Context, employeeID, projectID uint) uint {
	project, err := s.resolveProject(ctx, employeeID, projectID)
	if err != nil || project == nil {
		return 0
	}
	return project.ID
}

// ResolveCheckoutProjectID returns the project attached to the attendance record
// a checkout attempt would target at attemptedAt. This is best-effort forensic
// metadata for failed-attempt logging and dashboard enrichment; errors are
// swallowed so the original checkout response path is never blocked.
func (s *AttendanceService) ResolveCheckoutProjectID(ctx context.Context, employeeID uint, attemptedAt time.Time) uint {
	if s == nil || s.attendanceRepo == nil {
		return 0
	}
	if attemptedAt.IsZero() {
		if s.clock == nil {
			return 0
		}
		attemptedAt = s.clock.Now()
	}

	attemptedAt = attemptedAt.In(clock.DefaultLocation)
	today := time.Date(attemptedAt.Year(), attemptedAt.Month(), attemptedAt.Day(), 0, 0, 0, 0, clock.DefaultLocation)

	attendance, err := s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, today)
	if err != nil {
		return 0
	}
	if attendance == nil {
		yesterday := today.AddDate(0, 0, -1)
		attendance, err = s.attendanceRepo.GetByEmployeeAndDate(ctx, employeeID, yesterday)
		if err != nil {
			return 0
		}
	}
	if attendance == nil {
		return 0
	}
	return attendance.ProjectID
}
