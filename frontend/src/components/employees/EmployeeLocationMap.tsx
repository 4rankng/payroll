import { useEffect, useMemo, useState } from "react";
import { Circle, CircleMarker, MapContainer, Polyline, TileLayer, Tooltip, useMap } from "react-leaflet";
import type { LatLngBoundsExpression, LatLngExpression } from "leaflet";
import { MapPin, Navigation } from "lucide-react";
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

  return (
    <div className="overflow-hidden rounded-xl border border-sky-100 bg-white shadow-sm">
      <div className="flex items-center justify-between gap-3 px-3 py-2.5">
        <div className="min-w-0">
          <p className="truncate text-[15px] font-bold leading-5 text-slate-950">{statusTitle(guidance)}</p>
          <p className="mt-0.5 truncate text-[12px] font-semibold leading-4 text-slate-500">
            {statusDescription(guidance, target)}
          </p>
        </div>
        <span className="inline-flex h-9 shrink-0 items-center gap-1.5 rounded-full bg-sky-50 px-2.5 text-[11px] font-bold leading-4 text-sky-700">
          <Navigation className="h-3.5 w-3.5" />
          {sample?.accuracy ? (
            <>+/-{Math.round(sample.accuracy)}m</>
          ) : (
            <>GPS</>
          )}
        </span>
      </div>
      {tileFailed ? (
        <div className="border-t border-slate-100 bg-slate-50 px-3 py-3 text-[13px] font-medium leading-5 text-slate-600">
          Không tải được bản đồ.
        </div>
      ) : (
        <div className="relative h-60 w-full border-t border-slate-100 bg-slate-100">
          <div className="pointer-events-none absolute left-3 top-3 z-[500] inline-flex items-center gap-2 rounded-full bg-white/95 px-3 py-2 text-[12px] font-bold leading-4 text-slate-900 shadow-sm ring-1 ring-slate-200/80">
            <MapPin className="h-4 w-4 text-emerald-600" />
            {target.project_name}
          </div>
          {guidance.distanceMeters != null ? (
            <div className="pointer-events-none absolute bottom-3 left-3 right-3 z-[500] rounded-lg bg-white/95 px-3 py-2 text-[12px] font-semibold leading-4 text-slate-700 shadow-sm ring-1 ring-slate-200/80">
              Cách {guidance.nearestGate?.name || "cổng chấm công"} {formatDistanceMeters(guidance.distanceMeters)}
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
                }}
              >
                <Tooltip direction="top" offset={[0, -8]} opacity={0.95}>
                  {gate.name || "Cổng chấm công"}
                </Tooltip>
              </CircleMarker>
            ))}
            {sample && nearestPoint ? (
              <Polyline
                positions={[center, nearestPoint]}
                pathOptions={{ color: "#0284c7", dashArray: "6 6", weight: 2 }}
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
          <p className="text-[11px] font-bold uppercase leading-4 text-slate-500">Bán kính</p>
          <p className="mt-0.5 truncate text-[14px] font-extrabold leading-5 text-slate-950">
            {formatDistanceMeters(target.radius_meters)}
          </p>
        </div>
        <div className="min-w-0 rounded-lg bg-slate-50 px-3 py-2">
          <p className="text-[11px] font-bold uppercase leading-4 text-slate-500">Cổng gần nhất</p>
          <p className="mt-0.5 truncate text-[14px] font-extrabold leading-5 text-slate-950">
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

  useEffect(() => {
    const points: Array<[number, number]> = sample ? [[sample.lat, sample.lng]] : [];
    for (const gate of target.gates) {
      points.push([gate.lat, gate.lng]);
    }
    if (guidance.nearestGate) {
      points.push([guidance.nearestGate.lat, guidance.nearestGate.lng]);
    }
    const bounds = points as LatLngBoundsExpression;
    map.fitBounds(bounds, { padding: [24, 24], maxZoom: 17 });
  }, [guidance.nearestGate, map, sample, target.gates]);

  return null;
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
