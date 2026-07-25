import { toast } from '@/components/ui/sonner';

/**
 * Centralized error handling utility for the entire application
 * Provides consistent error titles and messages for backend errors
 */

export interface ErrorDetails {
  title: string;
  message: string;
  status?: number;
}

export interface AppError {
  message: string;
  code?: string;
  statusCode?: number;
  details?: Record<string, unknown>;
}

interface ApiErrorResponse {
  status: 'error';
  message?: string;
  http_status?: number;
  retry_after?: number;
}

interface AxiosErrorResponse {
  response?: {
    data?: {
      message?: string;
      error?: string;
    };
    status?: number;
  };
  message?: string;
}

interface NetworkError {
  code?: string;
  response?: unknown;
}

function isApiErrorResponse(error: unknown): error is ApiErrorResponse {
  return (
    error !== null &&
    typeof error === 'object' &&
    'status' in error &&
    error.status === 'error'
  );
}

function isAxiosErrorResponse(error: unknown): error is AxiosErrorResponse {
  return (
    error !== null &&
    typeof error === 'object' &&
    'response' in error
  );
}

function hasMessage(error: unknown): error is { message: string } {
  return (
    error !== null &&
    typeof error === 'object' &&
    'message' in error &&
    typeof (error as { message: unknown }).message === 'string'
  );
}

function isNetworkErrorType(error: unknown): error is NetworkError {
  return (
    error !== null &&
    typeof error === 'object'
  );
}

/**
 * Context-aware error messages for different domains
 */
export const ERROR_MESSAGES_BY_CONTEXT = {
  PROJECT_EMPLOYEE: {
    REMOVE_FAILED: 'Không thể gỡ nhân viên khỏi dự án',
    ASSIGN_FAILED: 'Không thể giao nhân viên vào dự án',
    UPDATE_FAILED: 'Không thể cập nhật thông tin phân công',
    FETCH_FAILED: 'Không thể tải danh sách nhân viên',
    VALIDATION_FAILED: 'Dữ liệu không hợp lệ',
    VALIDATION_ERROR: 'Dữ liệu không hợp lệ',
    NETWORK_ERROR: 'Lỗi kết nối mạng',
    SERVER_ERROR: 'Lỗi máy chủ',
    UNKNOWN_ERROR: 'Có lỗi không xác định xảy ra',
    PERMISSION_DENIED: 'Bạn không có quyền thực hiện thao tác này',
  },
  GENERAL: {
    UNKNOWN_ERROR: 'Có lỗi không xác định xảy ra',
    NETWORK_ERROR: 'Lỗi kết nối mạng',
    SERVER_ERROR: 'Lỗi máy chủ',
    VALIDATION_ERROR: 'Dữ liệu không hợp lệ',
    VALIDATION_FAILED: 'Dữ liệu không hợp lệ',
    PERMISSION_DENIED: 'Bạn không có quyền thực hiện thao tác này',
  },
} as const;

export type ErrorContext = keyof typeof ERROR_MESSAGES_BY_CONTEXT;

/**
 * Creates a standardized error object from various error types
 */
export function createAppError(error: unknown): AppError {
  if (error instanceof Error) {
    const err = error as Error & { code?: string; statusCode?: number; details?: Record<string, unknown> };
    return {
      message: err.message,
      code: err.code,
      statusCode: err.statusCode,
      details: err.details,
    };
  }

  if (typeof error === 'string') {
    return {
      message: error,
    };
  }

  if (error && typeof error === 'object') {
    const errorObj = error as Record<string, unknown>;
    const message = (errorObj.message as string) ||
                   (errorObj.error as string) ||
                   'Có lỗi xảy ra';
    const statusCode = (errorObj.http_status as number) ||
                      (errorObj.statusCode as number) ||
                      (errorObj.status_code as number);

    return {
      message,
      code: errorObj.code as string,
      statusCode,
      details: errorObj.details as Record<string, unknown>,
    };
  }

  return {
    message: 'Có lỗi không xác định xảy ra',
  };
}

/**
 * Maps HTTP status codes to user-friendly messages with context awareness
 */
