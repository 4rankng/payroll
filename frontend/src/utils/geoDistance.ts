const VI_NUMBER_FORMAT = new Intl.NumberFormat('vi-VN', {
  maximumFractionDigits: 0,
});

const VI_KILOMETER_FORMAT = new Intl.NumberFormat('vi-VN', {
  maximumFractionDigits: 1,
});

export function formatDistanceMeters(distanceMeters?: number | null): string {
  if (distanceMeters == null || !Number.isFinite(distanceMeters)) return '—';

  if (distanceMeters >= 1000) {
    return `${VI_KILOMETER_FORMAT.format(distanceMeters / 1000)} km`;
  }

  return `${VI_NUMBER_FORMAT.format(Math.max(0, Math.round(distanceMeters)))} m`;
}

export function formatGeofenceDistanceDelta(
  distanceMeters?: number | null,
  radiusMeters?: number | null,
): string | null {
  if (
    distanceMeters == null ||
    radiusMeters == null ||
    !Number.isFinite(distanceMeters) ||
    !Number.isFinite(radiusMeters) ||
    radiusMeters <= 0
  ) {
    return null;
  }

  const deltaMeters = distanceMeters - radiusMeters;
  if (deltaMeters > 0) {
    return `vượt ${formatDistanceMeters(deltaMeters)}`;
  }

  return `trong bán kính ${formatDistanceMeters(radiusMeters)}`;
}
