import { useEffect, useMemo, useRef, useState } from "react";
import { Circle, CircleMarker, MapContainer, Marker, Polyline, TileLayer, Tooltip, useMap } from "react-leaflet";
import { divIcon } from "leaflet";
import type { LatLngBoundsExpression, LatLngExpression, Marker as LeafletMarker } from "leaflet";
import { BadgeCheck, MapPin, Navigation } from "lucide-react";
import "leaflet/dist/leaflet.css";
import type { CheckInTarget } from "@/types/api/auth.types";
import type { LocationSample } from "@/utils/geolocation";
import {
  getCheckInGeofenceGuidance,
  type CheckInGeofenceGuidance,
} from "@/utils/checkInGeofenceGuidance";
import { formatDistanceMeters } from "@/utils/geoDistance";

interface EmployeeLocationMapProps {
  target: CheckInTarget;
  sample?: LocationSample | null;
}

export function EmployeeLocationMap({ target, sample }: EmployeeLocationMapProps) {
  const [tileFailed, setTileFailed] = useState(false);
  const guidance = useMemo(
    () => getCheckInGeofenceGuidance(target, sample),
    [target, sample]
  );
  const firstGate = target.gates[0];
  const center = useMemo<LatLngExpression>(
    () => sample ? [sample.lat, sample.lng] : [firstGate.lat, firstGate.lng],
    [firstGate.lat, firstGate.lng, sample]
  );
  const nearestPoint = guidance.nearestGate
    ? ([guidance.nearestGate.lat, guidance.nearestGate.lng] as LatLngExpression)
    : null;
  const directionArrow = useMemo(
    () => sample && guidance.nearestGate
      ? createDirectionArrow(sample, guidance.nearestGate)
      : null,
    [guidance.nearestGate, sample]
  );
  const nearestGateName = guidance.nearestGate?.name || "cổng chấm công";
  const hasRoute = Boolean(sample && nearestPoint && directionArrow);
  const isAtGate = Boolean(
    sample && nearestPoint && guidance.distanceMeters != null && !directionArrow
  );

  return (
    <div
      className="relative isolate z-0 overflow-hidden rounded-xl border border-sky-100 bg-white"
      role="group"
      aria-label={hasRoute
        ? `Bản đồ hướng tới ${nearestGateName}, cách ${formatDistanceMeters(guidance.distanceMeters)} theo đường thẳng`
        : isAtGate
          ? `Bạn đang ở ${nearestGateName}`
        : "Bản đồ khu vực chấm công"}
    >
      <div className="flex items-center justify-between gap-3 px-3 py-2.5">
        <div className="min-w-0">
          <p className="employee-type-card-title truncate text-slate-950">{statusTitle(guidance)}</p>
          <p className="employee-type-pill mt-0.5 truncate text-slate-500">
            {statusDescription(guidance, target)}
          </p>
        </div>
        <span className={`employee-type-pill inline-flex h-9 shrink-0 items-center gap-1.5 rounded-full px-2.5 ${sample?.accuracy != null && sample.accuracy < 50 ? "gps-accuracy-confirmed bg-emerald-50 text-emerald-700" : "bg-sky-50 text-sky-700"}`}>
          <Navigation className="h-3.5 w-3.5" />
          {sample?.accuracy ? (
            <>{sample.accuracy < 50 ? <BadgeCheck className="h-3.5 w-3.5" aria-hidden="true" /> : null}+/-{Math.round(sample.accuracy)}m</>
          ) : (
            <>GPS</>
          )}
        </span>
      </div>
      {tileFailed ? (
        <div className="employee-type-body-sm border-t border-slate-100 bg-slate-50 px-3 py-3 text-slate-600">
          Không tải được bản đồ.
        </div>
      ) : (
        <div className="relative h-60 w-full border-t border-slate-100 bg-slate-100">
          {hasRoute || isAtGate ? (
            <div className="employee-type-body pointer-events-none absolute bottom-3 left-1/2 z-[500] inline-flex max-w-[calc(100%-1.5rem)] -translate-x-1/2 items-center gap-2 rounded-full border border-emerald-200 bg-white px-3 py-2 font-semibold text-emerald-800">
              <MapPin className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span className="truncate">{nearestGateName}</span>
            </div>
          ) : null}
          <MapContainer
            center={center}
            zoom={16}
            className="h-full w-full"
            zoomControl={false}
            attributionControl={false}
            scrollWheelZoom={false}
            dragging
          >
            <TileLayer
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              eventHandlers={{ tileerror: () => setTileFailed(true) }}
            />
            <FitLocationBounds target={target} sample={sample} guidance={guidance} />
            {target.gates.map((gate) => (
              <Circle
                key={`${gate.name}-${gate.lat}-${gate.lng}`}
                center={[gate.lat, gate.lng]}
                radius={target.radius_meters}
                pathOptions={{
                  color: "#059669",
                  fillColor: "#10b981",
                  fillOpacity: 0.12,
                  weight: 2,
                }}
              >
                <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                  {gate.name || "Khu vực chấm công"}
                </Tooltip>
              </Circle>
            ))}
            {target.gates.map((gate) => (
              <CircleMarker
                key={`gate-${gate.name}-${gate.lat}-${gate.lng}`}
                center={[gate.lat, gate.lng]}
                radius={8}
                pathOptions={{
                  color: "#047857",
                  fillColor: "#ffffff",
                  fillOpacity: 1,
                  weight: 3,
                  className: sample && guidance.nearestGate?.lat === gate.lat && guidance.nearestGate?.lng === gate.lng ? "checkpoint-marker-emphasis" : undefined,
                }}
              >
                <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                  {gate.name || "Cổng chấm công"}
                </Tooltip>
              </CircleMarker>
            ))}
            {hasRoute && nearestPoint ? (
              <Polyline
                positions={[center, nearestPoint]}
                pathOptions={{
                  color: "#ffffff",
                  opacity: 0.88,
                  weight: 7,
                  lineCap: "round",
                  className: "checkpoint-route-underlay",
                }}
              />
            ) : null}
            {hasRoute && nearestPoint ? (
              <Polyline
                positions={[center, nearestPoint]}
                pathOptions={{
                  color: "#0284c7",
                  dashArray: "8 7",
                  weight: 3,
                  lineCap: "round",
                  className: "checkpoint-direction-route checkpoint-route-reveal",
                }}
              />
            ) : null}
            {directionArrow && sample && guidance.nearestGate ? (
              <AnimatedDirectionArrow
                start={sample}
                target={guidance.nearestGate}
                icon={directionArrow.icon}
              />
            ) : null}
            {sample ? (
              <>
                <Circle
                  center={center}
                  radius={Math.max(0, sample.accuracy)}
                  pathOptions={{ color: "#2563eb", fillColor: "#3b82f6", fillOpacity: 0.12, weight: 1 }}
                />
                <CircleMarker
                  center={center}
                  radius={9}
                  pathOptions={{
                    color: "#ffffff",
                    fillColor: "#2563eb",
                    fillOpacity: 1,
                    weight: 3,
                  }}
                >
                  <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                    Bạn đang ở đây
                  </Tooltip>
                </CircleMarker>
              </>
            ) : null}
          </MapContainer>
        </div>
      )}
      <div className="grid grid-cols-2 gap-2 border-t border-slate-100 bg-white px-3 py-2.5">
        <div className="min-w-0 rounded-lg bg-slate-50 px-3 py-2">
          <p className="employee-type-pill uppercase text-slate-500">Bán kính</p>
          <p className="employee-type-body mt-0.5 truncate font-semibold text-slate-950">
            {formatDistanceMeters(target.radius_meters)}
          </p>
        </div>
        <div className="min-w-0 rounded-lg bg-slate-50 px-3 py-2">
          <p className="employee-type-pill uppercase text-slate-500">Cổng gần nhất</p>
          <p className="employee-type-body mt-0.5 truncate font-semibold text-slate-950">
            {guidance.nearestGate?.name || target.gates[0]?.name || "Chưa xác định"}
          </p>
        </div>
      </div>
    </div>
  );
}

