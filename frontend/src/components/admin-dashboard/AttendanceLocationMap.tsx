import { MapPin, Navigation } from 'lucide-react';

import { formatDistanceMeters } from '@/utils/geoDistance';
import type { AdminAttendanceResponse } from '@/types/api/attendance.types';
import {
  LocationMap,
  formatGpsAccuracy,
  type AccuracyChip,
  type AttemptTone,
  type LocationMapCheckpoint,
  type LocationMapPoint,
  type ReasonSeverity,
} from './LocationMap';

export interface AttendanceLocationMapProps {
  row: AdminAttendanceResponse;
  badge?: { label: string; tone: AttemptTone };
  reasonLabel?: string;
  reasonSeverity?: ReasonSeverity;
  timeLabel?: string;
  subtitle?: string;
  onClose?: () => void;
}

export function AttendanceLocationMap({
  row,
  badge,
  reasonLabel,
  reasonSeverity,
  timeLabel,
  subtitle,
  onClose,
}: AttendanceLocationMapProps) {
  const points: LocationMapPoint[] = [];

  // Check-in coords are non-nullable on the backend (default 0); treat a 0,0
  // fix as "no coordinate" so we don't render a stray Atlantic marker unless
  // real data is present.
  const hasCheckIn = row.check_in_lat != null && row.check_in_lng != null && !(row.check_in_lat === 0 && row.check_in_lng === 0);
  const hasCheckOut =
    row.check_out_lat != null &&
    row.check_out_lng != null &&
    !(row.check_out_lat === 0 && row.check_out_lng === 0);

  if (hasCheckIn) {
    points.push({
      lat: row.check_in_lat,
      lng: row.check_in_lng,
      accuracy: row.check_in_accuracy,
      tone: 'blue',
      label: 'Vị trí lúc vào làm',
      legendLabel: 'Vào làm',
    });
  }
  if (hasCheckOut) {
    points.push({
      lat: row.check_out_lat as number,
      lng: row.check_out_lng as number,
      accuracy: row.check_out_accuracy,
      tone: 'amber',
      label: 'Vị trí lúc tan ca',
      legendLabel: 'Tan ca',
    });
  }

  const checkpoint: LocationMapCheckpoint | null =
    row.nearest_checkpoint_lat != null && row.nearest_checkpoint_lng != null
      ? {
          lat: row.nearest_checkpoint_lat,
          lng: row.nearest_checkpoint_lng,
          name: row.nearest_checkpoint_name?.trim() || undefined,
          radiusMeters: row.geofence_radius_meters,
        }
      : null;

  const radiusMeters = row.geofence_radius_meters;

  const accuracyChips: AccuracyChip[] = [{ icon: Navigation, label: `Vào · ${formatGpsAccuracy(row.check_in_accuracy)}` }];
  if (row.check_out_accuracy != null) {
    accuracyChips.push({ icon: Navigation, label: `Ra · ${formatGpsAccuracy(row.check_out_accuracy)}` });
  }
  if (radiusMeters != null) {
    accuracyChips.push({ icon: MapPin, label: `Bán kính ${formatDistanceMeters(radiusMeters)}` });
  }

  return (
    <LocationMap
      points={points}
      checkpoint={checkpoint}
      badge={badge}
      employeeName={row.employee_name ?? `#${row.employee_id}`}
      subtitle={subtitle ?? row.nearest_checkpoint_name?.trim() ?? undefined}
      distanceMeters={row.nearest_checkpoint_distance_meters}
      distanceLabel="lúc vào · tới điểm chấm"
      radiusMeters={radiusMeters}
      accuracyChips={accuracyChips}
      reasonLabel={reasonLabel}
      reasonSeverity={reasonSeverity}
      timeLabel={timeLabel}
      onClose={onClose}
    />
  );
}
