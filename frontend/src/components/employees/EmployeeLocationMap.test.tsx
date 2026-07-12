import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmployeeLocationMap } from "./EmployeeLocationMap";

vi.mock("react-leaflet", () => ({
  Circle: () => null,
  CircleMarker: ({ pathOptions }: { pathOptions?: { className?: string } }) => <div data-testid="marker" className={pathOptions?.className} />,
  MapContainer: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  Polyline: ({ pathOptions }: { pathOptions?: { className?: string } }) => <div data-testid="route" className={pathOptions?.className} />,
  TileLayer: () => null,
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
  useMap: () => ({ fitBounds: vi.fn() }),
}));

describe("EmployeeLocationMap", () => {
  it("keeps Leaflet layers within a stacking context below the fixed action dock", () => {
    const { container } = render(
      <EmployeeLocationMap
        target={{
          project_id: 1,
          project_name: "Ting Ting Soft HQ",
          radius_meters: 100,
          gates: [{ name: "Cổng chính", lat: 10.7769, lng: 106.7009 }],
        }}
      />
    );

    expect(container.firstElementChild).toHaveClass("relative", "isolate", "z-0");
  });

  it("emphasizes the route, nearest checkpoint, and a sub-50m GPS fix when outside", () => {
    render(
      <EmployeeLocationMap
        target={{
          project_id: 1,
          project_name: "Ting Ting Soft HQ",
          radius_meters: 100,
          gates: [{ name: "Cổng chính", lat: 10.7769, lng: 106.7009 }],
        }}
        sample={{ lat: 11.1, lng: 106.9, accuracy: 35, timestamp: Date.now() }}
      />
    );

    expect(screen.getByTestId("route")).toHaveClass("checkpoint-route-reveal");
    expect(screen.getAllByTestId("marker").some((marker) => marker.classList.contains("checkpoint-marker-emphasis"))).toBe(true);
    expect(screen.getByText("+/-35m").closest("span")).toHaveClass("gps-accuracy-confirmed");
  });
});
