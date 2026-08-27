import type { PropsWithChildren } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { authService } from '@/services/api/auth.service';
import { useAuth } from '@/contexts';
import { useLogout } from './useAuth';

vi.mock('@/services/api/auth.service', () => ({
  authService: {
    logout: vi.fn(),
    clearLocalSession: vi.fn(),
  },
}));

vi.mock('@/contexts', () => ({
  useAuth: vi.fn(() => ({ logout: vi.fn() })),
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
  return wrapper;
}

describe('useLogout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('calls the logout API on a normal logout', async () => {
    vi.mocked(authService.logout).mockResolvedValue({} as never);
    const { result } = renderHook(() => useLogout(), { wrapper: createWrapper() });

    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(authService.logout).toHaveBeenCalledTimes(1);
    expect(authService.clearLocalSession).not.toHaveBeenCalled();
  });

  it('skips the doomed POST /auth/logout after a password change (token already blacklisted server-side)', async () => {
    const { result } = renderHook(() => useLogout({ skipApiCall: true }), {
      wrapper: createWrapper(),
    });

    result.current.mutate();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // The token was blacklisted by /auth/change-password, so the API call can
    // only ever 401 — it must not be sent at all.
    expect(authService.logout).not.toHaveBeenCalled();
    expect(authService.clearLocalSession).toHaveBeenCalledTimes(1);
    expect(useAuth).toHaveBeenCalled();
  });
});
