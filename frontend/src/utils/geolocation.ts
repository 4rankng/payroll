export type LocationPermissionState = PermissionState | "unsupported" | "unknown";

export interface LocationPermissionIssue {
  type: "unsupported" | "denied" | "unavailable" | "timeout" | "unknown";
  title: string;
  description: string;
  canRetry: boolean;
  requiresSettings: boolean;
}

const GEOLOCATION_PERMISSION_DENIED = 1;
const GEOLOCATION_POSITION_UNAVAILABLE = 2;
const GEOLOCATION_TIMEOUT = 3;
const GEOLOCATION_UNSUPPORTED = 0;

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
        "Bấm Thử lại. Nếu vẫn bị chặn, mở Cài đặt > Safari Websites > Vị trí.",
      canRetry: true,
      requiresSettings: true,
    };
  }

  if (geolocationError.code === GEOLOCATION_POSITION_UNAVAILABLE) {
    return {
      type: "unavailable",
      title: "Chưa lấy được vị trí",
      description: "Vui lòng bật GPS, kiểm tra kết nối mạng rồi thử lại.",
      canRetry: true,
      requiresSettings: false,
    };
  }

  if (geolocationError.code === GEOLOCATION_TIMEOUT) {
    return {
      type: "timeout",
      title: "Lấy vị trí quá lâu",
      description: "Vui lòng đứng ở nơi thoáng hơn hoặc kiểm tra GPS rồi thử lại.",
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

export function createGeolocationError(code: number, message: string): Error & { code: number } {
  const error = new Error(message) as Error & { code: number };
  error.code = code;
  return error;
}
