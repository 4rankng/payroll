export type LocationPermissionState = PermissionState | "unsupported" | "unknown";

export interface LocationPermissionIssue {
  type: "unsupported" | "denied" | "unavailable" | "timeout" | "inaccurate" | "unknown";
  title: string;
  description: string;
  canRetry: boolean;
  requiresSettings: boolean;
}

export interface LocationSample {
  lat: number;
  lng: number;
  accuracy: number;
  timestamp: number;
}

export interface LocationAcquisitionProgress {
  sampleCount: number;
  elapsedMs: number;
  latestAccuracy?: number;
  bestAccuracy?: number;
  latestFreshSample?: LocationSample;
  bestFreshSample?: LocationSample;
  requiredAccuracyMeters: number;
  status: "warming" | "excellent" | "acceptable" | "weak";
}

export interface LocationAcquisitionResult {
  position: GeolocationPosition;
  sampleCount: number;
  bestAccuracy: number;
  bestFreshSample: LocationSample;
  requiredAccuracyMeters: number;
  elapsedMs: number;
}

interface LocationAcquisitionOptions {
  timeoutMs: number;
  freshMaxAgeMs: number;
  excellentAccuracyMeters: number;
  requiredAccuracyMeters: number;
  minimumExcellentSamples: number;
  minimumAcceptableSamples: number;
  minimumWarmupMs: number;
}

type GeolocationError = Error & {
  code: number;
  accuracy?: number;
  requiredAccuracy?: number;
};

const GEOLOCATION_PERMISSION_DENIED = 1;
const GEOLOCATION_POSITION_UNAVAILABLE = 2;
const GEOLOCATION_TIMEOUT = 3;
const GEOLOCATION_UNSUPPORTED = 0;
const GEOLOCATION_INACCURATE = 4;

const DEFAULT_LOCATION_ACQUISITION_OPTIONS: LocationAcquisitionOptions = {
  // High-accuracy GNSS can need 25-30s for a cold first fix (indoor / weak
  // signal / battery saver). The browser watchPosition has no per-attempt
  // timeout (see below), so this is the single total budget; onProgress keeps
  // the wait feeling intentional rather than hung.
  timeoutMs: 30000,
  freshMaxAgeMs: 30000,
  excellentAccuracyMeters: 20,
  requiredAccuracyMeters: 50,
  minimumExcellentSamples: 2,
  minimumAcceptableSamples: 2,
  minimumWarmupMs: 3000,
};

export function isGeolocationError(error: unknown): boolean {
  const geolocationError = error as { code?: unknown };
  return (
    geolocationError?.code === GEOLOCATION_UNSUPPORTED ||
    geolocationError?.code === GEOLOCATION_PERMISSION_DENIED ||
    geolocationError?.code === GEOLOCATION_POSITION_UNAVAILABLE ||
    geolocationError?.code === GEOLOCATION_TIMEOUT ||
    geolocationError?.code === GEOLOCATION_INACCURATE
  );
}

export async function getLocationPermissionState(): Promise<LocationPermissionState> {
  if (!("permissions" in navigator) || !navigator.permissions?.query) {
    return "unknown";
  }

  try {
    const status = await navigator.permissions.query({ name: "geolocation" });
    return status.state;
  } catch {
    return "unknown";
  }
}

function getAccuracyStatus(
  accuracy: number | undefined,
  options: LocationAcquisitionOptions
): LocationAcquisitionProgress["status"] {
  if (typeof accuracy !== "number") return "warming";
  const excellentThreshold = Math.min(
    options.excellentAccuracyMeters,
    options.requiredAccuracyMeters
  );
  if (accuracy <= excellentThreshold) return "excellent";
  if (accuracy <= options.requiredAccuracyMeters) return "acceptable";
  return "weak";
}

function toLocationSample(position: GeolocationPosition): LocationSample {
  return {
    lat: position.coords.latitude,
    lng: position.coords.longitude,
    accuracy: position.coords.accuracy,
    timestamp: position.timestamp,
  };
}

