import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { CheckInTarget } from "@/types/api/auth.types";
import { getCheckInGeofenceGuidance } from "@/utils/checkInGeofenceGuidance";
import { EmployeeLocationMap } from "./EmployeeLocationMap";
import {
  buildGeofenceFeatureCollection,
  EMPLOYEE_MAP_STYLE,
  getEmployeeMapViewport,
  shouldShowEmployeeRouteForDisplay,
} from "./employee-location-map-model";

const fitBoundsMock = vi.hoisted(() => vi.fn());
const mapRemoveMock = vi.hoisted(() => vi.fn());
const webglContextMock = vi.hoisted(() => vi.fn((contextId: string) => (contextId === "webgl" ? {} : null)));
const mapRenderShouldThrow = vi.hoisted(() => ({ value: false }));
const mapInstances = vi.hoisted(() => [] as Array<{
  emit: (event: string) => void;
}>);

vi.mock("maplibre-gl", () => {
  class MockMarker {
    private element: HTMLElement;
    private lngLat: [number, number] | null = null;

    constructor(options: { element: HTMLElement }) {
      this.element = document.createElement("div");
      this.element.dataset.testid = "marker";
      this.element.appendChild(options.element);
    }

    setLngLat(lngLat: [number, number]) {
      this.lngLat = lngLat;
      this.element.dataset.position = JSON.stringify(lngLat);
      return this;
    }

    addTo(map: { container: HTMLElement }) {
      map.container.appendChild(this.element);
      return this;
    }

    remove() {
      this.element.remove();
    }
  }

  class MockMap {
    container: HTMLElement;
    private handlers = new Map<string, Array<() => void>>();
    private sources = new Map<string, { setData: (data: unknown) => void }>();
    private layers = new Set<string>();

    constructor(options: {
      container: HTMLElement;
      style: unknown;
      center: [number, number];
      zoom: number;
      interactive?: boolean;
      dragRotate?: boolean;
      touchPitch?: boolean;
      pitchWithRotate?: boolean;
      keyboard?: boolean;
      attributionControl?: boolean;
    }) {
      if (mapRenderShouldThrow.value) {
        throw new Error("MapLibre render failed");
      }
      this.container = options.container;
      this.container.dataset.testid = "map";
      this.container.dataset.style = JSON.stringify(options.style);
      this.container.dataset.viewState = JSON.stringify({
        longitude: options.center[0],
        latitude: options.center[1],
        zoom: options.zoom,
      });
      this.container.dataset.interactive = String(options.interactive);
      this.container.dataset.dragRotate = String(options.dragRotate);
      this.container.dataset.touchPitch = String(options.touchPitch);
      this.container.dataset.pitchWithRotate = String(options.pitchWithRotate);
      this.container.dataset.keyboard = String(options.keyboard);
      this.container.dataset.attributionControl = String(options.attributionControl);
      mapInstances.push({ emit: (event: string) => this.emit(event) });
    }

    on(event: string, handler: () => void) {
      this.handlers.set(event, [...(this.handlers.get(event) ?? []), handler]);
      if (event === "load") {
        queueMicrotask(handler);
      }
      return this;
    }

    emit(event: string) {
      for (const handler of this.handlers.get(event) ?? []) handler();
    }

    getSource(id: string) {
      return this.sources.get(id);
    }

    addSource(id: string, source: { data: unknown }) {
      const sourceElement = document.createElement("div");
      sourceElement.dataset.testid = "source";
      sourceElement.dataset.sourceId = id;
      sourceElement.dataset.sourceData = JSON.stringify(source.data);
      this.container.appendChild(sourceElement);
      this.sources.set(id, {
        setData: (data: unknown) => {
          sourceElement.dataset.sourceData = JSON.stringify(data);
        },
      });
    }

    getLayer(id: string) {
      return this.layers.has(id) ? { id } : undefined;
    }

    addLayer(layer: { id: string }) {
      this.layers.add(layer.id);
      const layerElement = document.createElement("div");
      layerElement.dataset.testid = "layer";
      layerElement.dataset.layerId = layer.id;
      this.container.appendChild(layerElement);
    }

    removeLayer(id: string) {
      this.layers.delete(id);
    }

    removeSource(id: string) {
      this.sources.delete(id);
    }

    resize() {}
    fitBounds = fitBoundsMock;
    remove = mapRemoveMock;
  }

  return {
    Map: MockMap,
    Marker: MockMarker,
    setWorkerUrl: vi.fn(),
  };
});

