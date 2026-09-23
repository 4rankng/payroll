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
    selfCheckInAdvancePercentage: '70',
    originalSelfCheckInAdvancePercentage: '70',
    selfCheckInAdvanceHoldHours: '24',
    originalSelfCheckInAdvanceHoldHours: '24',
    transferBankHolder: 'CONG TY TNHH MTV GPPM TING TING',
    originalTransferBankHolder: 'CONG TY TNHH MTV GPPM TING TING',
    transferBankNumber: '271866699',
    originalTransferBankNumber: '271866699',
    transferBankName: 'Ngân hàng Quân đội (MB)',
    originalTransferBankName: 'Ngân hàng Quân đội (MB)',
    transferBankVisible: true,
    originalTransferBankVisible: true,
    flexPayTransferBankHolder: 'CONG TY FLEXPAY',
    originalFlexPayTransferBankHolder: 'CONG TY FLEXPAY',
    flexPayTransferBankNumber: '444555666',
    originalFlexPayTransferBankNumber: '444555666',
    flexPayTransferBankName: 'Ngân hàng FlexPay',
    originalFlexPayTransferBankName: 'Ngân hàng FlexPay',
    flexPayTransferBankVisible: true,
    originalFlexPayTransferBankVisible: true,
    bulkTransferWorkbookLimitSaveError: null,
    bulkTransferWorkbookLimitUnavailableMessage: null,
    loadError: null as string | null,
    isSaving: false,
    isLoading: false,
    setWeeklyPaymentPercentage: vi.fn(),
    setMonthlyPaymentPercentage: vi.fn(),
    setPartnerCompany: vi.fn(),
    setBulkTransferWorkbookLimitVnd: vi.fn(),
    setSelfCheckInAdvancePercentage: vi.fn(),
    setSelfCheckInAdvanceHoldHours: vi.fn(),
    setTransferBankHolder: vi.fn(),
    setTransferBankNumber: vi.fn(),
    setTransferBankName: vi.fn(),
    setTransferBankVisible: vi.fn(),
    setFlexPayTransferBankHolder: vi.fn(),
    setFlexPayTransferBankNumber: vi.fn(),
    setFlexPayTransferBankName: vi.fn(),
    setFlexPayTransferBankVisible: vi.fn(),
    handleSaveWeeklyPayment: vi.fn(),
    handleSaveMonthlyPayment: vi.fn(),
    handleSavePartnerCompany: vi.fn(),
    handleSaveBulkTransferWorkbookLimitVnd: vi.fn(),
    handleSaveSelfCheckInAdvancePercentage: vi.fn(),
    handleSaveSelfCheckInAdvanceHoldHours: vi.fn(),
    handleSaveTransferBank: vi.fn(),
    handleSaveTransferBankVisible: vi.fn(),
    handleSaveFlexPayTransferBank: vi.fn(),
    handleSaveFlexPayTransferBankVisible: vi.fn(),
    retryLoading: vi.fn(),
  },
}));

vi.mock('@/hooks/settings/useSettingsForm', () => ({
  useSettingsForm: () => mocks.formState,
}));

vi.mock('@/components/admin/AdvancePaymentFeeSchedule/FeeScheduleSection', () => ({
  FeeScheduleSection: () => null,
}));
vi.mock('@/components/admin/WeeklyPaymentFeeSchedule/WeeklyPaymentFeeScheduleSection', () => ({
  WeeklyPaymentFeeScheduleSection: () => null,
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
vi.mock('@/components/settings/ZaloConnectionSection', () => ({
  ZaloConnectionSection: () => <div>Quy trình cấu hình Zalo ZNS</div>,
}));
vi.mock('@/components/settings/AdBannerSection', () => ({
  AdBannerSection: () => <div>Chiến dịch quảng cáo</div>,
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

    const limitSlider = screen.getByRole('slider', { name: 'Giới hạn tổng tiền mỗi file Chuyển lô' });
    expect(limitSlider).toHaveAttribute('aria-valuenow', '400000000');
    expect(limitSlider).toHaveAttribute('aria-valuemin', '100000000');
    expect(limitSlider).toHaveAttribute('aria-valuemax', '500000000');
    // The formatted value appears in the readout and again on the matching
    // quick-stop chip; both must render the full VND figure.
    expect(screen.getAllByText('400.000.000').length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: '400.000.000' })).toHaveAttribute(
      'aria-pressed',
      'true',
    );
    expect(screen.getByText(
      'Hệ thống tự tách file để tổng tiền mỗi file luôn nhỏ hơn giới hạn này.',
    )).toBeInTheDocument();
    expect(limitSlider.closest('.grid')?.querySelector('input')).toBeNull();
  });

  it('allows admins to configure the self-check-in advance percentage', () => {
    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    const percentageInput = screen.getByLabelText('Tỷ lệ ứng lương tự chấm công');
    expect(percentageInput).toHaveValue(70);
    expect(percentageInput).toHaveAttribute('min', '1');
    expect(percentageInput).toHaveAttribute('max', '100');
  });

  it('allows admins to configure the post-checkout advance wait in whole hours', () => {
    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    const holdInput = screen.getByLabelText('Thời gian chờ ứng lương tự chấm công sau khi tan ca');
    expect(holdInput).toHaveValue(24);
    expect(holdInput).toHaveAttribute('min', '0');
    expect(holdInput).toHaveAttribute('max', '720');
    expect(holdInput).toHaveAttribute('step', '1');
  });

  it('keeps both transfer account settings available in both responsive views', () => {
    render(
      <MemoryRouter>
        <PageComponent />
      </MemoryRouter>,
    );

    expect(screen.getByLabelText('Chủ tài khoản lương tuần')).toHaveValue(
      'CONG TY TNHH MTV GPPM TING TING',
    );
    expect(screen.getByLabelText('Số tài khoản lương tuần')).toHaveValue('271866699');
    expect(screen.getByLabelText('Ngân hàng lương tuần')).toHaveValue('Ngân hàng Quân đội (MB)');

    expect(screen.getByLabelText('Chủ tài khoản FlexPay')).toHaveValue('CONG TY FLEXPAY');
    expect(screen.getByLabelText('Số tài khoản FlexPay')).toHaveValue('444555666');
    expect(screen.getByLabelText('Ngân hàng FlexPay')).toHaveValue('Ngân hàng FlexPay');
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

  it('keeps the Zalo ZNS workflow available', () => {
    render(
      <MemoryRouter initialEntries={['/admin/settings?tab=zalo']}>
        <PageComponent />
      </MemoryRouter>,
    );

    expect(screen.getByRole('tab', { name: /Zalo ZNS/ })).toHaveAttribute('data-state', 'active');
    expect(screen.getByText('Quy trình cấu hình Zalo ZNS')).toBeInTheDocument();
  });

  it('keeps the ad campaign management available', () => {
    render(
      <MemoryRouter initialEntries={['/admin/settings?tab=ads']}>
        <PageComponent />
      </MemoryRouter>,
    );

    expect(screen.getByRole('tab', { name: /Quảng cáo/ })).toHaveAttribute('data-state', 'active');
    expect(screen.getByText('Chiến dịch quảng cáo')).toBeInTheDocument();
  });
});