export function requestBestCurrentLocation(
  onProgress?: (progress: LocationAcquisitionProgress) => void,
  optionOverrides?: Partial<LocationAcquisitionOptions>
): Promise<LocationAcquisitionResult> {
  const options = { ...DEFAULT_LOCATION_ACQUISITION_OPTIONS, ...optionOverrides };

  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) {
      reject(createGeolocationError(0, "Trình duyệt không hỗ trợ GPS"));
      return;
    }

    const startedAt = Date.now();
    let watchId: number | null = null;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    let warmupTimerId: ReturnType<typeof setTimeout> | null = null;
    let settled = false;
    let sampleCount = 0;
    let freshSampleCount = 0;
    let bestPosition: GeolocationPosition | null = null;

    const finish = (
      callback: () => void
    ) => {
      if (settled) return;
      settled = true;
      if (watchId !== null) {
        navigator.geolocation.clearWatch(watchId);
      }
      if (timeoutId !== null) {
        clearTimeout(timeoutId);
      }
      if (warmupTimerId !== null) {
        clearTimeout(warmupTimerId);
      }
      callback();
    };

    const emitProgress = (latestFreshPosition?: GeolocationPosition) => {
      const bestAccuracy = bestPosition?.coords.accuracy;
      onProgress?.({
        sampleCount,
        elapsedMs: Date.now() - startedAt,
        latestAccuracy: latestFreshPosition?.coords.accuracy,
        bestAccuracy,
        latestFreshSample: latestFreshPosition
          ? toLocationSample(latestFreshPosition)
          : undefined,
        bestFreshSample: bestPosition ? toLocationSample(bestPosition) : undefined,
        requiredAccuracyMeters: options.requiredAccuracyMeters,
        status: getAccuracyStatus(bestAccuracy, options),
      });
    };

    function resolveWithBestPosition() {
      if (!bestPosition) return;
      const bestAccuracy = bestPosition.coords.accuracy;
      finish(() =>
        resolve({
          position: bestPosition as GeolocationPosition,
          sampleCount,
          bestAccuracy,
          bestFreshSample: toLocationSample(bestPosition as GeolocationPosition),
          requiredAccuracyMeters: options.requiredAccuracyMeters,
          elapsedMs: Date.now() - startedAt,
        })
      );
    }

    function scheduleWarmupCheck(elapsedMs: number, bestAccuracy: number) {
      if (warmupTimerId !== null || elapsedMs >= options.minimumWarmupMs) return;
      if (bestAccuracy > options.requiredAccuracyMeters) return;

      warmupTimerId = setTimeout(() => {
        warmupTimerId = null;
        if (settled) return;
        emitProgress();
        maybeResolveWithBestPosition();
      }, options.minimumWarmupMs - elapsedMs);
    }

    function maybeResolveWithBestPosition() {
      if (!bestPosition || settled) return;

      const bestAccuracy = bestPosition.coords.accuracy;
      const elapsedMs = Date.now() - startedAt;
      const hasWarmedUp =
        freshSampleCount >= options.minimumExcellentSamples ||
        elapsedMs >= options.minimumWarmupMs;
      const isExcellent =
        bestAccuracy <= Math.min(
          options.excellentAccuracyMeters,
          options.requiredAccuracyMeters
        ) &&
        hasWarmedUp;
      const isAcceptable =
        bestAccuracy <= options.requiredAccuracyMeters &&
        (freshSampleCount >= options.minimumAcceptableSamples ||
          elapsedMs >= options.minimumWarmupMs);

      if (isExcellent || isAcceptable) {
        resolveWithBestPosition();
        return;
      }

      scheduleWarmupCheck(elapsedMs, bestAccuracy);
    }

    timeoutId = setTimeout(() => {
      finish(() => {
        if (bestPosition) {
          const bestAccuracy = bestPosition.coords.accuracy;
          if (bestAccuracy <= options.requiredAccuracyMeters) {
            resolve({
              position: bestPosition,
              sampleCount,
              bestAccuracy,
              bestFreshSample: toLocationSample(bestPosition),
              requiredAccuracyMeters: options.requiredAccuracyMeters,
              elapsedMs: Date.now() - startedAt,
            });
            return;
          }
          reject(createInaccurateGeolocationError(bestAccuracy, options.requiredAccuracyMeters));
          return;
        }
        reject(createGeolocationError(GEOLOCATION_TIMEOUT, "Lấy vị trí quá lâu"));
      });
    }, options.timeoutMs);

    try {
      watchId = navigator.geolocation.watchPosition(
        (position) => {
          if (settled) return;

          sampleCount += 1;
          const ageMs = Date.now() - position.timestamp;
          const isFresh = ageMs <= options.freshMaxAgeMs;
          if (isFresh) {
            freshSampleCount += 1;
          }
          if (
            isFresh &&
            (!bestPosition || position.coords.accuracy < bestPosition.coords.accuracy)
          ) {
            bestPosition = position;
          }

          emitProgress(isFresh ? position : undefined);
          maybeResolveWithBestPosition();
        },
        (error) => {
          if (settled) return;
          if (bestPosition && error.code !== GEOLOCATION_PERMISSION_DENIED) {
            return;
          }
          finish(() => reject(error));
        },
        {
          enableHighAccuracy: true,
          // Allow reusing a fix up to 15s old (e.g. the always-on preview's
          // already-acquired sample) instead of forcing a cold fix on every
          // tap. The accuracy gate still validates the coordinate server-side.
          maximumAge: 15000,
          // No per-attempt timeout: let the outer setTimeout(options.timeoutMs)
          // alone govern total wait. A browser-side timeout equal to the outer
          // budget pre-empted slow cold fixes (gps_timeout at exactly the cap),
          // killing devices that would have fixed a few seconds later.
        }
      );
    } catch (error) {
      finish(() => reject(error));
    }
  });
}

