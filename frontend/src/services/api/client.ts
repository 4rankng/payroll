import axios, { AxiosInstance, AxiosError, AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { authManager } from '@/lib/auth';
import { API_CONFIG, RETRY_CONFIG } from '@/config/api.config';
import { ERROR_MESSAGES } from '@/config/constants';
import { toast } from '@/components/ui/sonner';
import { getErrorMessage } from '@/utils/error-handler';
import { extractFilenameFromHeaders, triggerBlobDownload } from '@/utils/file-download';

// API Response types
export interface ApiResponse<T = unknown> {
  status: 'success' | 'error';
  data?: T;
  message?: string;
  http_status?: number;
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface ApiError {
  status: 'error';
  message: string;
  http_status: number;
  code?: string;
  details?: Record<string, unknown>;
  retry_after?: number;
}

export function createIdempotencyKey(scope: string): string {
  const randomID =
    typeof crypto !== 'undefined' && 'randomUUID' in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  return `${scope}:${randomID}`;
}

class ApiClient {
  private client: AxiosInstance;
  // BUG-005 fix: Global mutex to prevent concurrent 401 handling
  private isHandling401 = false;
  private handle401Promise: Promise<void> | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: `${API_CONFIG.baseURL}${API_CONFIG.apiV1}`,
      timeout: API_CONFIG.timeout,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.setupInterceptors();
  }

  private async delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  private calculateRetryDelay(retryCount: number): number {
    const delay = Math.min(
      RETRY_CONFIG.baseDelay * Math.pow(RETRY_CONFIG.backoffFactor, retryCount),
      RETRY_CONFIG.maxDelay
    );
    // Add jitter to avoid thundering herd
    return delay + Math.random() * 1000;
  }

  private shouldRetry(error: AxiosError, retryCount: number, method?: string): boolean {
    if (retryCount >= RETRY_CONFIG.maxRetries) {
      return false;
    }

    // Only retry safe reads automatically. Retrying side-effecting requests
    // can repeat writes if the server completed the action but the response was
    // lost or timed out.
    const normalizedMethod = method?.toUpperCase() || 'GET';
    if (!['GET', 'HEAD', 'OPTIONS'].includes(normalizedMethod)) {
      return false;
    }

    // Check for network errors
    if (!error.response) {
      const networkError = error.code;
      return networkError ? RETRY_CONFIG.retryableErrors.includes(networkError) : true;
    }

    // Check for retryable status codes
    return RETRY_CONFIG.retryableStatuses.includes(error.response.status);
  }

  private setupInterceptors() {
    // Request interceptor
    this.client.interceptors.request.use(
      (config: InternalAxiosRequestConfig) => {
        const token = authManager.getToken();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => {
        if (import.meta.env.DEV) {
          console.error('❌ Request Error:', error);
        }
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => {
        // Successful response
        return response;
      },
      async (error: AxiosError<ApiError>) => {
        const originalRequest = error.config as AxiosRequestConfig & {
          _retry?: boolean;
          _retryCount?: number;
        };

        // Initialize retry count
        if (!originalRequest._retryCount) {
          originalRequest._retryCount = 0;
        }

        // Development error logging
        if (import.meta.env.DEV) {
          console.group(`❌ API Error: ${originalRequest?.method?.toUpperCase()} ${originalRequest?.url}`);
          console.groupEnd();
        }

        // Handle 429 Too Many Requests (never retry, immediate rejection)
        if (error.response?.status === 429) {
          const retryAfter = error.response.data?.retry_after || 60;
          const rateLimitError: ApiError = {
            status: 'error',
            message: `Quá nhiều yêu cầu. Vui lòng thử lại sau ${retryAfter} giây`,
            http_status: 429,
            retry_after: retryAfter,
            details: { retryAfter },
          };
          return Promise.reject(rateLimitError);
        }

        // Handle 401 Unauthorized (don't retry auth errors)
        // BUG-005 fix: Use global mutex to prevent concurrent 401 handling
        if (error.response?.status === 401 && !originalRequest._retry) {
          originalRequest._retry = true;

          // The login endpoint also returns 401 for bad credentials — but in
          // that case the user clearly wasn't logged in, so showing
          // "Phiên làm việc hết hạn" / clearing the token / redirecting is
          // both noisy and incorrect. Let the login form surface its own
          // error message and bail out of the global handler.
          const requestUrl: string = originalRequest?.url ?? '';
          const isLoginRequest = requestUrl.includes('/auth/login');
          if (isLoginRequest) {
            return Promise.reject(error.response?.data || error);
          }

          // If already handling a 401, wait for it to complete then reject
          if (this.isHandling401) {
            if (this.handle401Promise) {
              await this.handle401Promise;
            }
            return Promise.reject(error.response?.data || error);
          }

          this.isHandling401 = true;
          this.handle401Promise = (async () => {
            // All 401s from backend mean authentication failure (invalid/missing/expired/blacklisted token).
            // Always logout — permission issues use 403, not 401.
            toast({
              variant: "destructive",
              title: "Phiên làm việc hết hạn",
              description: "Phiên đăng nhập của bạn đã hết hạn. Vui lòng đăng nhập lại.",
              duration: 5000,
            });

            authManager.removeToken();
            // Only redirect if not already on login page to prevent infinite loops
            if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
              setTimeout(() => {
                window.location.href = '/login';
              }, 1000);
            }
          })();

          try {
            await this.handle401Promise;
          } finally {
            this.isHandling401 = false;
            this.handle401Promise = null;
          }

          return Promise.reject(error.response?.data || error);
        }

        // Handle 403 Forbidden errors — do NOT show a toast here.
        // MutationCache.onError / QueryCache.onError in App.tsx is the single
        // source of truth for error notifications, preventing duplicate toasts.
        if (error.response?.status === 403) {
          return Promise.reject(error.response.data);
        }

        // Check if we should retry this request
        if (this.shouldRetry(error, originalRequest._retryCount, originalRequest.method)) {
          originalRequest._retryCount++;

          const delayMs = this.calculateRetryDelay(originalRequest._retryCount - 1);

          await this.delay(delayMs);
          return this.client(originalRequest);
        }

        // Handle network errors (after retries exhausted)
        if (!error.response) {
          const networkError: ApiError = {
            status: 'error',
            message: ERROR_MESSAGES.NETWORK_ERROR,
            http_status: 0,
          };
          return Promise.reject(networkError);
        }

        // Handle other errors
        return Promise.reject(error.response?.data || error);
      }
    );
  }

  // HTTP Methods
  private normalizePayload<T>(response: { data: ApiResponse<T>; status: number }): ApiResponse<T> {
    const payload = response.data as ApiResponse<T>;
    if (payload && typeof payload === 'object' && payload.http_status === undefined) {
      payload.http_status = response.status;
    }
    return payload;
  }

  async get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.client.get<ApiResponse<T>>(url, config);
    return this.normalizePayload(response);
  }

  async post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.client.post<ApiResponse<T>>(url, data, config);
    return this.normalizePayload(response);
  }

  async put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.client.put<ApiResponse<T>>(url, data, config);
    return this.normalizePayload(response);
  }

  async patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.client.patch<ApiResponse<T>>(url, data, config);
    return this.normalizePayload(response);
  }

  async delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    const response = await this.client.delete<ApiResponse<T>>(url, config);
    return this.normalizePayload(response);
  }

  // File upload
  async upload<T = unknown>(url: string, formData: FormData, onProgress?: (progress: number) => void): Promise<ApiResponse<T>> {
    const response = await this.client.post<ApiResponse<T>>(url, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const progress = Math.round((progressEvent.loaded * 100) / progressEvent.total);
          onProgress(progress);
        }
      },
    });
    return this.normalizePayload(response);
  }

  // File download
  async download(url: string, filename?: string): Promise<void> {
    const response = await this.client.get(url, {
      responseType: 'blob',
    }).catch(this.handleBlobError.bind(this));

    const headerFilename = extractFilenameFromHeaders(response.headers);
    triggerBlobDownload(response.data, headerFilename || filename || 'download');
  }

  // File download via POST request
  async downloadPost(url: string, data?: unknown, filename?: string): Promise<void> {
    const response = await this.client.post(url, data, {
      responseType: 'blob',
    }).catch(this.handleBlobError.bind(this));

    const headerFilename = extractFilenameFromHeaders(response.headers);
    triggerBlobDownload(response.data, headerFilename || filename || 'download');
  }

  // Raw blob download for external use
  async downloadBlob(url: string): Promise<Blob> {
    const response = await this.client.get(url, {
      responseType: 'blob',
    }).catch(this.handleBlobError.bind(this));
    return response.data;
  }

  // Raw request method for endpoints that don't use the standard API wrapper
  async rawRequest<T = unknown>(config: AxiosRequestConfig): Promise<T> {
    const response = await this.client.request<T>(config);
    return response.data;
  }

  // ─── Private helpers ──────────────────────────────────────────────────────

  /**
   * Error handler for blob requests. The response interceptor unwraps
   * AxiosError → error.response.data (which is a Blob for responseType:'blob').
   * Some interceptor paths (429, network errors) reject with non-Blob values,
   * so both cases must be handled.
   */
  private async handleBlobError(error: unknown): Promise<never> {
    const blob = error instanceof Blob
      ? error
      : (error as AxiosError)?.response?.data;
    if (blob instanceof Blob && blob.type.includes('application/json')) {
      try {
        const text = await blob.text();
        const parsed = JSON.parse(text);
        return Promise.reject(parsed);
      } catch {
        // fall through to original rejection
      }
    }
    return Promise.reject(error);
  }
}

// Export singleton instance
export const apiClient = new ApiClient();

// Helper function to handle API errors
export const handleApiError = (error: unknown): string => {
  return getErrorMessage(error);
};

// Helper function to build query string
export const buildQueryString = <T extends object>(params: T): string => {
  const query = new URLSearchParams();

  Object.entries(params as Record<string, unknown>).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      // Handle arrays by adding multiple query parameters
      if (Array.isArray(value)) {
        value.forEach((item) => {
          if (item !== undefined && item !== null && item !== '') {
            query.append(key, String(item));
          }
        });
      } else {
        query.append(key, String(value));
      }
    }
  });
  const queryString = query.toString();
  return queryString ? `?${queryString}` : '';
};
