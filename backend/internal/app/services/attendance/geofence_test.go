package attendance

import (
	"strings"
	"testing"

	"api-server/internal/domain"
)

// geofenceTestGateLat/Lng match LGD "Cổng A" seeded in migration 068. Using a
// real gate keeps these tests honest against the actual production coordinates.
const (
	geofenceTestGateLat = 20.8628815
	geofenceTestGateLng = 106.5653889
)

// metersNorthOf returns a coordinate roughly m meters north (m<0 = south) of the
// given point along a meridian (1m ≈ 8.993e-6° latitude). Precise enough for
// geofence threshold tests; we never assert exact distances here, only which side
// of the 100m radius a point lands on.
func metersNorthOf(lat, lng, m float64) (float64, float64) {
	const metersPerDegLat = 111195.0
	return lat + m/metersPerDegLat, lng
}

func newGeofenceTestService() *AttendanceService { return &AttendanceService{} }

func TestValidateGeofenceAcceptsOnGateWithGoodAccuracy(t *testing.T) {
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, 50) // ~50m from gate, inside 100m
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 100,
	}

	gate, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 10})
	if err != nil {
		t.Fatalf("expected on-gate check-in with good accuracy to pass, got %v", err)
	}
	if gate != "Cổng A" {
		t.Fatalf("expected gate \"Cổng A\", got %q", gate)
	}
}

// TestValidateGeofenceRejectsPoorAccuracyEvenOnGate pins the ~800m-bypass fix:
// the reported coordinate is on-gate, but an 800m accuracy means the phone could
// place the worker anywhere in an 800m circle, so the coordinate is untrustworthy
// and must be rejected regardless of where it landed.
func TestValidateGeofenceRejectsPoorAccuracyEvenOnGate(t *testing.T) {
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, 50)
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 100,
	}

	_, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 800})
	if err == nil {
		t.Fatal("expected poor-accuracy fix to be rejected, got nil")
	}
	if !domain.IsValidationError(err) {
		t.Fatalf("expected validation error, got %T", err)
	}
	if !strings.Contains(err.Error(), "GPS không đủ chính xác") {
		t.Fatalf("expected gps_inaccurate message, got %q", err.Error())
	}
}

func TestValidateGeofenceRejectsWhenUncertaintyCrossesRadius(t *testing.T) {
	// The reported point is inside the 100m geofence, but 80m from the gate with
	// 30m accuracy means the worker could plausibly be outside the configured area.
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, 80)
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 100,
	}

	_, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 30})
	if err == nil {
		t.Fatal("expected uncertain boundary fix to be rejected, got nil")
	}
	if !domain.IsValidationError(err) {
		t.Fatalf("expected validation error, got %T", err)
	}
	if !strings.Contains(err.Error(), "GPS không đủ chính xác") {
		t.Fatalf("expected gps_inaccurate message, got %q", err.Error())
	}
}

func TestValidateGeofenceSkipsAccuracyGateWhenUnknown(t *testing.T) {
	// accuracy == 0 (unknown, or not sent by a legacy client) must not block an
	// otherwise on-gate fix — otherwise the rollout would reject every old client.
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, 50)
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 100,
	}

	if _, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 0}); err != nil {
		t.Fatalf("expected unknown-accuracy on-gate fix to pass, got %v", err)
	}
}

