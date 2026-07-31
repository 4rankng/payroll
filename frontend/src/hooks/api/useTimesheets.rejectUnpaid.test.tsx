import { renderHook } from '@testing-library/react';
import { useMutation } from '@tanstack/react-query';
import { beforeEach, vi } from 'vitest';
import { invalidateCache } from '@/lib/cache/invalidationService';
import { timesheetService } from '@/services/api/timesheet.service';
import { showSuccessNotification } from '@/utils/error-handler';
import { toast } from '@/components/ui/sonner';
import { useRejectUnpaidTimesheets } from './useTimesheets';

vi.mock('@tanstack/react-query', () => ({
  useMutation: vi.fn(() => ({ mutateAsync: vi.fn(), isPending: false })),
  useQuery: vi.fn(),
  useQueryClient: vi.fn(),
}));

vi.mock('@/services/api/timesheet.service', () => ({
  timesheetService: { rejectUnpaid: vi.fn() },
}));

vi.mock('@/lib/cache/invalidationService', () => ({
  invalidateCache: vi.fn().mockResolvedValue(undefined),
}));

vi.mock('@/utils/error-handler', () => ({
  showBulkOperationNotification: vi.fn(),
  showErrorNotification: vi.fn(),
  showSuccessNotification: vi.fn(),
}));

vi.mock('@/components/ui/sonner', () => ({
  toast: vi.fn(),
}));

describe('useRejectUnpaidTimesheets', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('submits the server-scoped payload and invalidates timesheet/dashboard summaries', async () => {
    renderHook(() => useRejectUnpaidTimesheets());
    const options = vi.mocked(useMutation).mock.calls[0][0] as {
      mutationFn: (payload: unknown) => Promise<unknown>;
      onSuccess: (result: {
        rejected_count: number;
        post_commit_complete: boolean;
        warnings?: string[];
      }) => void;
    };
    const payload = {
      project_id: 7,
      from_date: '2026-07-01',
      to_date: '2026-07-31',
      rejection_reason: 'Sai dữ liệu',
    };
    vi.mocked(timesheetService.rejectUnpaid).mockResolvedValue({
      rejected_count: 4,
      post_commit_complete: true,
      warnings: [],
    });

    await options.mutationFn(payload);
    options.onSuccess({ rejected_count: 4, post_commit_complete: true, warnings: [] });

    expect(timesheetService.rejectUnpaid).toHaveBeenCalledWith(payload);
    expect(invalidateCache).toHaveBeenCalledWith('timesheet:bulkReject');
    expect(showSuccessNotification).toHaveBeenCalledWith(
      'Đã loại 4 bảng công chưa thanh toán',
    );
  });

  it('reports a committed rejection as successful while surfacing post-commit warnings', () => {
    renderHook(() => useRejectUnpaidTimesheets());
    const options = vi.mocked(useMutation).mock.calls[0][0] as {
      onSuccess: (result: {
        rejected_count: number;
        post_commit_complete: boolean;
        warnings?: string[];
      }) => void;
    };

    options.onSuccess({
      rejected_count: 6,
      post_commit_complete: false,
      warnings: ['Chưa làm mới bộ nhớ đệm.', 'Nhật ký sẽ được đồng bộ lại.'],
    });

    expect(toast).toHaveBeenCalledWith({
      title: 'Đã loại 6 bảng công chưa thanh toán',
      description: 'Dữ liệu đã được lưu. Cảnh báo sau khi lưu: Chưa làm mới bộ nhớ đệm. Nhật ký sẽ được đồng bộ lại.',
      variant: 'warning',
      duration: 8000,
    });
    expect(showSuccessNotification).not.toHaveBeenCalled();
    expect(invalidateCache).toHaveBeenCalledWith('timesheet:bulkReject');
  });
});
