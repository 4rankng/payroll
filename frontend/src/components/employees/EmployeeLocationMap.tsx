import { useEffect, useMemo, useRef, useState } from "react";
import * as maplibregl from "maplibre-gl";
import maplibreWorkerUrl from "maplibre-gl/dist/maplibre-gl-worker.mjs?url";
import { Navigation } from "lucide-react";
import "maplibre-gl/dist/maplibre-gl.css";
import { Button } from "@/components/ui/button";
import type { CheckInTarget } from "@/types/api/auth.types";
import type { LocationSample } from "@/utils/geolocation";
import {
  getCheckInGeofenceGuidance,
  type CheckInGeofenceGuidance,
} from "@/utils/checkInGeofenceGuidance";
import { formatDistanceMeters } from "@/utils/geoDistance";
import {
  buildGatePointFeatureCollection,
  buildGeofenceFeatureCollection,
  buildRouteFeature,
  EMPLOYEE_MAP_STYLE,
  getEmployeeMapViewport,
  shouldShowEmployeeRouteForDisplay,
} from "./employee-location-map-model";

maplibregl.setWorkerUrl(maplibreWorkerUrl);

interface EmployeeLocationMapProps {
  target: CheckInTarget;
  sample?: LocationSample | null;
}

export function EmployeeLocationMap({ target, sample }: EmployeeLocationMapProps) {
  const [mapFailed, setMapFailed] = useState(false);
  const guidance = useMemo(
    () => getCheckInGeofenceGuidance(target, sample),
    [target, sample]
  );
  const shouldShowRoute = shouldShowEmployeeRouteForDisplay(guidance);
  const viewport = useMemo(
    () => getEmployeeMapViewport(guidance, target, sample),
    [guidance, sample, target]
  );
  const geofenceData = useMemo(() => buildGeofenceFeatureCollection(target), [target]);
  const displayGate =
    guidance.nearestGate ?? target.gates.find(isRenderableCoordinate);
  const gateData = useMemo(
    () => buildGatePointFeatureCollection(target, displayGate),
    [displayGate, target]
  );
  const routeFeature = useMemo(
    () => buildRouteFeature(sample, guidance.nearestGate, shouldShowRoute),
    [guidance.nearestGate, sample, shouldShowRoute]
  );
  const nearestGateName = displayGate?.name || "Cổng chấm công";
  const hasRoute = Boolean(routeFeature);
  const isAtGate = Boolean(
    sample && guidance.nearestGate && guidance.distanceMeters != null && guidance.distanceMeters <= 1
  );
  const webGLAvailable = isWebGLAvailable();
  const canRenderMap = !mapFailed && webGLAvailable;

  return (
    <section
      className="relative isolate z-0 overflow-hidden"
      role="group"
      aria-label={mapAriaLabel(guidance, hasRoute, isAtGate, nearestGateName)}
    >
      <ul className="employee-type-body-sm divide-y divide-slate-100 border-y border-slate-200 bg-white text-slate-700">
        <li className="flex items-center justify-between gap-3 px-3 py-2.5">
          <span className="min-w-0 break-words font-semibold text-slate-950">
            {statusTitle(guidance)}
          </span>
          <span
            className={`badge employee-type-pill inline-flex h-8 shrink-0 items-center gap-1.5 rounded-full border px-2.5 ${
              sample?.accuracy != null && sample.accuracy < 50
                ? "gps-accuracy-confirmed border-emerald-100 bg-emerald-50/80 text-emerald-700"
                : "border-sky-100 bg-sky-50/80 text-sky-700"
            }`}
          >
            <Navigation className="h-3.5 w-3.5" aria-hidden="true" />
            {sample?.accuracy != null ? (
              <>GPS ±{Math.round(sample.accuracy)}m</>
            ) : (
              <>GPS</>
            )}
          </span>
        </li>
        <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2.5">
          <div className="min-w-0">
            <span className="employee-type-label-caps block text-slate-500">
              Điểm gần nhất
            </span>
            <span className="mt-0.5 block break-words font-semibold text-slate-700">
              {nearestGateName}
            </span>
          </div>
          <span className="shrink-0 whitespace-nowrap font-semibold text-slate-500">
            Bán kính <span>{formatDistanceMeters(target.radius_meters)}</span>
          </span>
        </li>
      </ul>
      {canRenderMap ? (
        <div className="relative h-80 w-full bg-slate-100 sm:h-96">
          <EmployeeMapCanvas
            gateData={gateData}
            geofenceData={geofenceData}
            guidance={guidance}
            routeFeature={routeFeature}
            sample={sample}
            target={target}
            viewport={viewport}
            onMapFailed={() => setMapFailed(true)}
          />
        </div>
      ) : (
        <MapFallback
          onRetry={
            mapFailed && webGLAvailable
              ? () => setMapFailed(false)
              : undefined
          }
        />
      )}
    </section>
  );
}

