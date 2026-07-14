import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { EmployeeLocationMap } from "./EmployeeLocationMap";

const fitBoundsMock = vi.hoisted(() => vi.fn());

vi.mock("react-leaflet", () => ({
  Circle: () => null,
  CircleMarker: ({
    center,
    pathOptions,
  }: {
    center?: [number, number];
    pathOptions?: { className?: string };
  }) => (
    <div
      data-testid="marker"
      data-position={center ? JSON.stringify(center) : undefined}
      className={pathOptions?.className}
    />
  ),
  MapContainer: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Marker: ({
    icon,
    position,
  }: {
    icon: { options: { className?: string; html?: string | HTMLElement } };
    position: [number, number];
  }) => (
    <div
      data-testid="direction-arrow"
      data-position={JSON.stringify(position)}
      className={icon.options.className}
      dangerouslySetInnerHTML={{ __html: typeof icon.options.html === "string" ? icon.options.html : "" }}
    />
  ),
  Polyline: ({
    pathOptions,
    positions,
  }: {
    pathOptions?: { className?: string };
    positions?: Array<[number, number]>;
  }) => (
    <div
      data-testid="route"
      data-positions={positions ? JSON.stringify(positions) : undefined}
      className={pathOptions?.className}
    />
  ),
  TileLayer: () => null,
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
  useMap: () => ({ fitBounds: fitBoundsMock }),
}));

const target = {
  project_id: 1,
  project_name: "Ting Ting Soft HQ",
  radius_meters: 150,
  gates: [
    { name: "Cổng xa", lat: 10.79, lng: 106.72 },
    { name: "Cổng D", lat: 10.7769, lng: 106.7009 },
  ],
};

describe("EmployeeLocationMap", () => {
  beforeEach(() => {
    fitBoundsMock.mockClear();
  });

  it("keeps Leaflet layers within a stacking context below the fixed action dock", () => {
    const { container } = render(<EmployeeLocationMap target={target} />);

    expect(container.firstElementChild).toHaveClass("relative", "isolate", "z-0");
  });

  it("shows a clean straight-line direction and arrow toward the nearest checkpoint", () => {
    const sample = { lat: 10.7758, lng: 106.7009, accuracy: 35, timestamp: Date.now() };
    render(<EmployeeLocationMap target={target} sample={sample} />);

    expect(screen.getAllByText("Cổng D").length).toBeGreaterThan(0);
    expect(screen.queryByText(/Hướng tới/)).not.toBeInTheDocument();
    expect(screen.queryByText(/Đường thẳng tham khảo/)).not.toBeInTheDocument();
    expect(screen.getByRole("group", { name: /Bản đồ hướng tới Cổng D/ })).toBeInTheDocument();

    const directionRoute = screen
      .getAllByTestId("route")
      .find((route) => route.classList.contains("checkpoint-direction-route"));
    expect(directionRoute).toHaveClass("checkpoint-route-reveal");
    expect(directionRoute).toHaveAttribute(
      "data-positions",
      JSON.stringify([[sample.lat, sample.lng], [target.gates[1].lat, target.gates[1].lng]])
    );

    const arrow = screen.getByTestId("direction-arrow");
    expect(arrow).toHaveClass("checkpoint-direction-arrow-marker");
    expect(arrow.querySelector("svg")).toHaveClass("checkpoint-direction-arrow");
    expect(screen.getAllByTestId("marker")[1]).toHaveClass("checkpoint-marker-emphasis");
    expect(screen.getByText("+/-35m").closest("span")).toHaveClass("gps-accuracy-confirmed");
  });

  it("fits the first usable route once and preserves pan or zoom across later GPS updates", () => {
    const firstSample = { lat: 10.7758, lng: 106.7009, accuracy: 48, timestamp: 1 };
    const { rerender } = render(<EmployeeLocationMap target={target} sample={firstSample} />);
    expect(fitBoundsMock).toHaveBeenCalledTimes(1);

    rerender(
      <EmployeeLocationMap
        target={target}
        sample={{ ...firstSample, lat: 10.776, accuracy: 36, timestamp: 2 }}
      />
    );
    expect(fitBoundsMock).toHaveBeenCalledTimes(1);
  });

  it("fits configured gates first when GPS is absent, then fits only the first usable route", () => {
    const { rerender } = render(<EmployeeLocationMap target={target} />);
    expect(fitBoundsMock).toHaveBeenCalledTimes(1);

    rerender(
      <EmployeeLocationMap
        target={target}
        sample={{ lat: 10.7758, lng: 106.7009, accuracy: 48, timestamp: 1 }}
      />
    );
    expect(fitBoundsMock).toHaveBeenCalledTimes(2);

    rerender(
      <EmployeeLocationMap
        target={target}
        sample={{ lat: 10.776, lng: 106.7009, accuracy: 36, timestamp: 2 }}
      />
    );
    expect(fitBoundsMock).toHaveBeenCalledTimes(2);
  });
});