func TestValidateGeofenceRejectsOutsideGate(t *testing.T) {
	// ~800m south of the gate: good accuracy, but well outside the 100m radius.
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, -800)
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 100,
	}

	_, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 10})
	if err == nil {
		t.Fatal("expected off-site check-in to be rejected")
	}
	if !strings.Contains(err.Error(), "ngoài khu vực chấm công") {
		t.Fatalf("expected geofence_outside message, got %q", err.Error())
	}
	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		t.Fatalf("expected DomainError, got %T", err)
	}
	if domainErr.Code != attendanceOutsideGeofenceCode {
		t.Fatalf("expected code %q, got %q", attendanceOutsideGeofenceCode, domainErr.Code)
	}
	checkpoint, ok := domainErr.Context["nearest_checkpoint"].(nearestCheckpointGuidance)
	if !ok {
		t.Fatalf("expected nearest checkpoint guidance, got %#v", domainErr.Context)
	}
	if checkpoint.Name != "Cổng A" || checkpoint.Lat != geofenceTestGateLat || checkpoint.Lng != geofenceTestGateLng {
		t.Fatalf("unexpected checkpoint: %#v", checkpoint)
	}
	if checkpoint.DistanceMeters < 700 || checkpoint.DistanceMeters > 900 {
		t.Fatalf("expected roughly 800m distance, got %.2fm", checkpoint.DistanceMeters)
	}
}

func TestValidateGeofenceRejectsWhenNoGatesConfigured(t *testing.T) {
	project := &domain.Project{GeofenceRadiusMeters: 100}

	_, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: geofenceTestGateLat, Lng: geofenceTestGateLng})
	if err == nil {
		t.Fatal("expected no-gates project to be rejected")
	}
	if !strings.Contains(err.Error(), "chưa cấu hình vị trí chấm công") {
		t.Fatalf("expected geofence_not_configured message, got %q", err.Error())
	}
}

// TestValidateGeofenceAcceptsInsideHalfWithZoneScaleAccuracy pins the inner-half
// relaxation: the reported point sits 50 m from a 150 m gate (plainly at the
// gate), but ±100 m accuracy pushes the worst-case circle 8 cm past the radius.
// The strict worst-case rule alone rejects this — observed on demo where an
// employee tapped check-out six times in 42 s and was blocked every time. With
// zone-scale accuracy (100 ≤ 150) and a point in the inner half (50 ≤ 75), the
// point estimate is trusted and the reading passes.
func TestValidateGeofenceAcceptsInsideHalfWithZoneScaleAccuracy(t *testing.T) {
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, 50) // ~50m from gate, inside 150m
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 150,
	}

	gate, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 100})
	if err != nil {
		t.Fatalf("expected an on-gate reading with zone-scale accuracy to pass, got %v", err)
	}
	if gate != "Cổng A" {
		t.Fatalf("expected gate \"Cổng A\", got %q", gate)
	}
}

// TestValidateGeofenceLabelsFarReadingAsOutsideDespitePoorAccuracy pins the
// mislabel fix: a reading ~5.9 km from the gate with ±200 m accuracy used to be
// reported as "GPS không chính xác" because the accuracy guard fired before the
// distance check. A far reading is unambiguously outside, so it must surface the
// geofence_outside message regardless of accuracy.
func TestValidateGeofenceLabelsFarReadingAsOutsideDespitePoorAccuracy(t *testing.T) {
	lat, lng := metersNorthOf(geofenceTestGateLat, geofenceTestGateLng, -5869) // ~5.9km south of the gate
	project := &domain.Project{
		GeofenceGates:        []domain.GeofenceGate{{Name: "Cổng A", Lat: geofenceTestGateLat, Lng: geofenceTestGateLng}},
		GeofenceRadiusMeters: 150,
	}

	_, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: lat, Lng: lng, Accuracy: 200})
	if err == nil {
		t.Fatal("expected a far reading to be rejected, got nil")
	}
	if !strings.Contains(err.Error(), "ngoài khu vực chấm công") {
		t.Fatalf("expected geofence_outside message for a far reading, got %q", err.Error())
	}
}

func TestClassifyAttemptErrorGPSInaccurate(t *testing.T) {
	got := ClassifyAttemptError("Tín hiệu GPS không đủ chính xác. Vui lòng thử lại ngoài trời.")
	if got != "gps_inaccurate" {
		t.Fatalf("expected gps_inaccurate category, got %q", got)
	}
}
