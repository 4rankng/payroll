import type { CheckInTarget, GeofenceGate } from "@/types/api/auth.types";
import type { StyleSpecification } from "maplibre-gl";
import {
  distanceMetersBetween,
  type CheckInGeofenceGuidance,
} from "@/utils/checkInGeofenceGuidance";
import type { LocationSample } from "@/utils/geolocation";

export type MapCoordinate = [longitude: number, latitude: number];
export type MapBounds = [southwest: MapCoordinate, northeast: MapCoordinate];

type Geometry =
  | { type: "Point"; coordinates: MapCoordinate }
  | { type: "LineString"; coordinates: MapCoordinate[] }
  | { type: "Polygon"; coordinates: MapCoordinate[][] };

export interface MapFeature<
  TGeometry extends Geometry = Geometry,
  TProperties extends Record<string, unknown> = Record<string, unknown>,
> {
  type: "Feature";
  geometry: TGeometry;
  properties: TProperties;
}

export interface MapFeatureCollection<
  TFeature extends MapFeature = MapFeature,
> {
  type: "FeatureCollection";
  features: TFeature[];
}

export interface EmployeeMapViewport {
  bounds: MapBounds | null;
  initialCenter: MapCoordinate;
  initialZoom: number;
  configKey: string;
  includeSampleInBounds: boolean;
}

export const EMPLOYEE_MAP_SATELLITE_TILE_URL =
  "https://services.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}";

export const EMPLOYEE_MAP_ATTRIBUTION =
  "© Esri, Maxar, Earthstar, GIS User Community";

export const EMPLOYEE_MAP_STYLE = {
  version: 8,
  name: "Employee attendance satellite",
  sources: {
    "employee-satellite": {
      type: "raster",
      tiles: [EMPLOYEE_MAP_SATELLITE_TILE_URL],
      tileSize: 256,
      attribution: EMPLOYEE_MAP_ATTRIBUTION,
    },
  },
  layers: [
    {
      id: "employee-satellite",
      type: "raster",
      source: "employee-satellite",
      paint: {
        "raster-saturation": -0.18,
        "raster-contrast": -0.08,
        "raster-brightness-min": 0.08,
        "raster-brightness-max": 0.9,
      },
    },
  ],
} satisfies StyleSpecification;

export const EMPLOYEE_MAP_ALLOWED_HOSTS = [
  "basemaps.cartocdn.com",
  "a.basemaps.cartocdn.com",
  "b.basemaps.cartocdn.com",
  "c.basemaps.cartocdn.com",
  "d.basemaps.cartocdn.com",
  "tiles.basemaps.cartocdn.com",
  "tiles-a.basemaps.cartocdn.com",
  "tiles-b.basemaps.cartocdn.com",
  "tiles-c.basemaps.cartocdn.com",
  "tiles-d.basemaps.cartocdn.com",
  "services.arcgisonline.com",
] as const;

const EARTH_RADIUS_METERS = 6_371_000;
const DEFAULT_CENTER: MapCoordinate = [106.7009, 10.7769];
const DEFAULT_ZOOM = 15;
const CIRCLE_SEGMENTS = 96;

export function shouldShowEmployeeRouteForDisplay(
  guidance: CheckInGeofenceGuidance
): boolean {
  if (!guidance.nearestGate || guidance.distanceMeters == null) return false;
  if (!Number.isFinite(guidance.distanceMeters) || guidance.distanceMeters <= 1) return false;

  if (guidance.status === "outside") {
    const radius = guidance.radiusMeters ?? 0;
    const cutoffMeters = Math.max(radius * 4, 500);
    return guidance.distanceMeters <= cutoffMeters;
  }

  return guidance.status === "inside";
}

export function buildGeofencePolygon(
  gate: GeofenceGate,
  radiusMeters: number
): MapFeature<{ type: "Polygon"; coordinates: MapCoordinate[][] }, { name: string; radiusMeters: number }> | null {
  if (!isValidGate(gate) || !Number.isFinite(radiusMeters) || radiusMeters <= 0) return null;

  const lat = toRadians(gate.lat);
  const lng = toRadians(gate.lng);
  const angularDistance = radiusMeters / EARTH_RADIUS_METERS;
  const ring: MapCoordinate[] = [];

  for (let i = 0; i <= CIRCLE_SEGMENTS; i += 1) {
    const bearing = (2 * Math.PI * i) / CIRCLE_SEGMENTS;
    const pointLat = Math.asin(
      Math.sin(lat) * Math.cos(angularDistance) +
        Math.cos(lat) * Math.sin(angularDistance) * Math.cos(bearing)
    );
    const pointLng =
      lng +
      Math.atan2(
        Math.sin(bearing) * Math.sin(angularDistance) * Math.cos(lat),
        Math.cos(angularDistance) - Math.sin(lat) * Math.sin(pointLat)
      );

    ring.push([normalizeLongitude(toDegrees(pointLng)), toDegrees(pointLat)]);
  }

  return {
    type: "Feature",
    geometry: {
      type: "Polygon",
      coordinates: [ring],
    },
    properties: {
      name: gate.name || "Khu vực chấm công",
      radiusMeters,
    },
  };
}

export function buildGeofenceFeatureCollection(
  target: CheckInTarget
): MapFeatureCollection<ReturnType<typeof buildGeofencePolygon> extends infer T ? Exclude<T, null> : never> {
  return {
    type: "FeatureCollection",
    features: target.gates
      .map((gate) => buildGeofencePolygon(gate, target.radius_meters))
      .filter((feature): feature is Exclude<typeof feature, null> => Boolean(feature)),
  };
}

