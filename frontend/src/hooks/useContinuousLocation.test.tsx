import { act, renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CheckInTarget } from "@/types/api/auth.types";
import { CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS } from "@/utils/geolocation";
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

  it("resolves awaitAccurateSample for an outside-geofence fix without making submit-ready true", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(1.3521, 103.8198, 35);
    });
    await waitFor(() => expect(result.current.sample).toMatchObject({ lat: 1.3521 }));

    await expect(result.current.awaitAccurateSample(30_000)).resolves.toMatchObject({
      lat: 1.3521,
      lng: 103.8198,
      accuracy: 35,
    });
    expect(result.current.isSubmitReady).toBe(false);

    unmount();
  });

  it("keeps awaitSubmitReady gated on an inside-geofence fix", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(1.3521, 103.8198, 35);
    });
    await waitFor(() => expect(result.current.sample).toMatchObject({ lat: 1.3521 }));

    vi.useFakeTimers();
    let submitReadyPromise: Promise<unknown>;
    act(() => {
      submitReadyPromise = result.current.awaitSubmitReady(1).catch((error: unknown) => error);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    await expect(submitReadyPromise).resolves.toMatchObject({
      code: 4,
    });

    unmount();
  });

  it("stops the GPS watch after a sufficiently accurate fix while retaining the fresh sample", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(target.gates[0].lat, target.gates[0].lng, 49.9);
    });

    await waitFor(() => expect(stub.clearWatch).toHaveBeenCalledWith(11));
    expect(result.current.isWatching).toBe(false);
    await expect(result.current.awaitAccurateSample()).resolves.toMatchObject({ accuracy: 49.9 });

    unmount();
  });

  it("accepts an on-site fix within the project's configured accuracy radius", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(target.gates[0].lat, target.gates[0].lng, 80);
    });

    await waitFor(() => expect(result.current.sample).toMatchObject({ accuracy: 80 }));
    expect(result.current.isSubmitReady).toBe(true);
    await expect(result.current.awaitAccurateSample(1)).resolves.toMatchObject({ accuracy: 80 });

    unmount();
  });

  it("waits for a fix within the project's accuracy radius before providing a check-in sample", async () => {
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    const accurateSamplePromise = result.current.awaitAccurateSample();
    await act(async () => {
      stub.emitFix(target.gates[0].lat, target.gates[0].lng, target.radius_meters + 0.1);
    });
    expect(stub.clearWatch).not.toHaveBeenCalled();

    await act(async () => {
      stub.emitFix(target.gates[0].lat, target.gates[0].lng, target.radius_meters - 0.1);
    });
    await expect(accurateSamplePromise).resolves.toMatchObject({ accuracy: target.radius_meters - 0.1 });

    unmount();
  });

  it("restarts GPS for a stale retained fix before returning a check-in sample", async () => {
    const now = Date.now();
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(now);
    const { result, unmount } = renderHook(() =>
      useContinuousLocation({ target, enabled: true })
    );
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(1));

    await act(async () => {
      stub.emitFix(target.gates[0].lat, target.gates[0].lng, 25, now);
    });
    await waitFor(() => expect(result.current.isWatching).toBe(false));

    nowSpy.mockReturnValue(now + CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS + 1);

    let freshSamplePromise: Promise<unknown>;
    act(() => {
      freshSamplePromise = result.current.awaitAccurateSample();
    });
    await waitFor(() => expect(stub.watchPosition).toHaveBeenCalledTimes(2));

    await act(async () => {
      stub.emitFix(target.gates[0].lat, target.gates[0].lng, 20, now + CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS + 1);
    });
    await expect(freshSamplePromise).resolves.toMatchObject({ accuracy: 20 });

    unmount();
  });
});