export function getLocationPermissionIssue(error: unknown): LocationPermissionIssue {
  const geolocationError = error as Partial<GeolocationPositionError> & {
    accuracy?: number;
    message?: string;
    requiredAccuracy?: number;
  };

  if (!navigator.geolocation) {
    return {
      type: "unsupported",
      title: "Thiết bị không hỗ trợ GPS",
      description: "Vui lòng dùng điện thoại hoặc trình duyệt có định vị để chấm công.",
      canRetry: false,
      requiresSettings: false,
    };
  }

  if (geolocationError.code === GEOLOCATION_PERMISSION_DENIED) {
    return {
      type: "denied",
      title: "Cho phép truy cập vị trí",
      description:
        "Bấm Thử lại và chọn Cho phép. Nếu trình duyệt vẫn chặn, mở Cài đặt vị trí và bật Vị trí chính xác.",
      canRetry: true,
      requiresSettings: true,
    };
  }

  if (geolocationError.code === GEOLOCATION_POSITION_UNAVAILABLE) {
    return {
      type: "unavailable",
      title: "Chưa bắt được GPS",
      description: "Bật GPS, tắt tiết kiệm pin, đứng ở nơi thoáng hơn rồi thử lại.",
      canRetry: true,
      requiresSettings: false,
    };
  }

  if (geolocationError.code === GEOLOCATION_TIMEOUT) {
    return {
      type: "timeout",
      title: "GPS phản hồi chậm",
      description: "Tín hiệu đang yếu. Đứng ở nơi thoáng hơn, giữ điện thoại yên vài giây rồi thử lại.",
      canRetry: true,
      requiresSettings: false,
    };
  }

  if (geolocationError.code === GEOLOCATION_INACCURATE) {
    return createPoorAccuracyLocationIssue(
      geolocationError.accuracy,
      geolocationError.requiredAccuracy
    );
  }

  return {
    type: "unknown",
    title: "Không thể lấy vị trí hiện tại",
    description: geolocationError.message || "Vui lòng kiểm tra quyền vị trí và thử lại.",
    canRetry: true,
    requiresSettings: false,
  };
}