export function getErrorMessageByStatusCode(statusCode: number, context: ErrorContext = 'GENERAL'): string {
  const messages = (ERROR_MESSAGES_BY_CONTEXT[context] || ERROR_MESSAGES_BY_CONTEXT.GENERAL) as typeof ERROR_MESSAGES_BY_CONTEXT.GENERAL;

  switch (statusCode) {
    case 400:
      return messages.VALIDATION_FAILED || messages.VALIDATION_ERROR;
    case 401:
    case 403:
      return messages.PERMISSION_DENIED || 'Không có quyền truy cập';
    case 404:
      return 'Không tìm thấy tài nguyên';
    case 409:
      return 'Xung đột dữ liệu';
    case 422:
      return messages.VALIDATION_FAILED || messages.VALIDATION_ERROR;
    case 429:
      return 'Quá nhiều yêu cầu, vui lòng thử lại sau';
    case 500:
    case 502:
    case 503:
    case 504:
      return messages.SERVER_ERROR || 'Lỗi máy chủ';
    default:
      if (statusCode >= 500) {
        return messages.SERVER_ERROR || 'Lỗi máy chủ';
      }
      if (statusCode >= 400) {
        return messages.VALIDATION_ERROR || 'Yêu cầu không hợp lệ';
      }
      return messages.UNKNOWN_ERROR;
  }
}

/**
 * Creates a user-friendly error message with context
 */
export function createErrorMessage(
  error: unknown,
  context: ErrorContext = 'GENERAL',
  fallbackMessage?: string
): string {
  const appError = createAppError(error);

  // Always prioritize the backend message if it exists and is meaningful
  if (appError.message && appError.message !== 'Có lỗi xảy ra' && appError.message.trim().length > 0) {
    return appError.message;
  }

  if (appError.statusCode) {
    return getErrorMessageByStatusCode(appError.statusCode, context);
  }

  if (appError.code) {
    switch (appError.code) {
      case 'NETWORK_ERROR':
        return (ERROR_MESSAGES_BY_CONTEXT[context] as typeof ERROR_MESSAGES_BY_CONTEXT.GENERAL | undefined)?.NETWORK_ERROR || ERROR_MESSAGES_BY_CONTEXT.GENERAL.NETWORK_ERROR;
      case 'VALIDATION_ERROR':
        return (ERROR_MESSAGES_BY_CONTEXT[context] as typeof ERROR_MESSAGES_BY_CONTEXT.GENERAL | undefined)?.VALIDATION_FAILED || ERROR_MESSAGES_BY_CONTEXT.GENERAL.VALIDATION_ERROR;
      default:
        break;
    }
  }

  return fallbackMessage || (ERROR_MESSAGES_BY_CONTEXT[context] as typeof ERROR_MESSAGES_BY_CONTEXT.GENERAL | undefined)?.UNKNOWN_ERROR || ERROR_MESSAGES_BY_CONTEXT.GENERAL.UNKNOWN_ERROR;
}

/**
 * Type guard to check if an error is a validation error
 */
export function isValidationError(error: unknown): boolean {
  const appError = createAppError(error);
  return appError.statusCode === 400 ||
         appError.statusCode === 422 ||
         appError.code === 'VALIDATION_ERROR';
}

/**
 * Type guard to check if an error is a permission error
 */
export function isPermissionError(error: unknown): boolean {
  const appError = createAppError(error);
  return appError.statusCode === 401 ||
         appError.statusCode === 403 ||
         appError.code === 'PERMISSION_DENIED';
}

/**
 * Maps HTTP status codes to Vietnamese error titles
 */
export const getErrorTitle = (statusCode?: number): string => {
  if (!statusCode) return 'Lỗi hệ thống';

  switch (statusCode) {
    case 400:
      return 'Dữ liệu không hợp lệ';
    case 401:
      return 'Không có quyền truy cập';
    case 403:
      return 'Truy cập bị từ chối';
    case 404:
      return 'Không tìm thấy';
    case 409:
      return 'Xung đột dữ liệu';
    case 422:
      return 'Dữ liệu không đúng định dạng';
    case 429:
      return 'Quá nhiều yêu cầu';
    case 500:
    case 502:
    case 503:
    case 504:
      return 'Lỗi máy chủ';
    default:
      return statusCode >= 500 ? 'Lỗi máy chủ' : 'Lỗi hệ thống';
  }
};

