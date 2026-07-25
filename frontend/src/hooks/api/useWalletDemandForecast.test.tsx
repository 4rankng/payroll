import { renderHook } from '@testing-library/react';
import { useQuery } from '@tanstack/react-query';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useWalletDemandForecast } from './useWalletDemandForecast';

vi.mock('@tanstack/react-query', () => ({
  useQuery: vi.fn(),
}));

vi.mock('@/services/api/wallet.service', () => ({
  walletService: {
    getDemandForecast: vi.fn(),
  },
}));

describe('useWalletDemandForecast', () => {
  beforeEach(() => {
    vi.mocked(useQuery).mockReset();
  });

  it('keeps the page-session forecast until the browser page is refreshed', () => {
    renderHook(() => useWalletDemandForecast());

    const options = vi.mocked(useQuery).mock.calls[0][0];
    expect(options.staleTime).toBe(Number.POSITIVE_INFINITY);
    expect(options.refetchOnWindowFocus).toBe(false);
    expect(options).not.toHaveProperty('refetchInterval');
  });
});
