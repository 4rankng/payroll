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
}

func TestValidateGeofenceRejectsWhenNoGatesConfigured(t *testing.T) {
	project := &domain.Project{GeofenceRadiusMeters: 100}

	_, err := newGeofenceTestService().validateGeofence(project, domain.GeoReading{Lat: geofenceTestGateLat, Lng: geofenceTestGateLng})
	if err == nil {
		t.Fatal("expected no-gates project to be rejected")
	}
	if !strings.Contains(err.Error(), "Chưa cấu hình vị trí vào làm") {
		t.Fatalf("expected geofence_not_configured message, got %q", err.Error())
	}
}

func TestClassifyAttemptErrorGPSInaccurate(t *testing.T) {
	got := ClassifyAttemptError("Tín hiệu GPS không đủ chính xác. Vui lòng thử lại ngoài trời.")
	if got != "gps_inaccurate" {
		t.Fatalf("expected gps_inaccurate category, got %q", got)
	}
}