/**
 * Extracts error message from various error response formats
 */
export const getErrorMessage = (error: unknown): string => {
  // Handle null/undefined
  if (!error) {
    return 'Có lỗi không xác định xảy ra';
  }

  // Handle structured API error response (from client.ts)
  if (isApiErrorResponse(error)) {
    const apiError = error;

    // Handle rate limiting with retry_after — use the server's factual remaining
    // seconds (computed from X-RateLimit-Reset, not a hardcoded 60).
    if (apiError.http_status === 429) {
      const retryAfter = apiError.retry_after || 60;
      const baseMsg = apiError.message || 'Quá nhiều yêu cầu. Vui lòng thử lại sau';
      // The apiClient 429 interceptor already constructs the canonical
      // "...sau N giây" message, and a server may also send a fully-formed
      // sentence. If the duration is already present ("giây"), return verbatim
      // to avoid producing "…sau 60 giây (thử lại sau 60 giây)".
      if (baseMsg.includes('giây')) {
        return baseMsg;
      }
      // If the server message ends with "sau", append the seconds; else
      // build the full sentence in parens.
      if (baseMsg.endsWith('sau')) {
        return `${baseMsg} ${retryAfter} giây`;
      }
      return `${baseMsg} (thử lại sau ${retryAfter} giây)`;
    }

    // If no message, return based on HTTP status code
    if (!apiError.message) {
      const statusCode = apiError.http_status;
      return statusCode >= 200 && statusCode < 300 ? 'Thành công' : 'Thất bại';
    }

    return apiError.message;
  }

  // Handle axios error response
  if (isAxiosErrorResponse(error)) {
    const axiosError = error;
    const responseData = axiosError.response?.data;
    const statusCode = axiosError.response?.status;

    if (responseData) {
      // Handle structured error response: { status: "error", message: "...", http_status: 400 }
      if (responseData.message) {
        return responseData.message;
      }
      // Handle alternative error field
      if (responseData.error) {
        return responseData.error;
      }

      // If no message/error field, return based on HTTP status code
      if (statusCode) {
        return statusCode >= 200 && statusCode < 300 ? 'Thành công' : 'Thất bại';
      }
    }

    // Fallback to axios error message
    if (axiosError.message) {
      return axiosError.message;
    }

    // If no message but we have status code, return based on it
    if (statusCode) {
      return statusCode >= 200 && statusCode < 300 ? 'Thành công' : 'Thất bại';
    }
  }

  // Handle direct error objects
  if (hasMessage(error)) {
    return error.message;
  }

  // Handle Error instances
  if (error instanceof Error) {
    return error.message;
  }

  // Handle string errors
  if (typeof error === 'string') {
    return error;
  }

  // Fallback
  return 'Có lỗi không xác định xảy ra';
};

/**
 * Extracts HTTP status code from various error formats
 */
export const getErrorStatusCode = (error: unknown): number | undefined => {
  if (!error || typeof error !== 'object') return undefined;

  // Handle structured API error response
  if (isApiErrorResponse(error)) {
    return error.http_status;
  }

  // Handle axios error response
  if (isAxiosErrorResponse(error) && error.response?.status) {
    return error.response.status;
  }

  return undefined;
};

/**
 * Gets complete error details from any error format
 */
export const getErrorDetails = (error: unknown): ErrorDetails => {
  const statusCode = getErrorStatusCode(error);
  const title = getErrorTitle(statusCode);
  const message = getErrorMessage(error);

  return {
    title,
    message,
    status: statusCode,
  };
};

/**
 * Shows error notification using toast system
 * This is the main function to use for displaying errors to users
 */
export const showErrorNotification = (error: unknown, customTitle?: string): void => {
  const details = getErrorDetails(error);
  const title = customTitle || details.title || 'Thất bại';
  const description = details.message;
  const duration = description.length <= 80 ? 3000 : 6000;
  toast({ title, description, duration });
};

/**
 * Gets success message from response or fallback to default
 */
