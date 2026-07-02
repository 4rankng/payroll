export type LocationPermissionState = PermissionState | "unsupported" | "unknown";

export interface LocationPermissionIssue {
  type: "unsupported" | "denied" | "unavailable" | "timeout" | "inaccurate" | "unknown";
  title: string;
  description: string;
  canRetry: boolean;
  requiresSettings: boolean;
}

export interface LocationAcquisitionProgress {
  sampleCount: number;
  elapsedMs: number;
  latestAccuracy?: number;
  bestAccuracy?: number;
  status: "warming" | "excellent" | "acceptable" | "weak";
}

export interface LocationAcquisitionResult {
  position: GeolocationPosition;
  sampleCount: number;
  bestAccuracy: number;
  elapsedMs: number;
}

interface LocationAcquisitionOptions {
  timeoutMs: number;
  freshMaxAgeMs: number;
  excellentAccuracyMeters: number;
  acceptableAccuracyMeters: number;
  minimumAcceptableSamples: number;
  minimumWarmupMs: number;
}

const GEOLOCATION_PERMISSION_DENIED = 1;
const GEOLOCATION_POSITION_UNAVAILABLE = 2;
const GEOLOCATION_TIMEOUT = 3;
const GEOLOCATION_UNSUPPORTED = 0;

const DEFAULT_LOCATION_ACQUISITION_OPTIONS: LocationAcquisitionOptions = {
  timeoutMs: 20000,
  freshMaxAgeMs: 30000,
  excellentAccuracyMeters: 20,
  acceptableAccuracyMeters: 50,
  minimumAcceptableSamples: 2,
  minimumWarmupMs: 3000,
};

export function isGeolocationError(error: unknown): boolean {
  const geolocationError = error as { code?: unknown };
  return (
    geolocationError?.code === GEOLOCATION_UNSUPPORTED ||
    geolocationError?.code === GEOLOCATION_PERMISSION_DENIED ||
    geolocationError?.code === GEOLOCATION_POSITION_UNAVAILABLE ||
    geolocationError?.code === GEOLOCATION_TIMEOUT
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

export function requestCurrentLocation(): Promise<GeolocationPosition> {
  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) {
      reject(createGeolocationError(0, "Trình duyệt không hỗ trợ GPS"));
      return;
    }

    navigator.geolocation.getCurrentPosition(resolve, reject, {
      enableHighAccuracy: true,
      timeout: 10000,
      maximumAge: 0,
    });
  });
}

function getAccuracyStatus(
  accuracy: number | undefined,
  options: LocationAcquisitionOptions
): LocationAcquisitionProgress["status"] {
  if (typeof accuracy !== "number") return "warming";
  if (accuracy <= options.excellentAccuracyMeters) return "excellent";
  if (accuracy <= options.acceptableAccuracyMeters) return "acceptable";
  return "weak";
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

    const emitProgress = (latestPosition?: GeolocationPosition) => {
      const bestAccuracy = bestPosition?.coords.accuracy;
      onProgress?.({
        sampleCount,
        elapsedMs: Date.now() - startedAt,
        latestAccuracy: latestPosition?.coords.accuracy,
        bestAccuracy,
        status: getAccuracyStatus(bestAccuracy, options),
      });
    };

    timeoutId = setTimeout(() => {
      finish(() => {
        if (bestPosition) {
          resolve({
            position: bestPosition,
            sampleCount,
            bestAccuracy: bestPosition.coords.accuracy,
            elapsedMs: Date.now() - startedAt,
          });
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
          if (
            isFresh &&
            (!bestPosition || position.coords.accuracy < bestPosition.coords.accuracy)
          ) {
            bestPosition = position;
          }

          emitProgress(position);

          if (!bestPosition) return;

          const bestAccuracy = bestPosition.coords.accuracy;
          const elapsedMs = Date.now() - startedAt;
          const isExcellent = bestAccuracy <= options.excellentAccuracyMeters;
          const isAcceptable =
            bestAccuracy <= options.acceptableAccuracyMeters &&
            (sampleCount >= options.minimumAcceptableSamples ||
              elapsedMs >= options.minimumWarmupMs);

          if (isExcellent || isAcceptable) {
            finish(() =>
              resolve({
                position: bestPosition as GeolocationPosition,
                sampleCount,
                bestAccuracy,
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
  const geolocationError = error as Partial<GeolocationPositionError> & { message?: string };

  if (!navigator.geolocation) {
    return {
      type: "unsupported",
      title: "Thiết bị không hỗ trợ GPS",
      description: "Vui lòng dùng điện thoại hoặc trình duyệt có hỗ trợ định vị để chấm công.",
      canRetry: false,
      requiresSettings: false,
    };
  }

  if (geolocationError.code === GEOLOCATION_PERMISSION_DENIED) {
    return {
      type: "denied",
      title: "Bật quyền vị trí",
      description:
        "Bấm Thử lại. Nếu vẫn bị chặn, mở Cài đặt trình duyệt, bật quyền vị trí và chọn vị trí chính xác.",
      canRetry: true,
      requiresSettings: true,
    };
  }

  if (geolocationError.code === GEOLOCATION_POSITION_UNAVAILABLE) {
    return {
      type: "unavailable",
      title: "Chưa lấy được vị trí",
      description: "Vui lòng bật GPS, tắt tiết kiệm pin, đứng ở nơi thoáng hơn rồi thử lại.",
      canRetry: true,
      requiresSettings: false,
    };
  }

  if (geolocationError.code === GEOLOCATION_TIMEOUT) {
    return {
      type: "timeout",
      title: "Lấy vị trí quá lâu",
      description: "Vui lòng bước ra ngoài trời, bật vị trí chính xác/GPS độ chính xác cao và chờ vài giây trước khi thử lại.",
      canRetry: true,
      requiresSettings: false,
    };
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

export function createPoorAccuracyLocationIssue(accuracy?: number): LocationPermissionIssue {
  const roundedAccuracy = typeof accuracy === "number" ? Math.round(accuracy) : null;
  const accuracyText = roundedAccuracy ? ` Độ chính xác vừa đo khoảng ${roundedAccuracy}m.` : "";
  return {
    type: "inaccurate",
    title: "Vị trí chưa đủ chính xác",
    description: `${accuracyText} Vui lòng bước ra ngoài trời, bật vị trí chính xác/GPS độ chính xác cao, tắt tiết kiệm pin và chờ vài giây rồi thử lại.`.trim(),
    canRetry: true,
    requiresSettings: false,
  };
}

export function createGeolocationError(code: number, message: string): Error & { code: number } {
  const error = new Error(message) as Error & { code: number };
  error.code = code;
  return error;
}
