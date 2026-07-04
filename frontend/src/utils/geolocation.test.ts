import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  isSampleFresh,
  requestBestCurrentLocation,
  watchContinuousLocation,
  type ContinuousLocationHandle,
  type LocationSample,
} from "./geolocation";

const originalGeolocation = navigator.geolocation;

function makePosition(accuracy: number, timestamp = Date.now()): GeolocationPosition {
  return {
    coords: {
      latitude: 20.8628815,
      longitude: 106.5653889,
      accuracy,
      altitude: null,
      altitudeAccuracy: null,
      heading: null,
      speed: null,
    },
    timestamp,
  };
}

describe("requestBestCurrentLocation", () => {
  afterEach(() => {
    vi.useRealTimers();
    Object.defineProperty(navigator, "geolocation", {
      configurable: true,
      value: originalGeolocation,
    });
    vi.restoreAllMocks();
  });

  it("resolves after warm-up when the first fresh sample is accurate enough", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-07-02T08:00:00.000Z"));

    let onSuccess: PositionCallback | null = null;
    const clearWatch = vi.fn();
    Object.defineProperty(navigator, "geolocation", {
      configurable: true,
      value: {
        clearWatch,
        watchPosition: vi.fn((success: PositionCallback) => {
          onSuccess = success;
          return 7;
        }),
      },
    });

    let resolved = false;
    const promise = requestBestCurrentLocation(undefined, {
      timeoutMs: 20000,
      minimumWarmupMs: 3000,
      minimumAcceptableSamples: 2,
      requiredAccuracyMeters: 300,
    }).then((result) => {
      resolved = true;
      return result;
    });

    onSuccess?.(makePosition(8));

    await vi.advanceTimersByTimeAsync(2999);
    expect(resolved).toBe(false);

    await vi.advanceTimersByTimeAsync(1);
    const result = await promise;

    expect(result.bestAccuracy).toBe(8);
    expect(result.sampleCount).toBe(1);
    expect(result.elapsedMs).toBe(3000);
    expect(clearWatch).toHaveBeenCalledWith(7);
  });
});

// ---------------------------------------------------------------------------
// watchContinuousLocation — continuous warm-up primitive for instant
// tap-to-submit. Replaces the cold-start that timed out on slow GNSS fixes.
// ---------------------------------------------------------------------------

type WatchCb = (pos: GeolocationPosition) => void;
type ErrCb = (err: GeolocationPositionError) => void;

interface ContinuousStubApi {
  watchPosition: ReturnType<typeof vi.fn>;
  clearWatch: ReturnType<typeof vi.fn>;
  emitFix: (lat: number, lng: number, accuracy: number, ts: number) => void;
  emitError: (code: number) => void;
  watchId: number;
}

function makeFix(lat: number, lng: number, accuracy: number, ts: number): GeolocationPosition {
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
    timestamp: ts,
  } as GeolocationPosition;
}

function installContinuousStub(): ContinuousStubApi {
  let watchCb: WatchCb | null = null;
  let errCb: ErrCb | null = null;
  const watchId = 42;
  const watchPosition = vi.fn((cb: WatchCb, ecb: ErrCb) => {
    watchCb = cb;
    errCb = ecb;
    return watchId;
  });
  const clearWatch = vi.fn((_id: number) => {});
  Object.defineProperty(navigator, "geolocation", {
    configurable: true,
    writable: true,
    value: { watchPosition, clearWatch },
  });
  return {
    watchPosition,
    clearWatch,
    watchId,
    emitFix: (lat, lng, accuracy, ts) => watchCb?.(makeFix(lat, lng, accuracy, ts)),
    emitError: (code) =>
      errCb?.({ code, message: "stub", PERMISSION_DENIED: 1, POSITION_UNAVAILABLE: 2, TIMEOUT: 3 } as unknown as GeolocationPositionError),
  };
}

const freshSample: LocationSample = { lat: 0, lng: 0, accuracy: 5, timestamp: 1000 };

