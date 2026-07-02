import { afterEach, describe, expect, it, vi } from "vitest";
import { requestBestCurrentLocation } from "./geolocation";

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
