import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

let responseErrorHandler: ((error: unknown) => Promise<unknown>) | undefined;

const retryRequest = vi.fn();
const axiosCreate = vi.fn(() => mockAxiosInstance);

const mockAxiosInstance = Object.assign(retryRequest, {
  interceptors: {
    request: {
      use: vi.fn(),
    },
    response: {
      use: vi.fn((_onFulfilled, onRejected) => {
        responseErrorHandler = onRejected;
      }),
    },
  },
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
});

vi.mock('axios', () => ({
  default: {
    create: axiosCreate,
  },
}));

describe('ApiClient retry policy', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    vi.resetModules();
    responseErrorHandler = undefined;
    retryRequest.mockResolvedValue({ data: { status: 'success' }, status: 200 });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not retry side-effecting POST requests after an ambiguous network error', async () => {
    await import('@/services/api/client');

    await expect(
      responseErrorHandler?.({
        config: {
          method: 'post',
          url: '/timesheets/payroll/report/send-email',
        },
        code: 'ECONNABORTED',
      }),
    ).rejects.toMatchObject({
      status: 'error',
      http_status: 0,
    });

    expect(retryRequest).not.toHaveBeenCalled();
  });

  it('still retries safe read requests after an ambiguous network error', async () => {
    await import('@/services/api/client');

    const retry = responseErrorHandler?.({
      config: {
        method: 'get',
        url: '/timesheets/payroll/report',
      },
      code: 'ECONNABORTED',
    });

    await vi.runAllTimersAsync();
    await expect(retry).resolves.toEqual({ data: { status: 'success' }, status: 200 });

    expect(retryRequest).toHaveBeenCalledTimes(1);
    expect(retryRequest).toHaveBeenCalledWith(
      expect.objectContaining({
        method: 'get',
        url: '/timesheets/payroll/report',
        _retryCount: 1,
      }),
    );
  });
});
