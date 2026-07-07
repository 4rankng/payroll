import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CheckInTarget } from "@/types/api/auth.types";
import { useContinuousLocation } from "./useContinuousLocation";

type WatchCb = (pos: GeolocationPosition) => void;
type ErrCb = (err: GeolocationPositionError) => void;

interface GeolocationStub {
  emitFix: (lat: number, lng: number, accuracy: number, ts?: number) => void;
  watchPosition: ReturnType<typeof vi.fn>;
  clearWatch: ReturnType<typeof vi.fn>;
}

const originalGeolocation = navigator.geolocation;

const target: CheckInTarget = {
  project_id: 58,
  project_name: "LGD",
  radius_meters: 150,
  gates: [{ name: "Cong A", lat: 20.8628815, lng: 106.5653889 }],
};

function makePosition(lat: number, lng: number, accuracy: number, timestamp = Date.now()): GeolocationPosition {
  return {
    coords: {
      latitude: lat,
      longitude: lng,
      accuracy,
      altitude: null,
      altitudeAccuracy: null,
      heading: null,
      speed: null,
    } as GeolocationCoordinates,
    timestamp,
  } as GeolocationPosition;
}

function installGeolocationStub(): GeolocationStub {
  let watchCb: WatchCb | null = null;
  const watchPosition = vi.fn((success: WatchCb, error: ErrCb) => {
    watchCb = success;
    void error;
    return 11;
  });
  const clearWatch = vi.fn();
  Object.defineProperty(navigator, "geolocation", {
    configurable: true,
    writable: true,
    value: { watchPosition, clearWatch },
  });

  return {
    watchPosition,
    clearWatch,
    emitFix: (lat, lng, accuracy, ts) => watchCb?.(makePosition(lat, lng, accuracy, ts)),
  };
}

describe("useContinuousLocation", () => {
  let stub: GeolocationStub;

  beforeEach(() => {
    stub = installGeolocationStub();
  });

  afterEach(() => {
    Object.defineProperty(navigator, "geolocation", {
      configurable: true,
      value: originalGeolocation,
    });
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("resolves awaitFreshSample for an outside-geofence fix without making submit-ready true", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true, requiredAccuracyMeters: target.radius_meters })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(1.3521, 103.8198, 35);
    });
    await waitFor(() => expect(result.current.sample).toMatchObject({ lat: 1.3521 }));

    await expect(result.current.awaitFreshSample(30_000)).resolves.toMatchObject({
      lat: 1.3521,
      lng: 103.8198,
      accuracy: 35,
    });
    expect(result.current.isSubmitReady).toBe(false);

    unmount();
  });

  it("keeps awaitSubmitReady gated on an inside-geofence fix", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true, requiredAccuracyMeters: target.radius_meters })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(1.3521, 103.8198, 35);
    });
    await waitFor(() => expect(result.current.sample).toMatchObject({ lat: 1.3521 }));

    vi.useFakeTimers();
    const submitReadyPromise = result.current.awaitSubmitReady(1).catch((error: unknown) => error);
    await vi.advanceTimersByTimeAsync(1);
    await expect(submitReadyPromise).resolves.toMatchObject({
      code: 4,
    });

    unmount();
  });
});
