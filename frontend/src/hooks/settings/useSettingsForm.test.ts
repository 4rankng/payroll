import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useSettingsForm } from './useSettingsForm';

const mocks = vi.hoisted(() => ({
  mutate: vi.fn(),
  mutateAsync: vi.fn(),
  refetch: vi.fn(),
  settings: [
    {
      id: 1,
      key: 'bulk_transfer_payment_percentage',
      value: '0.8',
      value_type: 'number',
    },
    {
      id: 2,
      key: 'monthly_payment_percentage',
      value: '0.9',
      value_type: 'number',
    },
    {
      id: 3,
      key: 'partner_company',
      value: 'TingTing',
      value_type: 'string',
    },
    {
      id: 4,
      key: 'bulk_transfer_workbook_limit_vnd',
      value: '400000000',
      value_type: 'number',
    },
  ],
}));

vi.mock('@/hooks/api/useSettings', () => ({
  useMultipleSettings: () => ({
    data: mocks.settings,
    error: null,
    isError: false,
    isLoading: false,
    refetch: mocks.refetch,
  }),
  useUpdateSetting: () => ({
    mutate: mocks.mutate,
    mutateAsync: mocks.mutateAsync,
    isPending: false,
  }),
}));

describe('useSettingsForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.mutateAsync.mockResolvedValue({ message: 'Đã lưu' });
  });

  it('loads the VND limit and saves a canonical integer string', async () => {
    const { result } = renderHook(() => useSettingsForm());

    await waitFor(() => {
      expect(result.current.bulkTransferWorkbookLimitVnd).toBe('400000000');
    });
    expect(result.current.originalBulkTransferWorkbookLimitVnd).toBe('400000000');

    act(() => {
      result.current.setBulkTransferWorkbookLimitVnd('500000000');
    });
    await act(async () => {
      await result.current.handleSaveBulkTransferWorkbookLimitVnd();
    });

    expect(mocks.mutateAsync).toHaveBeenCalledWith({
      id: 4,
      data: { value: '500000000' },
    });
    expect(result.current.originalBulkTransferWorkbookLimitVnd).toBe('500000000');
  });

  it('keeps the original value unchanged and exposes a recoverable save error', async () => {
    mocks.mutateAsync.mockRejectedValueOnce(new Error('network'));
    const { result } = renderHook(() => useSettingsForm());

    await waitFor(() => {
      expect(result.current.originalBulkTransferWorkbookLimitVnd).toBe('400000000');
    });
    act(() => {
      result.current.setBulkTransferWorkbookLimitVnd('500000000');
    });
    await act(async () => {
      await result.current.handleSaveBulkTransferWorkbookLimitVnd();
    });

    expect(result.current.originalBulkTransferWorkbookLimitVnd).toBe('400000000');
    expect(result.current.bulkTransferWorkbookLimitSaveError).toContain('Không thể lưu');

    act(() => {
      result.current.setBulkTransferWorkbookLimitVnd('600000000');
    });
    expect(result.current.bulkTransferWorkbookLimitSaveError).toBeNull();
  });
});
