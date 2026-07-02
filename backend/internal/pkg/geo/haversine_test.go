package geo

import "testing"

func TestHaversineDistanceKnownValues(t *testing.T) {
	if d := HaversineDistance(0, 0, 0, 0); d != 0 {
		t.Fatalf("identical points must be 0m, got %f", d)
	}

	// 1° of longitude at the equator = R * π/180 = 111194.93m for R=6371000.
	// Pins both the earth-radius constant and the degree→radian conversion so a
	// future edit (e.g. swapping to km / wrong constant) is caught.
	d := HaversineDistance(0, 0, 0, 1)
	if d < 111194 || d > 111196 {
		t.Fatalf("expected ~111195m for 1° longitude at the equator, got %fm", d)
	}

	// Regression pin for the geofence check-in bypass: a point ~800m due south of
	// LGD "Cổng A" must read ~800m so the 100m geofence can reject an off-site
	// worker. 800m / 111195 m/deg ≈ 0.0071944° latitude.
	const gateLat, gateLng = 20.8628815, 106.5653889
	south := gateLat - 800.0/111195.0
	got := HaversineDistance(gateLat, gateLng, south, gateLng)
	if got < 799 || got > 801 {
		t.Fatalf("expected ~800m south of gate, got %fm", got)
	}
}