type EmployeeMapCanvasProps = {
  gateData: ReturnType<typeof buildGatePointFeatureCollection>;
  geofenceData: ReturnType<typeof buildGeofenceFeatureCollection>;
  guidance: CheckInGeofenceGuidance;
  routeFeature: ReturnType<typeof buildRouteFeature>;
  sample?: LocationSample | null;
  target: CheckInTarget;
  viewport: ReturnType<typeof getEmployeeMapViewport>;
  onMapFailed: () => void;
};

function EmployeeMapCanvas({
  gateData,
  geofenceData,
  guidance,
  routeFeature,
  sample,
  target,
  viewport,
  onMapFailed,
}: EmployeeMapCanvasProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const markersRef = useRef<maplibregl.Marker[]>([]);
  const onMapFailedRef = useRef(onMapFailed);
  const fittedConfigRef = useRef<string | null>(null);
  const initialViewportRef = useRef(viewport);
  const [mapLoaded, setMapLoaded] = useState(false);

  useEffect(() => {
    onMapFailedRef.current = onMapFailed;
  }, [onMapFailed]);

  useEffect(() => {
    if (!containerRef.current || mapRef.current) return;

    try {
      const map = new maplibregl.Map({
        container: containerRef.current,
        style: EMPLOYEE_MAP_STYLE,
        center: initialViewportRef.current.initialCenter,
        zoom: initialViewportRef.current.initialZoom,
        attributionControl: false,
        interactive: true,
        dragRotate: false,
        touchPitch: false,
        pitchWithRotate: false,
        keyboard: false,
      });

      mapRef.current = map;
      map.on("error", () => onMapFailedRef.current());
      map.on("load", () => setMapLoaded(true));

      return () => {
        markersRef.current.forEach((marker) => marker.remove());
        markersRef.current = [];
        map.remove();
        mapRef.current = null;
      };
    } catch {
      onMapFailedRef.current();
    }
  }, []);

  useEffect(() => {
    const map = mapRef.current;
    if (!mapLoaded || !map) return;

    try {
      syncMapData(map, {
        gateData,
        geofenceData,
        guidance,
        routeFeature,
        sample,
        target,
      });

      if (fittedConfigRef.current !== viewport.configKey && viewport.bounds) {
        map.resize();
        map.fitBounds(viewport.bounds, {
          padding: { top: 64, right: 96, bottom: 48, left: 96 },
          maxZoom: 17,
          duration: prefersReducedMotion() ? 0 : 300,
        });
        fittedConfigRef.current = viewport.configKey;
      }
    } catch {
      onMapFailedRef.current();
    }
  }, [
    gateData,
    geofenceData,
    guidance,
    mapLoaded,
    routeFeature,
    sample,
    target,
    viewport,
  ]);

  useEffect(() => {
    const map = mapRef.current;
    if (!mapLoaded || !map) return;

    markersRef.current.forEach((marker) => marker.remove());
    markersRef.current = [];

    const nextMarkers: maplibregl.Marker[] = [];
    const displayGate =
      guidance.nearestGate ?? target.gates.find(isRenderableCoordinate);
    for (const gate of target.gates.filter(isRenderableCoordinate)) {
      const isDisplayGate =
        displayGate?.lat === gate.lat && displayGate?.lng === gate.lng;
      const gateName = gate.name || "Cổng chấm công";
      const element = document.createElement("span");
      element.className = "relative grid h-4 w-4 place-items-center overflow-visible";
      element.title = gateName;
      element.dataset.checkpointName = gateName;
      element.setAttribute("aria-label", gateName);

      const dot = document.createElement("span");
      dot.className = `block h-4 w-4 rounded-full border-[3px] border-emerald-700 bg-white shadow-sm ${
        isDisplayGate ? "checkpoint-marker-emphasis" : ""
      }`;
      element.appendChild(dot);

      if (isDisplayGate) {
        const connector = document.createElement("span");
        connector.className =
          "pointer-events-none absolute bottom-4 left-1/2 h-3 w-px -translate-x-1/2 bg-emerald-700/80";
        connector.setAttribute("aria-hidden", "true");
        element.appendChild(connector);

        const label = document.createElement("span");
        label.className =
          "employee-type-pill pointer-events-none absolute bottom-7 left-1/2 z-10 w-max max-w-44 -translate-x-1/2 whitespace-normal break-words rounded-md border border-emerald-200 bg-white/95 px-2 py-1 text-center font-semibold leading-4 text-emerald-900 shadow-sm backdrop-blur";
        label.dataset.checkpointLabel = "true";
        label.textContent = gateName;
        element.appendChild(label);
      }

      nextMarkers.push(new maplibregl.Marker({ element, anchor: "center" }).setLngLat([gate.lng, gate.lat]).addTo(map));
    }

    if (sample && isRenderableCoordinate(sample)) {
      const element = document.createElement("span");
      element.className = "employee-user-location-marker block h-10 w-6";
      element.title = "Bạn đang ở đây";
      element.dataset.userLocation = "true";
      element.innerHTML = `
        <svg viewBox="0 0 28 44" aria-hidden="true" class="h-full w-full overflow-visible">
          <g data-marker-silhouette="true" fill="#fbbf24" stroke="#a16207" stroke-width="1.35">
            <rect x="5.6" y="15.4" width="3.8" height="14.2" rx="1.9" transform="rotate(13 7.5 22.5)"></rect>
            <rect x="18.6" y="15.4" width="3.8" height="14.2" rx="1.9" transform="rotate(-13 20.5 22.5)"></rect>
            <rect x="9.8" y="26.2" width="4.15" height="16.2" rx="2.05" transform="rotate(2 11.9 34.3)"></rect>
            <rect x="14.05" y="26.2" width="4.15" height="16.2" rx="2.05" transform="rotate(-2 16.1 34.3)"></rect>
            <rect x="9.7" y="13.1" width="8.6" height="17.2" rx="3.8"></rect>
            <circle cx="14" cy="6.6" r="5.35"></circle>
          </g>
          <path d="M11.8 15.5v9.2M11.7 4.8a3.8 3.8 0 0 1 2.8-1.6" fill="none" stroke="#fef3c7" stroke-width="1.25" stroke-linecap="round" opacity="0.82"></path>
        </svg>
      `;
      nextMarkers.push(new maplibregl.Marker({ element, anchor: "bottom" }).setLngLat([sample.lng, sample.lat]).addTo(map));
    }

    if (routeFeature && sample && guidance.nearestGate) {
      const element = document.createElement("span");
      element.className = "checkpoint-direction-arrow-marker grid h-8 w-8 place-items-center";
      element.setAttribute("aria-hidden", "true");
      const arrow = document.createElement("span");
      arrow.className = "checkpoint-direction-arrow text-sky-700";
      arrow.style.transform = `rotate(${getBearingDegrees(sample, guidance.nearestGate)}deg)`;
      arrow.textContent = "▲";
      element.appendChild(arrow);
      nextMarkers.push(
        new maplibregl.Marker({ element, anchor: "center" })
          .setLngLat([(sample.lng + guidance.nearestGate.lng) / 2, (sample.lat + guidance.nearestGate.lat) / 2])
          .addTo(map)
      );
    }

    markersRef.current = nextMarkers;
  }, [guidance.nearestGate, guidance.status, mapLoaded, routeFeature, sample, target.gates]);

  return (
    <div
      ref={containerRef}
      className="h-full w-full"
      role="application"
      aria-label="Bản đồ vệ tinh có thể kéo và phóng to"
    />
  );
}

