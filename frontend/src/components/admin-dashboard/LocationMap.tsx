import { Fragment, useEffect, useMemo, useState } from 'react';
import {
  Circle,
  CircleMarker,
  MapContainer,
  Polyline,
  TileLayer,
  Tooltip,
  useMap,
  ZoomControl,
} from 'react-leaflet';
import type { LatLngBoundsExpression, LatLngExpression } from 'leaflet';
import { AlertTriangle, Clock, Map as MapIcon, MapPin, Satellite, WifiOff, X, type LucideIcon } from 'lucide-react';
import 'leaflet/dist/leaflet.css';

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { formatDistanceMeters } from '@/utils/geoDistance';

export type AttemptTone = 'blue' | 'amber' | 'slate';
export type ReasonSeverity = 'danger' | 'warning' | 'info' | 'neutral';
type BaseLayer = 'street' | 'satellite';
type MarkerTone = 'blue' | 'amber' | 'rose' | 'slate';
type CheckpointTone = 'emerald' | 'amber' | 'blue';

export interface LocationMapPoint {
  lat: number;
  lng: number;
  accuracy?: number | null;
  /** Tooltip shown on hover/tap of the marker. */
  label: string;
  tone: MarkerTone;
  /** When set, the point also appears in the legend. */
  legendLabel?: string;
  /** Optional key of the checkpoint this point should connect to. */
  checkpointKey?: string;
}

export interface LocationMapCheckpoint {
  key?: string;
  lat: number;
  lng: number;
  name?: string;
  radiusMeters?: number | null;
  tone?: CheckpointTone;
  legendLabel?: string;
}

export interface AccuracyChip {
  icon: LucideIcon;
  label: string;
}

interface LocationMapProps {
  points: LocationMapPoint[];
  checkpoint?: LocationMapCheckpoint | null;
  checkpoints?: LocationMapCheckpoint[];
  badge?: { label: string; tone: AttemptTone };
  employeeName?: string;
  subtitle?: string;
  /** Hero distance (e.g. to nearest checkpoint). */
  distanceMeters?: number | null;
  distanceLabel?: string;
  /** Geofence radius — drives the inside/outside status pill + checkpoint ring. */
  radiusMeters?: number | null;
  accuracyChips?: AccuracyChip[];
  reasonLabel?: string;
  reasonSeverity?: ReasonSeverity;
  timeLabel?: string;
  onClose?: () => void;
}

const REASON_ICON_COLOR: Record<ReasonSeverity, string> = {
  danger: 'text-rose-600',
  warning: 'text-amber-600',
  info: 'text-primary',
  neutral: 'text-muted-foreground',
};

const ATTEMPT_BADGE_TONES: Record<AttemptTone, string> = {
  blue: 'bg-blue-50 text-blue-700 ring-blue-600/20',
  amber: 'bg-amber-50 text-amber-700 ring-amber-600/20',
  slate: 'bg-slate-100 text-slate-700 ring-slate-600/20',
};

const POINT_COLORS: Record<MarkerTone, { fill: string; haloStroke: string; haloFill: string; legend: string }> = {
  blue: { fill: '#2563eb', haloStroke: '#2563eb', haloFill: '#3b82f6', legend: 'bg-blue-500' },
  amber: { fill: '#d97706', haloStroke: '#d97706', haloFill: '#f59e0b', legend: 'bg-amber-500' },
  rose: { fill: '#e11d48', haloStroke: '#e11d48', haloFill: '#fb7185', legend: 'bg-rose-500' },
  slate: { fill: '#475569', haloStroke: '#475569', haloFill: '#64748b', legend: 'bg-slate-500' },
};

const CHECKPOINT_COLORS: Record<CheckpointTone, { stroke: string; fill: string; legend: string }> = {
  emerald: { stroke: '#047857', fill: '#10b981', legend: 'bg-emerald-500' },
  amber: { stroke: '#b45309', fill: '#f59e0b', legend: 'bg-amber-500' },
  blue: { stroke: '#1d4ed8', fill: '#3b82f6', legend: 'bg-blue-500' },
};