const target: CheckInTarget = {
  project_id: 1,
  project_name: "TingTing Soft HQ",
  radius_meters: 150,
  gates: [
    { name: "Cổng xa", lat: 10.79, lng: 106.72 },
    { name: "Cổng D", lat: 10.7769, lng: 106.7009 },
  ],
};

const singleGateTarget: CheckInTarget = {
  project_id: 2,
  project_name: "TingTing Soft HQ",
  radius_meters: 150,
  gates: [{ name: "Cổng D", lat: 10.7769, lng: 106.7009 }],
};

describe("employee-location-map-model", () => {
  it("builds closed geofence polygons with finite coordinates", () => {
    const collection = buildGeofenceFeatureCollection(singleGateTarget);

    expect(collection.features).toHaveLength(1);
    const ring = collection.features[0].geometry.coordinates[0];
    expect(ring.length).toBeGreaterThan(32);
    expect(ring[0]).toEqual(ring[ring.length - 1]);
    expect(ring.every(([lng, lat]) => Number.isFinite(lng) && Number.isFinite(lat))).toBe(true);
  });

  it("uses the route cutoff only as display guidance", () => {
    expect(
      shouldShowEmployeeRouteForDisplay({
        status: "outside",
        nearestGate: singleGateTarget.gates[0],
        distanceMeters: 499,
        radiusMeters: 100,
      })
    ).toBe(true);
    expect(
      shouldShowEmployeeRouteForDisplay({
        status: "outside",
        nearestGate: singleGateTarget.gates[0],
        distanceMeters: 500,
        radiusMeters: 100,
      })
    ).toBe(true);
    expect(
      shouldShowEmployeeRouteForDisplay({
        status: "outside",
        nearestGate: singleGateTarget.gates[0],
        distanceMeters: 501,
        radiusMeters: 100,
      })
    ).toBe(false);
    expect(
      shouldShowEmployeeRouteForDisplay({
        status: "inaccurate",
        nearestGate: singleGateTarget.gates[0],
        distanceMeters: 80,
        radiusMeters: 100,
      })
    ).toBe(false);
  });

  it("includes geofence radius extents in gate-only viewport bounds", () => {
    const guidance = getCheckInGeofenceGuidance(singleGateTarget, null);
    const viewport = getEmployeeMapViewport(guidance, singleGateTarget, null);
    const gate = singleGateTarget.gates[0];

    expect(viewport.includeSampleInBounds).toBe(false);
    expect(viewport.bounds?.[0][0]).toBeLessThan(gate.lng);
    expect(viewport.bounds?.[1][0]).toBeGreaterThan(gate.lng);
    expect(viewport.bounds?.[0][1]).toBeLessThan(gate.lat);
    expect(viewport.bounds?.[1][1]).toBeGreaterThan(gate.lat);
  });

  it("fits local samples to the nearest gate geofence instead of unrelated project gates", () => {
    const multiGateTarget: CheckInTarget = {
      ...singleGateTarget,
      gates: [
        singleGateTarget.gates[0],
        { name: "Cổng rất xa", lat: 20.844, lng: 106.683 },
      ],
    };
    const sample = { lat: 10.7765, lng: 106.7009, accuracy: 35, timestamp: 1 };
    const guidance = getCheckInGeofenceGuidance(multiGateTarget, sample);
    const viewport = getEmployeeMapViewport(guidance, multiGateTarget, sample);

    expect(viewport.includeSampleInBounds).toBe(true);
    expect(viewport.bounds?.[0][1]).toBeLessThan(sample.lat);
    expect(viewport.bounds?.[1][1]).toBeLessThan(11);
  });

  it("keeps a far current position and its nearest checkpoint in view", () => {
    const multiGateTarget: CheckInTarget = {
      ...singleGateTarget,
      gates: [
        singleGateTarget.gates[0],
        { name: "HQ", lat: 20.844, lng: 106.683 },
      ],
    };
    const farSample = { lat: 10.9, lng: 106.7009, accuracy: 35, timestamp: 1 };
    const farGuidance = getCheckInGeofenceGuidance(multiGateTarget, farSample);
    const farViewport = getEmployeeMapViewport(farGuidance, multiGateTarget, farSample);
    const noPositionViewport = getEmployeeMapViewport(
      getCheckInGeofenceGuidance(multiGateTarget, null),
      multiGateTarget,
      null
    );

    expect(farViewport.includeSampleInBounds).toBe(true);
    expect(farViewport.bounds?.[1][1]).toBeGreaterThanOrEqual(farSample.lat);
    expect(farViewport.bounds?.[0][1]).toBeLessThan(singleGateTarget.gates[0].lat);
    expect(noPositionViewport.bounds?.[1][1]).toBeLessThan(11);
  });
});

