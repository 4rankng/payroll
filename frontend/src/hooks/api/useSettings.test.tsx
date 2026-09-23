import type { PropsWithChildren } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { settingsService } from '@/services/api/settings.service';
import { useMultipleSettings } from './useSettings';

vi.mock('@/services/api/settings.service', () => ({
  settingsService: {
    getSettingByKey: vi.fn(),
    getSettings: vi.fn(),
  },
}));

describe('useMultipleSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads every requested setting through its exact key endpoint', async () => {
    vi.mocked(settingsService.getSettingByKey).mockImplementation(async (key) => ({
      id: key === 'first' ? 1 : 2,
      key,
      value: '400000000',
      value_type: 'number',
      updated_at: '2026-07-26T00:00:00Z',
    }));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(() => useMultipleSettings(['first', 'second']), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(settingsService.getSettingByKey).toHaveBeenNthCalledWith(1, 'first');
    expect(settingsService.getSettingByKey).toHaveBeenNthCalledWith(2, 'second');
    expect(settingsService.getSettings).not.toHaveBeenCalled();
    expect(result.current.data?.map((setting) => setting.key)).toEqual(['first', 'second']);
  });

  it('tolerates an unseeded key while the others load', async () => {
    vi.mocked(settingsService.getSettingByKey).mockImplementation(async (key) => {
      if (key === 'missing') {
        throw new Error('setting not found');
      }
      return {
        id: 1,
        key,
        value: '400000000',
        value_type: 'number',
        updated_at: '2026-07-26T00:00:00Z',
      };
    });
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(() => useMultipleSettings(['present', 'missing']), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.[0]?.key).toBe('present');
    expect(result.current.data?.[1]).toBeUndefined();
  });

  it('surfaces a total load failure instead of rendering empty settings', async () => {
    vi.mocked(settingsService.getSettingByKey).mockRejectedValue(new Error('settings API down'));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const wrapper = ({ children }: PropsWithChildren) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(() => useMultipleSettings(['first', 'second']), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(result.current.error?.message).toBe('settings API down');
  });
});