function FitLocationBounds({
  guidance,
  sample,
  target,
}: {
  guidance: CheckInGeofenceGuidance;
  sample?: LocationSample | null;
  target: CheckInTarget;
}) {
  const map = useMap();
  const fittedConfigRef = useRef<string | null>(null);
  const fittedGateViewRef = useRef(false);
  const fittedRouteRef = useRef(false);
  const configKey = [
    target.project_id,
    target.radius_meters,
    ...target.gates.flatMap((gate) => [gate.name, gate.lat, gate.lng]),
  ].join(":");

  useEffect(() => {
    if (fittedConfigRef.current !== configKey) {
      fittedConfigRef.current = configKey;
      fittedGateViewRef.current = false;
      fittedRouteRef.current = false;
    }

    if (sample && guidance.nearestGate && !fittedRouteRef.current) {
      const bounds = [
        [sample.lat, sample.lng],
        [guidance.nearestGate.lat, guidance.nearestGate.lng],
      ] as LatLngBoundsExpression;
      map.fitBounds(bounds, { padding: [24, 24], maxZoom: 17 });
      fittedRouteRef.current = true;
      fittedGateViewRef.current = true;
      return;
    }

    if (!sample && !fittedGateViewRef.current && target.gates.length > 0) {
      const bounds = target.gates.map((gate) => [gate.lat, gate.lng]) as LatLngBoundsExpression;
      map.fitBounds(bounds, { padding: [24, 24], maxZoom: 17 });
      fittedGateViewRef.current = true;
    }
  }, [configKey, guidance.nearestGate, map, sample, target.gates]);

  return null;
}

