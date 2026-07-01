package admin

import (
	"testing"

	"api-server/internal/domain"
)

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