describe("isSampleFresh", () => {
  it("treats the max-age boundary as fresh (inclusive)", () => {
    expect(isSampleFresh(freshSample, 16000, 15000)).toBe(true);
  });

  it("treats anything over max age as stale", () => {
    expect(isSampleFresh(freshSample, 16001, 15000)).toBe(false);
  });
});

describe("watchContinuousLocation", () => {
  let stub: ContinuousStubApi;

  beforeEach(() => {
    stub = installContinuousStub();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("calls watchPosition once and returns a handle", () => {
    const handle = watchContinuousLocation();
    expect(stub.watchPosition).toHaveBeenCalledTimes(1);
    expect(stub.clearWatch).not.toHaveBeenCalled();
    expect(handle).toEqual(
      expect.objectContaining({
        unsubscribe: expect.any(Function),
        getLatestFreshSample: expect.any(Function),
        getProgress: expect.any(Function),
      })
    );
    handle.unsubscribe();
  });

  it("exposes the latest fresh sample and emits progress on each fix", () => {
    const onUpdate = vi.fn();
    const handle = watchContinuousLocation({ onUpdate });

    stub.emitFix(20.868, 106.571, 8, Date.now());

    expect(handle.getLatestFreshSample()).toMatchObject({ lat: 20.868, lng: 106.571, accuracy: 8 });
    expect(onUpdate).toHaveBeenCalledTimes(1);
    const progress = handle.getProgress();
    expect(progress).not.toBeNull();
    expect(progress?.sampleCount).toBe(1);
    expect(progress?.bestAccuracy).toBe(8);
    expect(progress?.bestFreshSample).toMatchObject({ lat: 20.868 });
    expect(progress?.latestFreshSample).toMatchObject({ lat: 20.868 });

    handle.unsubscribe();
  });

  it("tracks the best (lowest) accuracy while exposing the latest fix", () => {
    const handle = watchContinuousLocation();
    const now = Date.now();
    stub.emitFix(20.868, 106.571, 20, now);
    stub.emitFix(20.868, 106.571, 5, now + 100);
    stub.emitFix(20.868, 106.571, 30, now + 200);

    expect(handle.getProgress()?.bestAccuracy).toBe(5);
    // Latest fresh sample is the most recent fix (current-position semantics).
    expect(handle.getLatestFreshSample()?.accuracy).toBe(30);
    handle.unsubscribe();
  });

  it("ignores stale fixes for the submit candidate but still counts them", () => {
    vi.spyOn(Date, "now").mockReturnValue(100_000);
    const handle = watchContinuousLocation();

    stub.emitFix(20.868, 106.571, 10, 100_000); // fresh
    stub.emitFix(21.0, 107.0, 12, 70_000); // stale (30s older than the 15s window)

    expect(handle.getLatestFreshSample()).toMatchObject({ lat: 20.868 }); // unchanged
    expect(handle.getProgress()?.sampleCount).toBe(2); // both counted
    handle.unsubscribe();
  });

  it("unsubscribes exactly once via clearWatch and is idempotent", () => {
    const handle = watchContinuousLocation();
    handle.unsubscribe();
    handle.unsubscribe();
    expect(stub.clearWatch).toHaveBeenCalledTimes(1);
    expect(stub.clearWatch).toHaveBeenCalledWith(stub.watchId);
  });

  it("forwards geolocation errors to onError without auto-unsubscribing", () => {
    const onError = vi.fn();
    const handle = watchContinuousLocation({ onError });
    stub.emitError(1); // PERMISSION_DENIED
    expect(onError).toHaveBeenCalledTimes(1);
    expect(stub.clearWatch).not.toHaveBeenCalled(); // hook owns lifecycle
    handle.unsubscribe();
  });

  it("returns a no-op handle when geolocation is unsupported", () => {
    Object.defineProperty(navigator, "geolocation", {
      configurable: true,
      writable: true,
      value: undefined,
    });
    const handle: ContinuousLocationHandle = watchContinuousLocation();
    expect(handle.getLatestFreshSample()).toBeNull();
    expect(handle.getProgress()).toBeNull();
    expect(() => handle.unsubscribe()).not.toThrow();
  });
});