export const getSuccessMessage = (response: unknown): string => {
  // Handle null/undefined
  if (!response) {
    return 'Thành công';
  }

  // Handle structured API response
  if (response && typeof response === 'object') {
    const apiResponse = response as Record<string, unknown>;

    // Check for message field
    if (typeof apiResponse.message === 'string') {
      return apiResponse.message;
    }

    // Check for data.message
    if (apiResponse.data && typeof apiResponse.data === 'object' && 'message' in apiResponse.data && typeof apiResponse.data.message === 'string') {
      return apiResponse.data.message;
    }
  }

  // Fallback to default success message
  return 'Thành công';
};

/**
 * Shows success notification
 */
export const showSuccessNotification = (messageOrResponse: string | unknown): void => {
  const message = typeof messageOrResponse === 'string'
    ? messageOrResponse
    : getSuccessMessage(messageOrResponse);

  toast('Thành công', { description: message, duration: 3000 });
};

/**
 * Shows notification for bulk operations with detailed results
 */
export const showBulkOperationNotification = (result: {
  total_created?: number;
  total_failed?: number;
  total_updated?: number;
  total_deleted?: number;
  total_approved?: number;
  total_rejected?: number;
  approved?: number;
  skipped?: number;
  failed?: number;
  errors?: string[];
}, operationType: 'create' | 'update' | 'delete' | 'approve' | 'reject' | 'reset' | 'import'): void => {
  const {
    total_created = 0,
    total_failed = 0,
    total_updated = 0,
    total_deleted = 0,
    total_approved = 0,
    total_rejected = 0,
    approved = 0,
    skipped = 0,
    failed = 0,
    errors = []
  } = result;

  const hasFailures = total_failed > 0 || failed > 0 || errors.length > 0;
  const hasSkipped = skipped > 0;
  const verbMap: Record<typeof operationType, string> = {
    create: 'tạo',
    update: 'cập nhật',
    delete: 'xóa',
    approve: 'phê duyệt',
    reject: 'loại',
    reset: 'khôi phục trạng thái chờ duyệt',
    import: 'nhập'
  };
  let affected = 0;
  switch (operationType) {
    case 'create': affected = total_created; break;
    case 'update': affected = total_updated; break;
    case 'delete': affected = total_deleted; break;
    case 'approve': affected = total_approved || approved; break;
    case 'reject': affected = total_rejected; break;
    case 'reset': affected = approved; break;
    case 'import': affected = total_created + total_updated; break;
  }

  const title = hasFailures ? 'Hoàn tất (một số lỗi)' : 'Thành công';

  let description = '';
  if (hasFailures && hasSkipped) {
    description = `Đã ${verbMap[operationType]} ${affected} bản ghi, bỏ qua ${skipped} bản đã phê duyệt, ${total_failed || failed} thất bại`;
  } else if (hasFailures) {
    description = `Đã ${verbMap[operationType]} ${affected} bản ghi, ${total_failed || failed} thất bại`;
  } else if (hasSkipped) {
    description = `Đã ${verbMap[operationType]} ${affected} bản ghi, bỏ qua ${skipped} bản đã phê duyệt`;
  } else {
    description = `Đã ${verbMap[operationType]} ${affected} bản ghi`;
  }

  toast(title, { description, duration: 3000 });
};

/**
 * Network error handler for specific network-related errors
 */
export const isNetworkError = (error: unknown): boolean => {
  if (!isNetworkErrorType(error)) return false;

  // Check for common network error indicators
  const errorCode = error.code;
  const networkErrors = ['ECONNABORTED', 'ECONNRESET', 'ECONNREFUSED', 'ENETDOWN', 'ENETUNREACH', 'EHOSTDOWN', 'EHOSTUNREACH', 'EPIPE'];

  if (errorCode && networkErrors.includes(errorCode)) return true;

  // Check if it's a network error without response
  if ('response' in error && !error.response) return true;

  return false;
};

/**
 * Handles network errors specifically
 */
export const handleNetworkError = (): void => {
  toast('Lỗi kết nối', { description: 'Không thể kết nối tới máy chủ. Vui lòng thử lại.', duration: 3000 });
};