describe("EmployeeLocationMap", () => {
  beforeEach(() => {
    fitBoundsMock.mockClear();
    mapRemoveMock.mockClear();
    mapInstances.length = 0;
    mapRenderShouldThrow.value = false;
    webglContextMock.mockImplementation((contextId: string) => (contextId === "webgl" ? {} : null));
    vi.spyOn(HTMLCanvasElement.prototype, "getContext").mockImplementation(webglContextMock as never);
  });

  it("keeps the map within a stacking context below the fixed action dock", async () => {
    const { container } = render(<EmployeeLocationMap target={target} />);
    await screen.findAllByTestId("source");

    expect(container.firstElementChild).toHaveClass("relative", "isolate", "z-0");
    expect(container.firstElementChild).not.toHaveClass("rounded-2xl", "shadow-[0_12px_30px_rgba(15,23,42,0.06)]");
    expect(screen.getByTestId("map").parentElement).toHaveClass("h-80", "sm:h-96");
  });

  it("keeps the complete checkpoint summary readable on narrow mobile cards", async () => {
    const checkpointName = "TingTing Soft Headquarters";
    const narrowCardTarget: CheckInTarget = {
      ...singleGateTarget,
      gates: [{ ...singleGateTarget.gates[0], name: checkpointName }],
    };
    const sample = { lat: 20.82, lng: 106.69, accuracy: 22, timestamp: Date.now() };

    render(<EmployeeLocationMap target={narrowCardTarget} sample={sample} />);
    await screen.findAllByTestId("source");

    const mapCard = screen.getByRole("group", { name: /Bản đồ/ });
    const summary = mapCard.firstElementChild;
    const status = screen.getByText("Ngoài khu vực");
    const checkpointLabel = screen.getAllByText(checkpointName)[0];
    const radius = screen.getByText("150 m").parentElement;
    const gpsAccuracy = screen.getByText("GPS ±22m").closest("span");

    expect(summary).toHaveClass("divide-y", "border-y");
    expect(status).toHaveClass("min-w-0", "break-words");
    expect(status).not.toHaveClass("truncate");
    expect(checkpointLabel).toHaveClass("break-words");
    expect(checkpointLabel).not.toHaveClass("truncate", "max-w-16");
    expect(gpsAccuracy).toHaveClass("shrink-0");
    expect(radius).toHaveClass("shrink-0", "whitespace-nowrap");
    expect(summary).toHaveTextContent("Bán kính 150 m");

    await waitFor(() =>
      expect(document.querySelector("[data-checkpoint-label='true']")).toBeInTheDocument()
    );
    const mapLabel = document.querySelector("[data-checkpoint-label='true']");
    expect(mapLabel).toHaveTextContent(checkpointName);
    expect(mapLabel).toHaveClass(
      "bottom-7",
      "max-w-44",
      "whitespace-normal",
      "break-words"
    );
    expect(mapLabel?.parentElement?.querySelector(".bottom-4.h-3.w-px")).toBeInTheDocument();
  });

  it("renders MapLibre with the approved light style", async () => {
    render(<EmployeeLocationMap target={target} />);

    expect(screen.getByTestId("map")).toHaveAttribute("data-style", JSON.stringify(EMPLOYEE_MAP_STYLE));
    expect(screen.getByTestId("map")).toHaveAttribute("data-interactive", "true");
    expect(screen.getByTestId("map")).toHaveAttribute("data-drag-rotate", "false");
    expect(screen.getByTestId("map")).toHaveAttribute("data-touch-pitch", "false");
    expect(screen.getByTestId("map")).toHaveAttribute("data-pitch-with-rotate", "false");
    expect(screen.getByTestId("map")).toHaveAttribute("data-keyboard", "false");
    expect(screen.getByTestId("map")).toHaveAttribute("data-attribution-control", "false");
    expect(screen.getByRole("application", { name: "Bản đồ vệ tinh có thể kéo và phóng to" })).toBeInTheDocument();
    await screen.findAllByTestId("source");
    expect(
      screen
        .getAllByTestId("source")
        .some((source) => source.getAttribute("data-source-id") === "employee-geofence")
    ).toBe(true);

    await waitFor(() => expect(screen.getAllByTestId("marker")).toHaveLength(target.gates.length));
    for (const gate of target.gates) {
      expect(
        screen
          .getAllByTestId("marker")
          .some((marker) => marker.querySelector(`[data-checkpoint-name="${gate.name}"]`))
      ).toBe(true);
    }
    expect(document.querySelectorAll("[data-checkpoint-label='true']")).toHaveLength(1);
  });

  it("shows a compact straight-line route toward the nearest checkpoint for local samples", async () => {
    const sample = { lat: 10.7765, lng: 106.7009, accuracy: 35, timestamp: Date.now() };
    render(<EmployeeLocationMap target={target} sample={sample} />);

    expect(screen.getAllByText("Cổng D").length).toBeGreaterThan(0);
    expect(screen.getByRole("group", { name: /Bản đồ hướng tới Cổng D/ })).toBeInTheDocument();
    expect(screen.getByText("GPS ±35m").closest("span")).toHaveClass("gps-accuracy-confirmed");

    await screen.findAllByTestId("source");
    const routeSource = screen
      .getAllByTestId("source")
      .find((source) => source.getAttribute("data-source-id") === "employee-route");
    expect(routeSource).toBeTruthy();
    expect(routeSource).toHaveAttribute(
      "data-source-data",
      expect.stringContaining(JSON.stringify([[sample.lng, sample.lat], [target.gates[1].lng, target.gates[1].lat]]))
    );
    expect(
      screen
        .getAllByTestId("source")
        .some((source) => source.getAttribute("data-source-id") === "employee-accuracy")
    ).toBe(false);
    expect(document.querySelector("[data-user-location='true']")).toBeInTheDocument();
    const directionMarker = screen
      .getAllByTestId("marker")
      .find((marker) => marker.querySelector(".checkpoint-direction-arrow"));
    expect(directionMarker?.firstElementChild).toHaveClass("checkpoint-direction-arrow-marker");
    expect(directionMarker?.firstElementChild).not.toHaveClass("rounded-full", "bg-white", "border");
  });

  it("shows a human user marker without a GPS radius or route when GPS is weak", async () => {
    const weakGpsSample = { lat: 10.776, lng: 106.7009, accuracy: 75, timestamp: Date.now() };
    render(<EmployeeLocationMap target={singleGateTarget} sample={weakGpsSample} />);

    await screen.findAllByTestId("source");
    expect(screen.getByText("GPS yếu")).toBeInTheDocument();
    expect(
      screen
        .getAllByTestId("source")
        .some((source) => source.getAttribute("data-source-id") === "employee-route")
    ).toBe(false);
    expect(
      screen
        .getAllByTestId("source")
        .some((source) => source.getAttribute("data-source-id") === "employee-accuracy")
    ).toBe(false);
    const userMarker = document.querySelector("[data-user-location='true']");
    expect(userMarker).toBeInTheDocument();
    expect(userMarker).toHaveClass("employee-user-location-marker");
    expect(userMarker?.querySelector("[data-marker-silhouette='true']")).toHaveAttribute("fill", "#fbbf24");
    expect(userMarker).not.toHaveClass("rounded-full", "bg-white", "border");
    expect(screen.getAllByTestId("marker").some((marker) => marker.textContent?.includes("▲"))).toBe(false);
    expect(document.querySelector(".bg-blue-600")).not.toBeInTheDocument();
  });

  it("suppresses misleading long routes for far outside samples while preserving outside status", async () => {
    const farSample = { lat: 20.82, lng: 106.69, accuracy: 35, timestamp: Date.now() };
    render(<EmployeeLocationMap target={singleGateTarget} sample={farSample} />);

    await screen.findAllByTestId("source");
    expect(screen.getByText("Ngoài khu vực")).toBeInTheDocument();
    expect(screen.getAllByText("Cổng D").length).toBeGreaterThan(0);
    expect(screen.getByText("150 m")).toBeInTheDocument();
    expect(
      screen
        .getAllByTestId("source")
        .some((source) => source.getAttribute("data-source-id") === "employee-route")
    ).toBe(false);
    expect(document.querySelector("[data-user-location='true']")).toBeInTheDocument();
    expect(
      screen.getByRole("group", {
        name: /Bản đồ hiển thị vị trí của bạn và Cổng D/,
      })
    ).toBeInTheDocument();
  });

  it("fits configured geofence first, then only the first usable route view", async () => {
    const firstSample = { lat: 10.7765, lng: 106.7009, accuracy: 35, timestamp: 1 };
    const { rerender } = render(<EmployeeLocationMap target={target} />);
    await waitFor(() => expect(fitBoundsMock).toHaveBeenCalledTimes(1));

    rerender(<EmployeeLocationMap target={target} sample={firstSample} />);
    await waitFor(() => expect(fitBoundsMock).toHaveBeenCalledTimes(2));

    rerender(
      <EmployeeLocationMap
        target={target}
        sample={{ ...firstSample, lat: 10.7766, accuracy: 30, timestamp: 2 }}
      />
    );
    expect(fitBoundsMock).toHaveBeenCalledTimes(2);
  });

  it("keeps one MapLibre instance while repeated location attempts update the sample", async () => {
    const firstSample = {
      lat: 10.7758,
      lng: 106.7009,
      accuracy: 35,
      timestamp: 1,
    };
    const { rerender } = render(
      <EmployeeLocationMap target={target} sample={firstSample} />
    );

    await waitFor(() => expect(mapInstances).toHaveLength(1));

    for (let attempt = 2; attempt <= 4; attempt += 1) {
      rerender(
        <EmployeeLocationMap
          target={target}
          sample={{
            ...firstSample,
            lat: firstSample.lat + attempt / 100_000,
            timestamp: attempt,
          }}
        />
      );
    }

    await waitFor(() => expect(screen.getByTestId("map")).toBeInTheDocument());
    expect(mapInstances).toHaveLength(1);
    expect(screen.queryByText("Không tải được bản đồ.")).not.toBeInTheDocument();
    await waitFor(() =>
      expect(
        document.querySelector("[data-user-location='true']")?.parentElement
      ).toHaveAttribute(
        "data-position",
        JSON.stringify([firstSample.lng, firstSample.lat + 4 / 100_000])
      )
    );
  });

  it("falls back without hiding status details when MapLibre reports an error", async () => {
    render(<EmployeeLocationMap target={target} sample={{ lat: 10.7758, lng: 106.7009, accuracy: 35, timestamp: 1 }} />);

    await waitFor(() => expect(mapInstances).toHaveLength(1));
    act(() => mapInstances[0].emit("error"));

    expect(await screen.findByText("Không tải được bản đồ.")).toBeInTheDocument();
    expect(screen.getByText("GPS yếu")).toBeInTheDocument();
    expect(screen.getByText("Cổng D")).toBeInTheDocument();
    expect(screen.getByText("150 m")).toBeInTheDocument();
  });

  it("can retry after a transient MapLibre error instead of remaining stuck", async () => {
    render(
      <EmployeeLocationMap
        target={target}
        sample={{ lat: 10.7758, lng: 106.7009, accuracy: 35, timestamp: 1 }}
      />
    );

    await waitFor(() => expect(mapInstances).toHaveLength(1));
    act(() => mapInstances[0].emit("error"));

    const retryButton = await screen.findByRole("button", {
      name: "Thử tải lại bản đồ",
    });
    fireEvent.click(retryButton);

    await waitFor(() => expect(mapInstances).toHaveLength(2));
    expect(mapRemoveMock).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("map")).toBeInTheDocument();
    expect(screen.queryByText("Không tải được bản đồ.")).not.toBeInTheDocument();
  });

  it("isolates MapLibre render failures inside the map card", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);
    mapRenderShouldThrow.value = true;

    render(<EmployeeLocationMap target={target} sample={{ lat: 10.7758, lng: 106.7009, accuracy: 35, timestamp: 1 }} />);

    expect(await screen.findByText("Không tải được bản đồ.")).toBeInTheDocument();
    expect(screen.getByText("GPS yếu")).toBeInTheDocument();
    expect(screen.getByText("Cổng D")).toBeInTheDocument();
    expect(screen.getByText("150 m")).toBeInTheDocument();
    consoleError.mockRestore();
  });

  it("does not render markers for malformed coordinates", async () => {
    const invalidTarget: CheckInTarget = {
      ...singleGateTarget,
      gates: [
        singleGateTarget.gates[0],
        { name: "Cổng lỗi", lat: Number.NaN, lng: 106.68 },
      ],
    };

    render(<EmployeeLocationMap target={invalidTarget} />);

    await waitFor(() => expect(screen.getAllByTestId("marker")).toHaveLength(1));
  });

  it("uses the Vietnamese fallback when WebGL is unsupported", () => {
    webglContextMock.mockReturnValue(null);

    render(<EmployeeLocationMap target={target} />);

    expect(screen.getByText("Không tải được bản đồ.")).toBeInTheDocument();
    expect(screen.queryByTestId("map")).not.toBeInTheDocument();
  });
});
