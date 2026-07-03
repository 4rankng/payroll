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
import { AlertTriangle, Clock, Map, MapPin, Satellite, WifiOff, X, type LucideIcon } from 'lucide-react';
import 'leaflet/dist/leaflet.css';

import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { formatDistanceMeters } from '@/utils/geoDistance';

export type AttemptTone = 'blue' | 'amber' | 'slate';
export type ReasonSeverity = 'danger' | 'warning' | 'info' | 'neutral';
type BaseLayer = 'street' | 'satellite';
type MarkerTone = 'blue' | 'amber' | 'rose' | 'slate';

export interface LocationMapPoint {
  lat: number;
  lng: number;
  accuracy?: number | null;
  /** Tooltip shown on hover/tap of the marker. */
  label: string;
  tone: MarkerTone;
  /** When set, the point also appears in the legend. */
  legendLabel?: string;
}

export interface LocationMapCheckpoint {
  lat: number;
  lng: number;
  name?: string;
  radiusMeters?: number | null;
}

export interface AccuracyChip {
  icon: LucideIcon;
  label: string;
}

interface LocationMapProps {
  points: LocationMapPoint[];
  checkpoint?: LocationMapCheckpoint | null;
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

export function LocationMap({
  points,
  checkpoint,
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

  const switchLayer = (layer: BaseLayer) => {
    setTileFailed(false);
    setBaseLayer(layer);
  };

  const center: LatLngExpression | null = useMemo(() => {
    if (points.length) return [points[0].lat, points[0].lng];
    if (checkpoint) return [checkpoint.lat, checkpoint.lng];
    return null;
  }, [points, checkpoint]);

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
            <FitBounds points={points} checkpoint={checkpoint} />
            <ZoomControl position="bottomright" />

            {checkpoint && hasRadius ? (
              <Circle
                center={[checkpoint.lat, checkpoint.lng]}
                radius={radiusMeters as number}
                pathOptions={{ color: '#059669', fillColor: '#10b981', fillOpacity: 0.12, weight: 2 }}
              >
                <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                  {checkpoint.name || 'Điểm chấm gần nhất'}
                </Tooltip>
              </Circle>
            ) : null}

            {checkpoint ? (
              <CircleMarker
                center={[checkpoint.lat, checkpoint.lng]}
                radius={8}
                pathOptions={{ color: '#047857', fillColor: '#ffffff', fillOpacity: 1, weight: 3 }}
              >
                <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                  {checkpoint.name || 'Điểm chấm gần nhất'}
                </Tooltip>
              </CircleMarker>
            ) : null}

            {checkpoint
              ? points.map((p, i) => (
                  <Polyline
                    key={`line-${i}`}
                    positions={[[p.lat, p.lng], [checkpoint.lat, checkpoint.lng]]}
                    pathOptions={{ color: '#2563eb', dashArray: '6 6', weight: 2 }}
                  />
                ))
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
      <div className="pointer-events-none absolute inset-x-2 top-2 z-[500] flex justify-start sm:inset-x-auto sm:left-4 sm:top-4 sm:max-w-sm">
        <div className="pointer-events-auto w-full overflow-hidden rounded-2xl border border-border/60 bg-background/85 shadow-xl shadow-black/10 ring-1 ring-black/5 backdrop-blur-md">
          <div className="flex items-start justify-between gap-3 px-4 pb-2.5 pt-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                {badge ? (
                  <span
                    className={cn(
                      'inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ring-1 ring-inset',
                      ATTEMPT_BADGE_TONES[badge.tone],
                    )}
                  >
                    {badge.label}
                  </span>
                ) : null}
                <span className="truncate text-sm font-semibold text-foreground">{employeeName}</span>
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

          <div className="flex items-center gap-2 border-t border-border/50 px-4 py-2.5">
            {hasDistance ? (
              <>
                <span
                  className={cn(
                    'text-2xl font-bold leading-none tabular-nums',
                    isInside ? 'text-emerald-600' : 'text-rose-600',
                  )}
                >
                  {formatDistanceMeters(distanceMeters)}
                </span>
                <span className="text-[11px] text-muted-foreground">{distanceLabel}</span>
                {hasRadius ? (
                  <span
                    className={cn(
                      'ml-auto inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold',
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
            <div className="space-y-2 px-4 pb-3 pt-0.5">
              {timeLabel || reasonLabel ? (
                <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
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

          <div className="flex items-center gap-2 border-t border-border/50 bg-muted/20 px-3 py-1.5">
            <span className="text-[10px] font-semibold uppercase tracking-wide text-muted-foreground">Lớp bản đồ</span>
            <div className="ml-auto flex items-center gap-0.5 rounded-full bg-background/70 p-0.5 ring-1 ring-inset ring-border/50">
              <LayerButton
                active={baseLayer === 'satellite'}
                onClick={() => switchLayer('satellite')}
                icon={Satellite}
                label="Vệ tinh"
              />
              <LayerButton
                active={baseLayer === 'street'}
                onClick={() => switchLayer('street')}
                icon={Map}
                label="Bản đồ"
              />
            </div>
          </div>
        </div>
      </div>

      {/* Bottom-left marker legend */}
      {showMap ? (
        <div className="pointer-events-none absolute bottom-2 left-2 z-[500] sm:bottom-4 sm:left-4">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-full border border-border/50 bg-background/80 px-3 py-1.5 shadow-lg shadow-black/5 backdrop-blur-md">
            {checkpoint ? (
              <LegendItem>
                <span className="h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-emerald-500/30" />
                Điểm chấm
              </LegendItem>
            ) : null}
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
  checkpoint,
}: {
  points: LocationMapPoint[];
  checkpoint?: LocationMapCheckpoint | null;
}) {
  const map = useMap();

  // Stable key derived from coordinates only, so layer toggles (which re-render
  // but don't move points) don't reset the user's pan/zoom.
  const boundsKey = useMemo(() => {
    const pts = points.map((p) => `${p.lat.toFixed(6)},${p.lng.toFixed(6)}`).join('|');
    const cp = checkpoint ? `${checkpoint.lat.toFixed(6)},${checkpoint.lng.toFixed(6)}` : '';
    return `${pts}#${cp}`;
  }, [points, checkpoint]);

  useEffect(() => {
    const pts: [number, number][] = points.map((p) => [p.lat, p.lng]);
    if (checkpoint) pts.push([checkpoint.lat, checkpoint.lng]);
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
    <span className="inline-flex items-center gap-1.5 rounded-full border border-border/60 bg-background/60 px-2.5 py-1 text-[11px] font-medium text-foreground/80">
      <Icon className="h-3 w-3 text-muted-foreground" />
      {label}
    </span>
  );
}

function LegendItem({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-[10px] font-medium text-muted-foreground">
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
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium transition-colors',
        active ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground',
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
      className="h-8 w-8 shrink-0 rounded-full border-border/60 bg-background/60 backdrop-blur"
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
