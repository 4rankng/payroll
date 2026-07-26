import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import SettingsPage from './index';
import SettingsPageMobile from '../../mobile/admin/SettingsPage';

const mocks = vi.hoisted(() => ({
  retryLoading: vi.fn(),
  formState: {
    weeklyPaymentPercentage: '80',
    originalWeeklyPayment: '80',
    monthlyPaymentPercentage: '90',
    originalMonthlyPayment: '90',
    partnerCompany: 'TingTing',
    originalPartnerCompany: 'TingTing',
    bulkTransferWorkbookLimitVnd: '400000000',
    originalBulkTransferWorkbookLimitVnd: '400000000',
    bulkTransferWorkbookLimitSaveError: null,
    bulkTransferWorkbookLimitUnavailableMessage: null,
    loadError: null as string | null,
    isSaving: false,
    isLoading: false,
    setWeeklyPaymentPercentage: vi.fn(),
    setMonthlyPaymentPercentage: vi.fn(),
    setPartnerCompany: vi.fn(),
    setBulkTransferWorkbookLimitVnd: vi.fn(),
    handleSaveWeeklyPayment: vi.fn(),
    handleSaveMonthlyPayment: vi.fn(),
    handleSavePartnerCompany: vi.fn(),
    handleSaveBulkTransferWorkbookLimitVnd: vi.fn(),
    retryLoading: vi.fn(),
  },
}));

vi.mock('@/hooks/settings/useSettingsForm', () => ({
  useSettingsForm: () => mocks.formState,
}));

vi.mock('@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection', () => ({
  FeeScheduleSection: () => null,
}));
vi.mock('@/components/admin/DisbursementFeeSchedule/DisbursementFeeScheduleSection', () => ({
  DisbursementFeeScheduleSection: () => null,
}));
vi.mock('@/components/email/AdminEmailComposer', () => ({
  AdminEmailComposer: () => null,
}));
vi.mock('@/components/settings/SendNotificationComposer', () => ({
  SendNotificationComposer: () => null,
}));

describe.each([
  ['desktop', SettingsPage],
  ['mobile', SettingsPageMobile],
] as const)('SettingsPage %s parity', (_view, PageComponent) => {
  beforeEach(() => {
    mocks.formState.loadError = null;
    mocks.formState.retryLoading = mocks.retryLoading;
    mocks.retryLoading.mockClear();
  });

  it('renders the exact Chuyển lô setting first with the full VND value', () => {
    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    const limitInput = screen.getByLabelText('Giới hạn tổng tiền mỗi file Chuyển lô');
    expect(limitInput).toHaveValue('400.000.000');
    expect(screen.getByText(
      'Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này.',
    )).toBeInTheDocument();
    expect(limitInput.closest('.grid')?.querySelector('input')).toBe(limitInput);
  });

  it('shows a page-level retry state when any required setting cannot load', () => {
    mocks.formState.loadError = 'Không thể tải cài đặt. Vui lòng thử lại.';

    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('Không thể tải cài đặt');
    expect(screen.queryByLabelText('Giới hạn tổng tiền mỗi file Chuyển lô')).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Thử lại' }));
    expect(mocks.retryLoading).toHaveBeenCalledOnce();
  });
});
