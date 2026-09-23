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
    {
      id: 5,
      key: 'self_check_in_advance_percentage',
      value: '70',
      value_type: 'number',
    },
    {
      id: 6,
      key: 'self_check_in_advance_hold_hours',
      value: '24',
      value_type: 'number',
    },
    {
      id: 7,
      key: 'transfer_bank_account_holder',
      value: 'CONG TY LUONG TUAN',
      value_type: 'string',
    },
    {
      id: 8,
      key: 'transfer_bank_account_number',
      value: '271866699',
      value_type: 'string',
    },
    {
      id: 9,
      key: 'transfer_bank_name',
      value: 'Ngân hàng Quân đội (MB)',
      value_type: 'string',
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
  useCreateSetting: () => ({
    mutate: mocks.mutate,
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

  it('loads and saves the self-check-in advance percentage as a whole percent', async () => {
    const { result } = renderHook(() => useSettingsForm());

    await waitFor(() => {
      expect(result.current.selfCheckInAdvancePercentage).toBe('70');
    });

    act(() => {
      result.current.setSelfCheckInAdvancePercentage('85');
    });
    act(() => {
      result.current.handleSaveSelfCheckInAdvancePercentage();
    });

    expect(mocks.mutate).toHaveBeenCalledWith(
      { id: 5, data: { value: '85' } },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );

    const [, options] = mocks.mutate.mock.calls.at(-1) as [
      unknown,
      { onSuccess: () => void },
    ];
    act(() => options.onSuccess());
    expect(result.current.originalSelfCheckInAdvancePercentage).toBe('85');
  });

  it('loads and saves the self-check-in advance hold in whole hours', async () => {
    const { result } = renderHook(() => useSettingsForm());

    await waitFor(() => {
      expect(result.current.selfCheckInAdvanceHoldHours).toBe('24');
    });

    act(() => {
      result.current.setSelfCheckInAdvanceHoldHours('6');
    });
    act(() => {
      result.current.handleSaveSelfCheckInAdvanceHoldHours();
    });

    expect(mocks.mutate).toHaveBeenCalledWith(
      { id: 6, data: { value: '6' } },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );

    const [, options] = mocks.mutate.mock.calls.at(-1) as [
      unknown,
      { onSuccess: () => void },
    ];
    act(() => options.onSuccess());
    expect(result.current.originalSelfCheckInAdvanceHoldHours).toBe('6');
  });

  it('mirrors the weekly account into unset FlexPay fields', async () => {
    const { result } = renderHook(() => useSettingsForm());

    await waitFor(() => {
      expect(result.current.flexPayTransferBankNumber).toBe('271866699');
    });
    expect(result.current.flexPayTransferBankHolder).toBe('CONG TY LUONG TUAN');
    expect(result.current.flexPayTransferBankName).toBe('Ngân hàng Quân đội (MB)');
    // No FlexPay visibility row exists yet, so it mirrors the weekly toggle.
    expect(result.current.flexPayTransferBankVisible).toBe(true);
  });

  it('creates FlexPay bank rows on first save', async () => {
    const { result } = renderHook(() => useSettingsForm());

    await waitFor(() => {
      expect(result.current.flexPayTransferBankHolder).toBe('CONG TY LUONG TUAN');
    });

    act(() => {
      result.current.setFlexPayTransferBankHolder('CONG TY FLEXPAY');
      result.current.setFlexPayTransferBankNumber('444555666');
      result.current.setFlexPayTransferBankName('Ngân hàng FlexPay');
    });
    act(() => {
      result.current.handleSaveFlexPayTransferBank();
    });

    for (const [key, value] of [
      ['flexpay_transfer_bank_account_holder', 'CONG TY FLEXPAY'],
      ['flexpay_transfer_bank_account_number', '444555666'],
      ['flexpay_transfer_bank_name', 'Ngân hàng FlexPay'],
    ] as const) {
      expect(mocks.mutate).toHaveBeenCalledWith(
        { key, value, value_type: 'string' },
        expect.objectContaining({ onSuccess: expect.any(Function) }),
      );
    }

    const holderCall = mocks.mutate.mock.calls.find((call) => {
      const [arg] = call as [{ key?: string }];
      return arg.key === 'flexpay_transfer_bank_account_holder';
    });
    const [, holderOptions] = holderCall as [unknown, { onSuccess: () => void }];
    act(() => holderOptions.onSuccess());
    expect(result.current.originalFlexPayTransferBankHolder).toBe('CONG TY FLEXPAY');
  });
});
