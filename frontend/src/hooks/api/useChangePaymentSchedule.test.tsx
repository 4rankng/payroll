import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { pendingScheduleChangesKey } from '@/lib/queryKeys';

// The service returns the backend's data field, which is null for this in-place
// mutation. Mock it to resolve null — the exact condition that caused the
// "Cannot read properties of null (reading 'data')" regression.
vi.mock('@/services/api/project-employee.service', () => ({
  projectEmployeeService: {
    changePaymentSchedule: vi.fn(),
  },
}));

// vi.mock factories are hoisted above imports, so create the spy via vi.hoisted.
const { showSuccessNotification } = vi.hoisted(() => ({
  showSuccessNotification: vi.fn(),
}));
vi.mock('@/utils/error-handler', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/error-handler')>();
  return { ...actual, showSuccessNotification };
});

import { useChangePaymentSchedule } from './useProjectEmployees';
import { projectEmployeeService } from '@/services/api/project-employee.service';

function createWrapper(qc: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={qc}>{children}</QueryClientProvider>;
  };
}

describe('useChangePaymentSchedule', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (projectEmployeeService.changePaymentSchedule as ReturnType<typeof vi.fn>).mockResolvedValue(
      null,
    );
  });

  it('shows a success toast, does not throw, and refreshes the affected queries when the backend resolves data:null', async () => {
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    });
    // Spy on the SAME QueryClient the hook pulls via useQueryClient, so we can
    // assert onSuccess actually invalidated the cache — the siblings of the
    // null-safety guard. A regression that drops them would otherwise leave the
    // list showing the old schedule until a manual refresh.
    const invalidateSpy = vi.spyOn(qc, 'invalidateQueries');

    const { result } = renderHook(() => useChangePaymentSchedule(), {
      wrapper: createWrapper(qc),
    });

    // Must not reject. Before the fix, onSuccess read `null.data` and threw
    // "Cannot read properties of null (reading 'data')".
    await act(async () => {
      await result.current.mutateAsync({
        assignmentId: 1023,
        data: { new_schedule: 'flexible' },
      });
    });

    expect(projectEmployeeService.changePaymentSchedule).toHaveBeenCalledWith(1023, {
      new_schedule: 'flexible',
    });
    expect(showSuccessNotification).toHaveBeenCalledWith('Cập nhật chu kỳ thanh toán thành công');

    expect(invalidateSpy).toHaveBeenCalledTimes(4);
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['project-employees'] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['projects'] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ['employees'] });
    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: pendingScheduleChangesKey() });
  });
});