function syncMapData(
  map: maplibregl.Map,
  data: Omit<EmployeeMapCanvasProps, "onMapFailed" | "viewport">
) {
  upsertGeoJsonSource(map, "employee-geofence", data.geofenceData);
  addLayerIfMissing(map, {
    id: "employee-geofence-fill",
    source: "employee-geofence",
    type: "fill",
    paint: { "fill-color": "#10b981", "fill-opacity": 0.14 },
  });
  addLayerIfMissing(map, {
    id: "employee-geofence-line",
    source: "employee-geofence",
    type: "line",
    paint: { "line-color": "#047857", "line-opacity": 0.9, "line-width": 2 },
  });

  removeLayerAndSource(map, ["employee-accuracy-fill", "employee-accuracy-line"], "employee-accuracy");

  if (data.routeFeature) {
    upsertGeoJsonSource(map, "employee-route", data.routeFeature);
    addLayerIfMissing(map, {
      id: "employee-route-underlay",
      source: "employee-route",
      type: "line",
      paint: { "line-color": "#ffffff", "line-opacity": 0.9, "line-width": 7 },
      layout: { "line-cap": "round", "line-join": "round" },
    });
    addLayerIfMissing(map, {
      id: "employee-route-line",
      source: "employee-route",
      type: "line",
      paint: { "line-color": "#0284c7", "line-dasharray": [1.4, 1.8], "line-width": 3 },
      layout: { "line-cap": "round", "line-join": "round" },
    });
  } else {
    removeLayerAndSource(map, ["employee-route-underlay", "employee-route-line"], "employee-route");
  }

  upsertGeoJsonSource(map, "employee-gates", data.gateData);
  addLayerIfMissing(map, {
    id: "employee-gate-halo",
    source: "employee-gates",
    type: "circle",
    paint: {
      "circle-color": "#ffffff",
      "circle-radius": ["case", ["get", "nearest"], 9, 8],
      "circle-stroke-color": "#047857",
      "circle-stroke-width": ["case", ["get", "nearest"], 4, 3],
    },
  });
}

