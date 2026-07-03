import { MapPin, Navigation } from 'lucide-react';

import { formatDistanceMeters } from '@/utils/geoDistance';
import type { AdminFailedAttempt } from '@/types/api/dashboard.types';
import {
  LocationMap,
  formatGpsAccuracy,
  type AccuracyChip,
  type AttemptTone,
  type LocationMapCheckpoint,
  type LocationMapPoint,
  type ReasonSeverity,
} from './LocationMap';

export interface FailedAttemptLocationMapProps {
  row: AdminFailedAttempt;
  attemptLabel: string;
  reasonLabel?: string;
  reasonSeverity?: ReasonSeverity;
  timeLabel?: string;
  onClose?: () => void;
}

export function FailedAttemptLocationMap({
  row,
  attemptLabel,
  reasonLabel,
  reasonSeverity,
  timeLabel,
  onClose,
}: FailedAttemptLocationMapProps) {
  const points: LocationMapPoint[] =
    row.lat != null && row.lng != null
      ? [
          {
            lat: row.lat,
            lng: row.lng,
            accuracy: row.accuracy,
            tone: 'blue',
            label: 'Điểm GPS của nhân viên',
            legendLabel: 'GPS nhân viên',
          },
        ]
      : [];

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

  const accuracyChips: AccuracyChip[] = [
    { icon: Navigation, label: formatGpsAccuracy(row.accuracy) },
    ...(radiusMeters != null
      ? [{ icon: MapPin, label: `Bán kính ${formatDistanceMeters(radiusMeters)}` }]
      : []),
  ];

  const attemptTone: AttemptTone =
    row.attempt_type === 'check_in' ? 'blue' : row.attempt_type === 'check_out' ? 'amber' : 'slate';

  return (
    <LocationMap
      points={points}
      checkpoint={checkpoint}
      badge={{ label: attemptLabel, tone: attemptTone }}
      employeeName={row.employee_name ?? `#${row.employee_id}`}
      subtitle={row.nearest_checkpoint_name?.trim() || 'Điểm chấm gần nhất'}
      distanceMeters={row.nearest_checkpoint_distance_meters}
      radiusMeters={radiusMeters}
      accuracyChips={accuracyChips}
      reasonLabel={reasonLabel}
      reasonSeverity={reasonSeverity}
      timeLabel={timeLabel}
      onClose={onClose}
    />
  );
}

export { formatGpsAccuracy } from './LocationMap';