function createDirectionArrow(
  start: { lat: number; lng: number },
  target: { lat: number; lng: number }
): { icon: ReturnType<typeof divIcon> } | null {
  const deltaLat = target.lat - start.lat;
  const deltaLng = target.lng - start.lng;
  if (Math.abs(deltaLat) < 1e-9 && Math.abs(deltaLng) < 1e-9) return null;

  const bearing = getBearingDegrees(start, target);
  const icon = divIcon({
    className: "checkpoint-direction-arrow-marker",
    html: `<svg class="checkpoint-direction-arrow" viewBox="0 0 20 24" aria-hidden="true" style="transform:rotate(${bearing}deg)"><path d="M10 1 19 22 10 17 1 22Z" fill="#0284c7" stroke="#ffffff" stroke-width="2" stroke-linejoin="round"/></svg>`,
    iconSize: [20, 24],
    iconAnchor: [10, 12],
  });

  return { icon };
}

function AnimatedDirectionArrow({
  icon,
  start,
  target,
}: {
  icon: ReturnType<typeof divIcon>;
  start: { lat: number; lng: number };
  target: { lat: number; lng: number };
}) {
  const markerRef = useRef<LeafletMarker | null>(null);

  useEffect(() => {
    const marker = markerRef.current;
    if (!marker) return;

    const updatePosition = (progress: number) => {
      marker.setLatLng([
        start.lat + (target.lat - start.lat) * progress,
        start.lng + (target.lng - start.lng) * progress,
      ]);
    };
    const reduceMotion =
      typeof window.matchMedia === "function" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    if (reduceMotion) {
      updatePosition(0.72);
      return;
    }

    const travelMs = 1_650;
    const pauseMs = 650;
    const startedAt = performance.now();
    let frameId = 0;
    const animate = (now: number) => {
      const elapsed = (now - startedAt) % (travelMs + pauseMs);
      const rawProgress = Math.min(elapsed / travelMs, 1);
      const easedProgress = rawProgress * rawProgress * (3 - 2 * rawProgress);
      updatePosition(0.06 + easedProgress * 0.88);
      const element = marker.getElement();
      if (element) element.style.opacity = elapsed < travelMs ? "1" : "0";
      frameId = window.requestAnimationFrame(animate);
    };
    frameId = window.requestAnimationFrame(animate);
    return () => window.cancelAnimationFrame(frameId);
  }, [start.lat, start.lng, target.lat, target.lng]);

  return (
    <Marker
      ref={markerRef}
      position={[start.lat, start.lng]}
      icon={icon}
      interactive={false}
      keyboard={false}
      zIndexOffset={450}
    />
  );
}

function getBearingDegrees(
  start: { lat: number; lng: number },
  target: { lat: number; lng: number }
): number {
  const startLat = toRadians(start.lat);
  const targetLat = toRadians(target.lat);
  const deltaLng = toRadians(target.lng - start.lng);
  const y = Math.sin(deltaLng) * Math.cos(targetLat);
  const x =
    Math.cos(startLat) * Math.sin(targetLat) -
    Math.sin(startLat) * Math.cos(targetLat) * Math.cos(deltaLng);
  return (Math.atan2(y, x) * 180) / Math.PI;
}

function toRadians(value: number): number {
  return (value * Math.PI) / 180;
}

function statusTitle(guidance: CheckInGeofenceGuidance): string {
  switch (guidance.status) {
    case "inside":
      return "Trong khu vực";
    case "outside":
      return "Ngoài khu vực";
    case "inaccurate":
      return "GPS yếu";
    case "no_position":
      return "Khu vực chấm công";
    case "no_target":
    default:
      return "Chưa có khu vực chấm công";
  }
}

function statusDescription(guidance: CheckInGeofenceGuidance, target: CheckInTarget): string {
  const gateName = guidance.nearestGate?.name || "cổng chấm công";
  switch (guidance.status) {
    case "inside":
      return gateName;
    case "outside":
      return `${gateName} · ${formatDistanceMeters(guidance.distanceMeters)}`;
    case "inaccurate":
      return gateName;
    case "no_position":
      return `${gateName} · bán kính ${formatDistanceMeters(target.radius_meters)}`;
    case "no_target":
    default:
      return "Dự án chưa có đủ thông tin vị trí. Vui lòng báo quản lý.";
  }
}