function upsertGeoJsonSource(
  map: maplibregl.Map,
  id: string,
  data: Parameters<maplibregl.GeoJSONSource["setData"]>[0]
) {
  const source = map.getSource(id);
  if (source) {
    (source as maplibregl.GeoJSONSource).setData(data);
    return;
  }
  map.addSource(id, { type: "geojson", data });
}

function addLayerIfMissing(map: maplibregl.Map, layer: maplibregl.LayerSpecification) {
  if (!map.getLayer(layer.id)) {
    map.addLayer(layer);
  }
}

function removeLayerAndSource(map: maplibregl.Map, layerIds: string[], sourceId: string) {
  for (const layerId of layerIds) {
    if (map.getLayer(layerId)) map.removeLayer(layerId);
  }
  if (map.getSource(sourceId)) map.removeSource(sourceId);
}

function MapFallback({ onRetry }: { onRetry?: () => void }) {
  return (
    <div className="employee-type-body-sm flex min-h-12 items-center justify-between gap-3 bg-slate-50 px-3 py-2 text-slate-600">
      <span>Không tải được bản đồ.</span>
      {onRetry ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="shrink-0 bg-white text-sky-700"
          onClick={onRetry}
        >
          Thử tải lại bản đồ
        </Button>
      ) : null}
    </div>
  );
}

function mapAriaLabel(
  guidance: CheckInGeofenceGuidance,
  hasRoute: boolean,
  isAtGate: boolean,
  nearestGateName: string
): string {
  if (hasRoute) {
    return `Bản đồ hướng tới ${nearestGateName}, cách ${formatDistanceMeters(guidance.distanceMeters)} theo đường thẳng`;
  }
  if (isAtGate) return `Bạn đang ở ${nearestGateName}`;
  if (guidance.nearestGate && guidance.distanceMeters != null) {
    return `Bản đồ hiển thị vị trí của bạn và ${nearestGateName}, cách ${formatDistanceMeters(guidance.distanceMeters)}`;
  }
  return "Bản đồ khu vực chấm công";
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

function prefersReducedMotion(): boolean {
  return (
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches
  );
}

function isWebGLAvailable(): boolean {
  if (typeof document === "undefined") return false;

  try {
    const canvas = document.createElement("canvas");
    return Boolean(
      canvas.getContext("webgl2") ||
        canvas.getContext("webgl") ||
        canvas.getContext("experimental-webgl")
    );
  } catch {
    return false;
  }
}

function isRenderableCoordinate(point: { lat: number; lng: number }): boolean {
  return (
    Number.isFinite(point.lat) &&
    Number.isFinite(point.lng) &&
    Math.abs(point.lat) <= 90 &&
    Math.abs(point.lng) <= 180
  );
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
