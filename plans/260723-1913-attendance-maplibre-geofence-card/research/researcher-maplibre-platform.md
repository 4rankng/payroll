# Research Report: MapLibre platform notes for employee geofence card

Generated: 2026-07-23

## Executive Summary

`react-map-gl` with the MapLibre entrypoint is a credible replacement for the employee-only Leaflet card. Current npm metadata shows `react-map-gl@8.1.1` depends on `@vis.gl/react-maplibre@8.1.1` and accepts `maplibre-gl` as a peer dependency. Current `maplibre-gl` is `6.0.0` and is BSD-3-Clause licensed.

The migration should treat MapLibre as a WebGL/vector renderer, not as a geofence authority. Geofence status, nearest gate, radius, GPS accuracy, and Vietnamese copy should keep coming from the existing frontend guidance helper and backend-facing contracts.

## Primary Source Findings

- `react-map-gl` supports a dedicated MapLibre import path: `react-map-gl/maplibre`. Use `Map`, `Marker`, `Source`, and `Layer` from that entrypoint so the implementation does not require a Mapbox token.
- MapLibre GL JS renders maps through WebGL and style documents. It supports custom GeoJSON sources and native style layers for lines, fills, symbols, and circles.
- MapLibre examples cover GeoJSON lines, polygons, points, fit-to-bounds workflows, and WebGL support checks. For a meter-based geofence radius, the safest React implementation is to generate a GeoJSON polygon approximation around each gate and render it with fill/line layers.
- The map should listen for runtime load/style/source errors and expose the same Vietnamese fallback pattern currently used by the Leaflet component.
- Attribution cannot be ignored as a generic control preference. If the chosen basemap style requires attribution, keep a compact attribution control or another visible attribution path that satisfies the provider terms.

## Implementation Notes

- Add `react-map-gl` and `maplibre-gl` with `pnpm`, not `npm`, because `frontend/package.json` declares `pnpm@10.33.2`.
- Import MapLibre CSS once from the employee map module or a narrow app-level style entry: `maplibre-gl/dist/maplibre-gl.css`.
- Keep Leaflet dependencies in place because admin map wrappers still use `react-leaflet` and `leaflet`.
- Prefer a local helper for map geometry and view decisions if it keeps the component readable:
  - `buildGeofencePolygon(gate, radiusMeters)`
  - `buildGeofenceFeatureCollection(target)`
  - `buildRouteFeature(sample, nearestGate, shouldShowRoute)`
  - `getEmployeeMapViewMode(guidance)`
- For `fitBounds`, use MapLibre bounds over the coordinates that are actually meant to be visible. Near samples can fit sample plus nearest gate. Far samples should fit the gate/geofence and show the sample as an outside status indicator instead of drawing a route across a continent-scale viewport.
- Preserve reduced-motion behavior for any marker animation. If animation is not critical to the MapLibre migration, prefer a static directional marker.
- For tests, mock `react-map-gl/maplibre` components and assert props/GeoJSON data instead of relying on WebGL in jsdom.

## Risks

- WebGL can fail on old or constrained devices. The component needs a non-crashing fallback.
- External style URLs can fail independently of application code. Treat style load errors as map load failures and keep the status card readable.
- Disabling attribution may create a provider compliance problem. Hide navigation controls, but do not suppress required attribution unless the selected style/provider allows it.
- Adding MapLibre increases frontend bundle size. The current disclosure/lazy-loading behavior should remain so the map is loaded only when the employee opens it.

## Recommended Provider Choice

Use a muted light MapLibre-compatible basemap such as CARTO Positron only if attribution requirements are satisfied. Keep the style URL configurable in code or a narrow constant so a future self-hosted or paid style can replace it without rewriting the card.
