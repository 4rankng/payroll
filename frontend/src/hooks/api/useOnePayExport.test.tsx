import type { PropsWithChildren } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { onepayExportService } from '@/services/api/onepay-export.service';
import { triggerBlobDownload } from '@/utils/file-download';
import { toast } from 'sonner';
import { useExportOnePayBulk } from './useOnePayExport';

vi.mock('@/services/api/onepay-export.service', () => ({
  onepayExportService: { exportBulk: vi.fn() },
}));
vi.mock('@/utils/file-download', () => ({ triggerBlobDownload: vi.fn() }));
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), warning: vi.fn(), error: vi.fn() },
}));

function createWrapper() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const params = { fromDate: '2026-11-01', toDate: '2026-11-07' };

describe('useExportOnePayBulk', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // An empty cohort answers with a headers-only workbook (HTTP 200, 0 rows), so
  // the hook must say the file is empty instead of "0 dòng" success — the admin
  // previously got a 422 whose wording blamed bank/SWIFT data.
  it('warns that the file is empty when no row qualified', async () => {
    vi.mocked(onepayExportService.exportBulk).mockResolvedValue({
      blob: new Blob(['x']),
      filename: 'Yeu_cau_chuyen_tien_weekly.xlsx',
      totalCount: 0,
      transferAmount: 0,
      skippedCount: 0,
      cycle: 'weekly',
    });

    const { result } = renderHook(() => useExportOnePayBulk(), { wrapper: createWrapper() });
    result.current.mutate(params);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(triggerBlobDownload).toHaveBeenCalledTimes(1);
    expect(toast.warning).toHaveBeenCalledWith(
      expect.stringContaining('file tải về đang rỗng'),
      expect.anything(),
    );
    expect(toast.success).not.toHaveBeenCalled();
  });

  it('reports the exported row count when rows exist', async () => {
    vi.mocked(onepayExportService.exportBulk).mockResolvedValue({
      blob: new Blob(['x']),
      filename: 'Yeu_cau_chuyen_tien_weekly.xlsx',
      totalCount: 36,
      transferAmount: 55_674_280,
      skippedCount: 0,
      cycle: 'weekly',
    });

    const { result } = renderHook(() => useExportOnePayBulk(), { wrapper: createWrapper() });
    result.current.mutate(params);
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(toast.success).toHaveBeenCalledWith('Đã xuất file OnePay (36 dòng)');
    expect(toast.warning).not.toHaveBeenCalled();
  });
});