export function LocationMap({
  points,
  checkpoint,
  checkpoints,
  badge,
  employeeName,
  subtitle,
  distanceMeters,
  distanceLabel = 'tới điểm chấm',
  radiusMeters,
  accuracyChips,
  reasonLabel,
  reasonSeverity,
  timeLabel,
  onClose,
}: LocationMapProps) {
  const [tileFailed, setTileFailed] = useState(false);
  const [baseLayer, setBaseLayer] = useState<BaseLayer>('satellite');
  const effectiveCheckpoints = useMemo(
    () => checkpoints ?? (checkpoint ? [checkpoint] : []),
    [checkpoint, checkpoints],
  );
  const primaryCheckpoint = effectiveCheckpoints[0] ?? null;
  const checkpointByKey = useMemo(() => {
    const entries = effectiveCheckpoints
      .filter((item): item is LocationMapCheckpoint & { key: string } => Boolean(item.key))
      .map((item) => [item.key, item] as const);
    return new Map(entries);
  }, [effectiveCheckpoints]);

  const switchLayer = (layer: BaseLayer) => {
    setTileFailed(false);
    setBaseLayer(layer);
  };

  const center: LatLngExpression | null = useMemo(() => {
    if (points.length) return [points[0].lat, points[0].lng];
    if (primaryCheckpoint) return [primaryCheckpoint.lat, primaryCheckpoint.lng];
    return null;
  }, [points, primaryCheckpoint]);

  const hasDistance = distanceMeters != null && Number.isFinite(distanceMeters);
  const hasRadius = radiusMeters != null && radiusMeters > 0;
  const isInside = hasRadius && hasDistance ? (distanceMeters as number) <= (radiusMeters as number) : false;
  const statusTone = isInside ? 'emerald' : 'rose';
  const showMap = Boolean(center) && !tileFailed;

  return (
    <div className="relative h-full w-full overflow-hidden bg-muted/30">
      {center && !tileFailed ? (
        <div className="absolute inset-0">
          <MapContainer
            center={center}
            zoom={16}
            maxZoom={19}
            className="h-full w-full"
            zoomControl={false}
            attributionControl={false}
            scrollWheelZoom
            dragging
          >
            {baseLayer === 'satellite' ? (
              <TileLayer
                url="https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}"
                maxNativeZoom={19}
                maxZoom={19}
                eventHandlers={{ tileerror: () => setTileFailed(true) }}
              />
            ) : (
              <TileLayer
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                maxNativeZoom={19}
                maxZoom={19}
                eventHandlers={{ tileerror: () => setTileFailed(true) }}
              />
            )}
            <FitBounds points={points} checkpoints={effectiveCheckpoints} />
            <ZoomControl position="bottomright" />

            {effectiveCheckpoints.map((item, i) => {
              const checkpointTone = item.tone ?? 'emerald';
              const c = CHECKPOINT_COLORS[checkpointTone];
              const checkpointRadius = item.radiusMeters ?? radiusMeters;
              return checkpointRadius != null && checkpointRadius > 0 ? (
                <Circle
                  key={`checkpoint-radius-${item.key ?? i}`}
                  center={[item.lat, item.lng]}
                  radius={checkpointRadius}
                  pathOptions={{ color: c.stroke, fillColor: c.fill, fillOpacity: 0.12, weight: 2 }}
                >
                  <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                    {item.name || item.legendLabel || 'Điểm chấm gần nhất'}
                  </Tooltip>
                </Circle>
              ) : null;
            })}

            {effectiveCheckpoints.map((item, i) => {
              const checkpointTone = item.tone ?? 'emerald';
              const c = CHECKPOINT_COLORS[checkpointTone];
              return (
                <CircleMarker
                  key={`checkpoint-marker-${item.key ?? i}`}
                  center={[item.lat, item.lng]}
                  radius={8}
                  pathOptions={{ color: c.stroke, fillColor: '#ffffff', fillOpacity: 1, weight: 3 }}
                >
                  <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                    {item.name || item.legendLabel || 'Điểm chấm gần nhất'}
                  </Tooltip>
                </CircleMarker>
              );
            })}

            {effectiveCheckpoints.length
              ? points.map((p, i) => {
                  const pointCheckpoint = p.checkpointKey ? checkpointByKey.get(p.checkpointKey) : null;
                  const lineCheckpoint = pointCheckpoint ?? primaryCheckpoint;
                  if (!lineCheckpoint) return null;
                  return (
                    <Polyline
                      key={`line-${i}`}
                      positions={[[p.lat, p.lng], [lineCheckpoint.lat, lineCheckpoint.lng]]}
                      pathOptions={{ color: POINT_COLORS[p.tone].haloStroke, dashArray: '6 6', weight: 2 }}
                    />
                  );
                })
              : null}

            {points.map((p, i) => {
              const c = POINT_COLORS[p.tone];
              return (
                <Fragment key={`pt-${i}`}>
                  {p.accuracy != null && p.accuracy > 0 ? (
                    <Circle
                      center={[p.lat, p.lng]}
                      radius={Math.max(0, p.accuracy)}
                      pathOptions={{ color: c.haloStroke, fillColor: c.haloFill, fillOpacity: 0.12, weight: 1 }}
                    />
                  ) : null}
                  <CircleMarker
                    center={[p.lat, p.lng]}
                    radius={9}
                    pathOptions={{ color: '#ffffff', fillColor: c.fill, fillOpacity: 1, weight: 3 }}
                  >
                    <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                      {p.label}
                    </Tooltip>
                  </CircleMarker>
                </Fragment>
              );
            })}
          </MapContainer>
        </div>
      ) : null}

      {/* Top floating context panel */}
      <div className="pointer-events-none absolute inset-x-3 top-[calc(env(safe-area-inset-top)+0.75rem)] z-[500] flex justify-start sm:inset-x-auto sm:left-4 sm:top-4 sm:max-w-sm">
        <div className="pointer-events-auto w-full overflow-hidden rounded-xl border border-white/60 bg-background/90 shadow-none backdrop-blur-md">
          <div className="flex items-start justify-between gap-3 px-3.5 pb-2 pt-3 sm:px-4">
            <div className="min-w-0">
              <div className="flex min-w-0 items-center gap-2">
                {badge ? (
                  <span
                    className={cn(
                      'inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide ring-1 ring-inset',
                      ATTEMPT_BADGE_TONES[badge.tone],
                    )}
                  >
                    {badge.label}
                  </span>
                ) : null}
                <span className="min-w-0 truncate text-sm font-semibold text-foreground">{employeeName}</span>
              </div>
              {subtitle ? (
                <p className="mt-1 flex items-center gap-1 truncate text-xs text-muted-foreground">
                  <MapPin className="h-3 w-3 shrink-0" />
                  {subtitle}
                </p>
              ) : null}
            </div>
            {onClose ? <CloseButton onClick={onClose} /> : null}
          </div>

          <div className="grid grid-cols-1 gap-2 border-t border-border/50 px-3.5 py-2.5 min-[380px]:flex min-[380px]:items-center sm:px-4">
            {hasDistance ? (
              <>
                <span
                  className={cn(
                    'text-[1.65rem] font-bold leading-none tabular-nums min-[380px]:shrink-0',
                    isInside ? 'text-emerald-600' : 'text-rose-600',
                  )}
                >
                  {formatDistanceMeters(distanceMeters)}
                </span>
                <span className="min-w-0 text-xs leading-snug text-muted-foreground">{distanceLabel}</span>
                {hasRadius ? (
                  <span
                    className={cn(
                    'inline-flex min-h-8 w-fit items-center gap-1 rounded-full px-2 py-1 text-[11px] font-semibold min-[380px]:ml-auto min-[380px]:shrink-0',
                      statusTone === 'emerald' ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700',
                    )}
                  >
                    <span
                      className={cn(
                        'h-1.5 w-1.5 rounded-full',
                        statusTone === 'emerald' ? 'bg-emerald-500' : 'bg-rose-500',
                      )}
                    />
                    {isInside ? 'Trong khu vực' : 'Ngoài khu vực'}
                  </span>
                ) : null}
              </>
            ) : (
              <span className="text-sm text-muted-foreground">Chưa có khoảng cách</span>
            )}
          </div>

          {timeLabel || reasonLabel || (accuracyChips && accuracyChips.length) ? (
            <div className="space-y-2 px-3.5 pb-3 pt-0.5 sm:px-4">
              {timeLabel || reasonLabel ? (
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5 text-xs text-muted-foreground">
                  {timeLabel ? (
                    <span className="inline-flex items-center gap-1 tabular-nums">
                      <Clock className="h-3 w-3" />
                      {timeLabel}
                    </span>
                  ) : null}
                  {reasonLabel ? (
                    <span className="inline-flex items-center gap-1">
                      <AlertTriangle
                        className={cn(
                          'h-3 w-3',
                          reasonSeverity ? REASON_ICON_COLOR[reasonSeverity] : 'text-muted-foreground',
                        )}
                      />
                      {reasonLabel}
                    </span>
                  ) : null}
                </div>
              ) : null}
              {accuracyChips && accuracyChips.length ? (
                <div className="flex flex-wrap items-center gap-1.5">
                  {accuracyChips.map((chip, i) => (
                    <StatChip key={i} icon={chip.icon} label={chip.label} />
                  ))}
                </div>
              ) : null}
            </div>
          ) : null}

          <div className="grid grid-cols-1 gap-2 border-t border-border/50 bg-muted/20 px-3 py-2 min-[380px]:flex min-[380px]:items-center">
            <span className="text-[11px] font-semibold uppercase tracking-wide text-muted-foreground">Lớp bản đồ</span>
            <div className="flex w-fit items-center gap-0.5 rounded-full bg-background/70 p-0.5 ring-1 ring-inset ring-border/50 min-[380px]:ml-auto">
              <LayerButton
                active={baseLayer === 'satellite'}
                onClick={() => switchLayer('satellite')}
                icon={Satellite}
                label="Vệ tinh"
              />
              <LayerButton
                active={baseLayer === 'street'}
                onClick={() => switchLayer('street')}
                icon={MapIcon}
                label="Bản đồ"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Bottom-left marker legend */}
      {showMap ? (
        <div className="pointer-events-none absolute bottom-[calc(env(safe-area-inset-bottom)+0.75rem)] left-3 z-[500] sm:bottom-4 sm:left-4">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-full border border-border/50 bg-background/80 px-3 py-1.5 shadow-none backdrop-blur-md">
            {effectiveCheckpoints.map((item, i) => {
              const checkpointTone = item.tone ?? 'emerald';
              return (
                <LegendItem key={`checkpoint-legend-${item.key ?? i}`}>
                  <span className={cn('h-2 w-2 rounded-full ring-2 ring-black/5', CHECKPOINT_COLORS[checkpointTone].legend)} />
                  {item.legendLabel || 'Điểm chấm'}
                </LegendItem>
              );
            })}
            {points
              .filter((p) => p.legendLabel)
              .map((p, i) => (
                <LegendItem key={i}>
                  <span className={cn('h-2 w-2 rounded-full ring-2 ring-black/5', POINT_COLORS[p.tone].legend)} />
                  {p.legendLabel}
                </LegendItem>
              ))}
          </div>
        </div>
      ) : null}

      {/* Empty / tile-failed state */}
      {!showMap ? (
        <div className="absolute inset-0 z-[500] flex items-center justify-center p-6">
          <div className="flex max-w-xs flex-col items-center gap-3 text-center">
            <span className="flex h-12 w-12 items-center justify-center rounded-full bg-muted text-muted-foreground">
              {tileFailed ? <WifiOff className="h-6 w-6" /> : <MapPin className="h-6 w-6" />}
            </span>
            <p className="text-sm text-muted-foreground">
              {tileFailed
                ? 'Không tải được nền bản đồ.'
                : 'Không có đủ dữ liệu GPS để hiển thị bản đồ.'}
            </p>
            {onClose ? (
              <Button variant="outline" size="sm" onClick={onClose}>
                Đóng
              </Button>
            ) : null}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function FitBounds({
  points,
  checkpoints,
}: {
  points: LocationMapPoint[];
  checkpoints: LocationMapCheckpoint[];
}) {
  const map = useMap();

  // Stable key derived from coordinates only, so layer toggles (which re-render
  // but don't move points) don't reset the user's pan/zoom.
  const boundsKey = useMemo(() => {
    const pts = points.map((p) => `${p.lat.toFixed(6)},${p.lng.toFixed(6)}`).join('|');
    const cp = checkpoints.map((item) => `${item.lat.toFixed(6)},${item.lng.toFixed(6)}`).join('|');
    return `${pts}#${cp}`;
  }, [points, checkpoints]);

  useEffect(() => {
    const pts: [number, number][] = points.map((p) => [p.lat, p.lng]);
    for (const item of checkpoints) {
      pts.push([item.lat, item.lng]);
    }
    if (!pts.length) return;
    map.fitBounds(pts as LatLngBoundsExpression, { padding: [48, 48], maxZoom: 17 });
    // Leaflet renders gray tiles when its container is sized during an open
    // animation (e.g. inside a Dialog). Re-measure once the animation settles.
    const reflow = window.setTimeout(() => map.invalidateSize(), 250);
    return () => window.clearTimeout(reflow);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [boundsKey, map]);

  return null;
}

function StatChip({ icon: Icon, label }: { icon: LucideIcon; label: string }) {
  return (
    <span className="inline-flex min-h-8 items-center gap-1.5 rounded-full border border-border/60 bg-background/65 px-2.5 py-1 text-[11px] font-medium text-foreground/80">
      <Icon className="h-3 w-3 text-muted-foreground" />
      {label}
    </span>
  );
}

function LegendItem({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-[11px] font-medium text-muted-foreground">
      {children}
    </span>
  );
}

function LayerButton({
  active,
  onClick,
  icon: Icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: LucideIcon;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        'inline-flex min-h-11 items-center gap-1.5 rounded-full px-3 py-1 text-[11px] font-medium transition-colors',
        active ? 'bg-primary text-primary-foreground shadow-none' : 'text-muted-foreground hover:text-foreground',
      )}
    >
      <Icon className="h-3 w-3" />
      {label}
    </button>
  );
}

function CloseButton({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      variant="outline"
      size="icon"
      className="h-11 w-11 shrink-0 rounded-full border-border/60 bg-background/70 backdrop-blur sm:h-11 sm:w-11"
      onClick={onClick}
      aria-label="Đóng bản đồ"
    >
      <X className="h-4 w-4" />
    </Button>
  );
}

export function formatGpsAccuracy(value?: number | null): string {
  if (value == null || Number.isNaN(value) || value <= 0) return 'GPS chưa rõ';
  return `GPS ±${Math.round(value)} m`;
}
