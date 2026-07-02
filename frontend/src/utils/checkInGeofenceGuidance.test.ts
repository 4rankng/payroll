import { describe, expect, it } from "vitest";
import type { CheckInTarget } from "@/types/api/auth.types";
import type { LocationSample } from "@/utils/geolocation";
import { getCheckInGeofenceGuidance } from "./checkInGeofenceGuidance";

const baseTarget: CheckInTarget = {
  project_id: 58,
  project_name: "LGD",
  radius_meters: 300,
  gates: [{ name: "Cong C", lat: 20.8679818, lng: 106.5711738 }],
};

function sample(lat: number, lng: number, accuracy = 5): LocationSample {
  return { lat, lng, accuracy, timestamp: Date.now() };
}

describe("getCheckInGeofenceGuidance", () => {
  it("returns inside when distance plus accuracy is within the radius", () => {
    const result = getCheckInGeofenceGuidance(baseTarget, sample(20.8679818, 106.5711738, 5));

    expect(result.status).toBe("inside");
    expect(result.nearestGate?.name).toBe("Cong C");
  });

  it("treats equality at the boundary as inside", () => {
    const target: CheckInTarget = {
      ...baseTarget,
      radius_meters: 10,
    };
    const result = getCheckInGeofenceGuidance(target, sample(20.8679818, 106.5711738, 10));

    expect(result.status).toBe("inside");
  });

  it("returns inaccurate when the coordinate is inside but uncertainty crosses the boundary", () => {
    const target: CheckInTarget = {
      ...baseTarget,
      radius_meters: 20,
    };
    const result = getCheckInGeofenceGuidance(target, sample(20.8679818, 106.5711738, 25));

    expect(result.status).toBe("inaccurate");
  });

  it("returns outside for the observed 4.4km-over LGD attempt", () => {
    const result = getCheckInGeofenceGuidance(baseTarget, sample(20.9099307, 106.5657482, 5));

    expect(result.status).toBe("outside");
    expect(result.nearestGate?.name).toBe("Cong C");
    expect(result.distanceMeters ?? 0).toBeGreaterThan(4600);
    expect(result.overByMeters ?? 0).toBeGreaterThan(4300);
  });

  it("does not expand uncertainty for zero or negative accuracy", () => {
    const target: CheckInTarget = {
      ...baseTarget,
      radius_meters: 1,
    };

    expect(getCheckInGeofenceGuidance(target, sample(20.8679818, 106.5711738, 0)).status).toBe("inside");
    expect(getCheckInGeofenceGuidance(target, sample(20.8679818, 106.5711738, -1)).status).toBe("inside");
  });

  it("chooses any passing gate for validation and nearest gate for outside display", () => {
    const target: CheckInTarget = {
      ...baseTarget,
      radius_meters: 50,
      gates: [
        { name: "Far", lat: 20.9, lng: 106.57 },
        { name: "Near", lat: 20.8679818, lng: 106.5711738 },
      ],
    };

    expect(getCheckInGeofenceGuidance(target, sample(20.8679818, 106.5711738, 5)).nearestGate?.name).toBe("Near");
  });

  it("returns no target or no position for missing inputs", () => {
    expect(getCheckInGeofenceGuidance(null, sample(20.8679818, 106.5711738)).status).toBe("no_target");
    expect(getCheckInGeofenceGuidance({ ...baseTarget, gates: [] }, sample(20.8679818, 106.5711738)).status).toBe("no_target");
    expect(getCheckInGeofenceGuidance(baseTarget, null).status).toBe("no_position");
  });
});
