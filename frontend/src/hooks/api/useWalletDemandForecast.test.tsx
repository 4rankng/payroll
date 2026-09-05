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

  it('refetches the forecast every 30s and on window focus instead of pinning the first snapshot', () => {
    renderHook(() => useWalletDemandForecast());

    const options = vi.mocked(useQuery).mock.calls[0][0];
    expect(options.staleTime).toBe(30_000);
    expect(options.refetchOnWindowFocus).toBe(true);
    expect(options).not.toHaveProperty('refetchInterval');
  });
});