export function buildAccuracyFeature(
  sample: LocationSample | null | undefined
): MapFeature<{ type: "Polygon"; coordinates: MapCoordinate[][] }, { accuracyMeters: number }> | null {
  if (!sample || !isValidCoordinate(sample)) return null;
  const accuracyMeters = Math.max(0, Number.isFinite(sample.accuracy) ? sample.accuracy : 0);
  if (accuracyMeters <= 0) return null;

  const polygon = buildGeofencePolygon(
    { name: "Sai số GPS", lat: sample.lat, lng: sample.lng },
    accuracyMeters
  );
  if (!polygon) return null;

  return {
    type: "Feature",
    geometry: polygon.geometry,
    properties: { accuracyMeters },
  };
}

export function buildRouteFeature(
  sample: LocationSample | null | undefined,
  nearestGate: GeofenceGate | null | undefined,
  shouldShowRoute: boolean
): MapFeature<{ type: "LineString"; coordinates: MapCoordinate[] }, { distanceMeters: number }> | null {
  if (!shouldShowRoute || !sample || !nearestGate) return null;
  if (!isValidCoordinate(sample) || !isValidGate(nearestGate)) return null;

  return {
    type: "Feature",
    geometry: {
      type: "LineString",
      coordinates: [
        [sample.lng, sample.lat],
        [nearestGate.lng, nearestGate.lat],
      ],
    },
    properties: {
      distanceMeters: distanceMetersBetween(sample, nearestGate),
    },
  };
}

export function buildGatePointFeatureCollection(
  target: CheckInTarget,
  nearestGate?: GeofenceGate
): MapFeatureCollection<MapFeature<{ type: "Point"; coordinates: MapCoordinate }, { name: string; nearest: boolean }>> {
  return {
    type: "FeatureCollection",
    features: target.gates
      .filter(isValidGate)
      .map((gate) => ({
        type: "Feature",
        geometry: {
          type: "Point",
          coordinates: [gate.lng, gate.lat],
        },
        properties: {
          name: gate.name || "Cổng chấm công",
          nearest: Boolean(nearestGate && sameGate(gate, nearestGate)),
        },
      })),
  };
}

export function getEmployeeMapViewport(
  guidance: CheckInGeofenceGuidance,
  target: CheckInTarget,
  sample?: LocationSample | null
): EmployeeMapViewport {
  const primaryGate =
    guidance.nearestGate && isValidGate(guidance.nearestGate)
      ? guidance.nearestGate
      : target.gates.find(isValidGate);
  const validSample = sample && isValidCoordinate(sample) ? sample : null;
  const includeSampleInBounds = Boolean(validSample);
  const geofenceFeatures = primaryGate
    ? [buildGeofencePolygon(primaryGate, target.radius_meters)].filter(
        (feature): feature is Exclude<typeof feature, null> => Boolean(feature)
      )
    : buildGeofenceFeatureCollection(target).features;
  const coordinates: MapCoordinate[] = geofenceFeatures.flatMap((feature) => feature.geometry.coordinates[0]);

  if (validSample) {
    coordinates.push([validSample.lng, validSample.lat]);
  }

  if (coordinates.length === 0 && target.gates[0] && isValidGate(target.gates[0])) {
    coordinates.push([target.gates[0].lng, target.gates[0].lat]);
  }

  const bounds = getBounds(coordinates);
  const firstCenter = coordinates[0] ?? DEFAULT_CENTER;
  const sampleViewportKey = validSample
    ? `${validSample.lat.toFixed(3)}:${validSample.lng.toFixed(3)}`
    : "gate-only";

  return {
    bounds,
    initialCenter: bounds ? getBoundsCenter(bounds) : firstCenter,
    initialZoom: DEFAULT_ZOOM,
    configKey: [
      target.project_id,
      target.radius_meters,
      ...target.gates.flatMap((gate) => [gate.name, gate.lat, gate.lng]),
      primaryGate?.name,
      primaryGate?.lat,
      primaryGate?.lng,
      sampleViewportKey,
    ].join(":"),
    includeSampleInBounds,
  };
}

export function toMapCoordinate(point: { lat: number; lng: number }): MapCoordinate {
  return [point.lng, point.lat];
}

function getBounds(coordinates: MapCoordinate[]): MapBounds | null {
  const finite = coordinates.filter(([lng, lat]) => Number.isFinite(lng) && Number.isFinite(lat));
  if (finite.length === 0) return null;

  let minLng = finite[0][0];
  let maxLng = finite[0][0];
  let minLat = finite[0][1];
  let maxLat = finite[0][1];

  for (const [lng, lat] of finite) {
    minLng = Math.min(minLng, lng);
    maxLng = Math.max(maxLng, lng);
    minLat = Math.min(minLat, lat);
    maxLat = Math.max(maxLat, lat);
  }

  return [[minLng, minLat], [maxLng, maxLat]];
}

function getBoundsCenter(bounds: MapBounds): MapCoordinate {
  return [
    (bounds[0][0] + bounds[1][0]) / 2,
    (bounds[0][1] + bounds[1][1]) / 2,
  ];
}

function sameGate(a: GeofenceGate, b: GeofenceGate): boolean {
  return a.lat === b.lat && a.lng === b.lng && a.name === b.name;
}

function isValidGate(gate: GeofenceGate): boolean {
  return isValidCoordinate(gate);
}

function isValidCoordinate(point: { lat: number; lng: number }): boolean {
  return (
    Number.isFinite(point.lat) &&
    Number.isFinite(point.lng) &&
    Math.abs(point.lat) <= 90 &&
    Math.abs(point.lng) <= 180
  );
}

function toRadians(value: number): number {
  return (value * Math.PI) / 180;
}

function toDegrees(value: number): number {
  return (value * 180) / Math.PI;
}

function normalizeLongitude(value: number): number {
  if (value > 180) return value - 360;
  if (value < -180) return value + 360;
  return value;
}