export function isPoorLocationAccuracyMessage(message: string): boolean {
  return message.includes("Tín hiệu GPS không đủ chính xác");
}

export function createPoorAccuracyLocationIssue(
  accuracy?: number,
  requiredAccuracy?: number
): LocationPermissionIssue {
  const roundedAccuracy = typeof accuracy === "number" ? Math.round(accuracy) : null;
  const roundedRequiredAccuracy =
    typeof requiredAccuracy === "number" ? Math.round(requiredAccuracy) : null;
  const accuracyText =
    roundedAccuracy && roundedRequiredAccuracy
      ? `Sai số hiện khoảng ${roundedAccuracy}m, cần trong vòng ${roundedRequiredAccuracy}m.`
      : roundedAccuracy
        ? `Sai số hiện khoảng ${roundedAccuracy}m.`
        : "";
  const recoveryText = "Hãy đứng ở nơi thoáng hơn, giữ điện thoại yên vài giây rồi thử lại.";
  return {
    type: "inaccurate",
    title: "Chưa thể chấm công",
    description: [accuracyText, recoveryText].filter(Boolean).join(" "),
    canRetry: true,
    requiresSettings: false,
  };
}

export function createInaccurateGeolocationError(
  accuracy?: number,
  requiredAccuracy?: number
): GeolocationError {
  const error = createGeolocationError(
    GEOLOCATION_INACCURATE,
    "Tín hiệu GPS không đủ chính xác"
  );
  error.accuracy = accuracy;
  error.requiredAccuracy = requiredAccuracy;
  return error;
}

export function createGeolocationError(code: number, message: string): GeolocationError {
  const error = new Error(message) as GeolocationError;
  error.code = code;
  return error;
}

// ============================================================================
// Continuous acquisition (warm-up for instant tap-to-submit)
// ============================================================================
//
// requestBestCurrentLocation above is a ONE-SHOT: it resolves/rejects on the
// first good sample or the outer timeout. The check-in card needs a fix that is
// already warm when the worker taps, so the tap submits instantly instead of
// starting a cold 25-30s acquisition — the dominant cause of on-site check-in
// failures (the phone's first GNSS fix outlasts the budget, so the request never
// reaches the geofence).
//
// watchContinuousLocation keeps a single watchPosition running and exposes the
// latest fresh sample + rolling progress until unsubscribe(). The React hook
// (useContinuousLocation) owns start/stop and the tap-time submit decision: it
// reuses getCheckInGeofenceGuidance to know when a sample is "inside" the gate,
// mirroring the backend validateGeofence "certain" contract (dist+accuracy<=r).

/** Freshness window for the continuous watch (15s). Matches the watchPosition
 *  maximumAge so a browser-cached fix counts as fresh iff within this age.
 *  Intentionally tighter than the one-shot requestBestCurrentLocation's 30s
 *  freshMaxAgeMs: the continuous watch exposes the LATEST fresh sample (current
 *  position), so a tighter window keeps the submit candidate honest about where
 *  the worker is now. */
export const CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS = 15000;

export interface ContinuousLocationHandle {
  /** Stop the underlying watchPosition. Idempotent. */
  unsubscribe: () => void;
  /** Most recent sample within the freshness window, or null. This is the sample
   *  a tap should submit — it reflects the worker's current position. */
  getLatestFreshSample: () => LocationSample | null;
  /** Most recent progress snapshot, or null before the first update. */
  getProgress: () => LocationAcquisitionProgress | null;
}

