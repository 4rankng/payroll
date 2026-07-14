import type { CheckInTarget, GeofenceGate } from "@/types/api/auth.types";
import { formatDistanceMeters } from "@/utils/geoDistance";
import type { LocationSample } from "@/utils/geolocation";

export type CheckInGeofenceStatus =
  | "inside"
  | "outside"
  | "inaccurate"
  | "no_position"
  | "no_target";

export interface CheckInGeofenceGuidance {
  status: CheckInGeofenceStatus;
  nearestGate?: GeofenceGate;
  distanceMeters?: number;
  overByMeters?: number;
  radiusMeters?: number;
  accuracyMeters?: number;
}

const EARTH_RADIUS_METERS = 6371000;

export function distanceMetersBetween(
  a: { lat: number; lng: number },
  b: { lat: number; lng: number }
): number {
  const lat1 = toRadians(a.lat);
  const lat2 = toRadians(b.lat);
  const deltaLat = toRadians(b.lat - a.lat);
  const deltaLng = toRadians(b.lng - a.lng);
  const h =
    Math.sin(deltaLat / 2) * Math.sin(deltaLat / 2) +
    Math.cos(lat1) * Math.cos(lat2) * Math.sin(deltaLng / 2) * Math.sin(deltaLng / 2);

  return EARTH_RADIUS_METERS * 2 * Math.atan2(Math.sqrt(h), Math.sqrt(1 - h));
}

export function getCheckInGeofenceGuidance(
  target: CheckInTarget | null | undefined,
  sample: LocationSample | null | undefined
): CheckInGeofenceGuidance {
  if (!target || !Array.isArray(target.gates) || target.gates.length === 0 || target.radius_meters <= 0) {
    return { status: "no_target" };
  }

  if (!sample) {
    return {
      status: "no_position",
      nearestGate: target.gates[0],
      radiusMeters: target.radius_meters,
    };
  }

  const accuracy = sample.accuracy > 0 ? sample.accuracy : 0;
  let nearestGate: GeofenceGate | undefined;
  let nearestDistance = Number.POSITIVE_INFINITY;
  let hasCoordinateInsideRadius = false;

  for (const gate of target.gates) {
    const distance = distanceMetersBetween(sample, gate);
    if (distance < nearestDistance) {
      nearestDistance = distance;
      nearestGate = gate;
    }
    if (distance + accuracy <= target.radius_meters) {
      return {
        status: "inside",
        nearestGate: gate,
        distanceMeters: distance,
        radiusMeters: target.radius_meters,
        accuracyMeters: accuracy,
      };
    }
    if (distance <= target.radius_meters) {
      hasCoordinateInsideRadius = true;
    }
  }

  if (!nearestGate || !Number.isFinite(nearestDistance)) {
    return { status: "no_target" };
  }

  const overByMeters = Math.max(0, nearestDistance - target.radius_meters);
  return {
    status: hasCoordinateInsideRadius ? "inaccurate" : "outside",
    nearestGate,
    distanceMeters: nearestDistance,
    overByMeters,
    radiusMeters: target.radius_meters,
    accuracyMeters: accuracy,
  };
}

export function getCheckInGeofenceInstruction(
  guidance: CheckInGeofenceGuidance
): string | null {
  const gateName = guidance.nearestGate?.name || "cổng chấm công gần nhất";

  if (guidance.status === "outside") {
    return `Hãy di chuyển gần hơn tới ${gateName}. Cách ${formatDistanceMeters(guidance.distanceMeters)}.`;
  }

  if (
    guidance.status === "inaccurate" &&
    typeof guidance.accuracyMeters === "number" &&
    typeof guidance.radiusMeters === "number" &&
    guidance.accuracyMeters <= guidance.radiusMeters
  ) {
    return `Hãy tiến gần hơn tới tâm khu vực chấm công tại ${gateName} rồi thử lại.`;
  }

  return null;
}

function toRadians(value: number): number {
  return (value * Math.PI) / 180;
}
