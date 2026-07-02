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
  timeoutMs: 20000,
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

          if (!bestPosition) return;

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
            finish(() =>
              resolve({
                position: bestPosition as GeolocationPosition,
                sampleCount,
                bestAccuracy,
                bestFreshSample: toLocationSample(bestPosition as GeolocationPosition),
                requiredAccuracyMeters: options.requiredAccuracyMeters,
                elapsedMs,
              })
            );
          }
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
          maximumAge: 0,
          timeout: options.timeoutMs,
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

function createInaccurateGeolocationError(
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