export interface WatchContinuousLocationOptions {
  freshMaxAgeMs?: number;
  requiredAccuracyMeters?: number;
  excellentAccuracyMeters?: number;
  onUpdate?: (progress: LocationAcquisitionProgress) => void;
  /** Geolocation error callback. The hook classifies: permission-denied is fatal
   *  (unsubscribe + recovery banner); signal-loss/timeout are transient (ignore). */
  onError?: (error: GeolocationPositionError) => void;
}

/** Pure freshness check (boundary-inclusive). Exported for unit testing. */
export function isSampleFresh(
  sample: LocationSample,
  nowMs: number,
  maxAgeMs: number
): boolean {
  return nowMs - sample.timestamp <= maxAgeMs;
}

/**
 * Start a continuous high-accuracy watch and expose the latest fresh sample plus
 * rolling progress. Non-terminating: keeps the watch warm until unsubscribe().
 *
 * Submit candidate semantics: this watch exposes the LATEST fresh sample (current
 * position), not the lowest-accuracy reading. That is intentionally different
 * from requestBestCurrentLocation, which resolves on the best accuracy. For a
 * stationary worker the two converge in practice; for a moving worker, latest is
 * positionally honest — the server-side gate is the final backstop regardless.
 */
export function watchContinuousLocation(
  options?: WatchContinuousLocationOptions
): ContinuousLocationHandle {
  const freshMaxAgeMs = options?.freshMaxAgeMs ?? CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS;
  const requiredAccuracyMeters =
    options?.requiredAccuracyMeters ?? DEFAULT_LOCATION_ACQUISITION_OPTIONS.requiredAccuracyMeters;
  const excellentAccuracyMeters =
    options?.excellentAccuracyMeters ?? DEFAULT_LOCATION_ACQUISITION_OPTIONS.excellentAccuracyMeters;
  const classificationOptions = { excellentAccuracyMeters, requiredAccuracyMeters };

  const noopHandle: ContinuousLocationHandle = {
    unsubscribe: () => {},
    getLatestFreshSample: () => null,
    getProgress: () => null,
  };
  if (!navigator.geolocation) {
    return noopHandle;
  }

  const startedAt = Date.now();
  let watchId: number | null = null;
  let sampleCount = 0;
  let latestFreshSample: LocationSample | null = null;
  let bestAccuracy: number | undefined;
  let progress: LocationAcquisitionProgress | null = null;

  const emit = () => {
    progress = {
      sampleCount,
      elapsedMs: Date.now() - startedAt,
      latestAccuracy: latestFreshSample?.accuracy,
      bestAccuracy,
      latestFreshSample: latestFreshSample ?? undefined,
      // "best to submit" for the continuous watch == latest fresh sample
      // (current position). See function doc comment.
      bestFreshSample: latestFreshSample ?? undefined,
      requiredAccuracyMeters,
      status: getAccuracyStatus(bestAccuracy, classificationOptions),
    };
    options?.onUpdate?.(progress);
  };

  try {
    watchId = navigator.geolocation.watchPosition(
      (position) => {
        sampleCount += 1;
        const sample = toLocationSample(position);
        if (isSampleFresh(sample, Date.now(), freshMaxAgeMs)) {
          latestFreshSample = sample;
          if (bestAccuracy === undefined || sample.accuracy < bestAccuracy) {
            bestAccuracy = sample.accuracy;
          }
        }
        emit();
      },
      (error) => {
        // Transient errors (momentary signal loss) are non-fatal; the watch keeps
        // running and the next fix updates state. The hook decides what to surface
        // — permission-denied is fatal, everything else is ignorable noise.
        options?.onError?.(error);
      },
      {
        enableHighAccuracy: true,
        maximumAge: CONTINUOUS_LOCATION_FRESH_MAX_AGE_MS,
      }
    );
  } catch {
    return noopHandle;
  }

  return {
    unsubscribe: () => {
      if (watchId !== null && navigator.geolocation) {
        navigator.geolocation.clearWatch(watchId);
        watchId = null;
      }
    },
    getLatestFreshSample: () => latestFreshSample,
    getProgress: () => progress,
  };
}
