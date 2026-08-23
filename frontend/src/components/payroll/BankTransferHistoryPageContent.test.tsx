import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { BankTransferHistoryPageContent } from './BankTransferHistoryPageContent';

const useBankTransferHistories = vi.fn();

vi.mock('@/hooks/api/usePayrolls', () => ({
  useBankTransferHistories: (...args: unknown[]) => useBankTransferHistories(...args),
}));

describe('BankTransferHistoryPageContent', () => {
  beforeEach(() => {
    // getCurrentMonthValue() reads the real clock; pin it so the month picker
    // always renders 07/2026 regardless of when the suite runs.
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-07-15T12:00:00+07:00'));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  beforeEach(() => {
    useBankTransferHistories.mockReturnValue({
      data: {
        status: 'success',
        message: '',
        data: [{
          employee_id: 82,
          employee_name: 'LÒ THỊ MINH THU',
          employee_cccd: '031189014251',
          project_ids: [1],
          project_names: ['Dự án Việt Nam'],
          work_month: '2026-07',
          cycle: 2,
          from_date: '2026-07-08',
          to_date: '2026-07-14',
          payment_date: '2026-07-17',
          total_amount: 1_998_000,
          transfers: [
            { transfer_code: 'VFIC6d037214', bank_reference: 'FT26198846619959', amount: 1_548_000, paid_at: '2026-07-17T20:24:00+07:00' },
            { transfer_code: 'VFIC7a193042', bank_reference: 'FT26198940380850', amount: 450_000, paid_at: '2026-07-17T20:31:00+07:00' },
          ],
        }],
        pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 1 },
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
  });

  it('shows every bank posting and the employee-cycle total', () => {
    render(<BankTransferHistoryPageContent />);

    expect(screen.getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText('2 bút toán')).toBeInTheDocument();
    expect(screen.getByText('FT26198846619959')).not.toBeVisible();

    const disclosure = screen.getByLabelText(/Chi tiết giao dịch/i);
    expect(disclosure.closest('details')).not.toHaveAttribute('open');

    fireEvent.click(disclosure);

    expect(screen.getByText('FT26198846619959')).toBeInTheDocument();
    expect(screen.getByText('FT26198940380850')).toBeInTheDocument();
    expect(screen.getByText('VFIC6d037214')).toBeInTheDocument();
    expect(screen.getByText('VFIC7a193042')).toBeInTheDocument();
    expect(screen.getAllByText('Ghi chú chuyển khoản')).toHaveLength(3);
    expect(screen.getAllByText('Mã giao dịch ngân hàng')).toHaveLength(3);
    expect(screen.getAllByText('Thời gian xử lý')).toHaveLength(3);
    expect(screen.getByText('17/07/2026 · 20:24')).toHaveAttribute('datetime', '2026-07-17T20:24:00+07:00');
    expect(screen.getByText('17/07/2026 · 20:31')).toHaveAttribute('datetime', '2026-07-17T20:31:00+07:00');
    expect(screen.getByText('CCCD 031189014251')).toBeInTheDocument();
    expect(screen.getByText(/1\.548\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText(/450\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 07/2026' })).toHaveTextContent('07/2026');
    expect(screen.queryByText('July 2026')).not.toBeInTheDocument();
    expect(screen.getByText('Kỳ 2')).toBeInTheDocument();
    expect(screen.getByText('08–14/07/2026')).toHaveAttribute('aria-label', 'Từ 08/07/2026 đến 14/07/2026');
    expect(screen.getByText('17/07/2026')).toBeInTheDocument();
    const expandedCard = screen.getByLabelText(/Chi tiết giao dịch/i);
    expect(expandedCard.closest('details')).toHaveAttribute('open');

    fireEvent.click(expandedCard);

    expect(screen.getByText('FT26198846619959')).not.toBeVisible();
    expect(expandedCard.closest('details')).not.toHaveAttribute('open');
  });

  it('shows an empty-note fallback when a legacy payment has no transfer code', () => {
    const result = useBankTransferHistories();
    result.data.data[0].employee_cccd = '';
    result.data.data[0].transfers = [
      { transfer_code: '', bank_reference: 'FT-LEGACY-001', amount: 1_998_000 },
    ];
    useBankTransferHistories.mockReturnValue(result);

    render(<BankTransferHistoryPageContent />);
    fireEvent.click(screen.getByLabelText(/Chi tiết giao dịch/i));

    expect(screen.getByText('FT-LEGACY-001')).toBeInTheDocument();
    expect(screen.getAllByText('Ghi chú chuyển khoản')).toHaveLength(2);
    expect(screen.getAllByText('Mã giao dịch ngân hàng')).toHaveLength(2);
    expect(screen.getAllByText('—')).toHaveLength(2);
    expect(screen.getByText('CCCD —')).toBeInTheDocument();
  });

  it('selects a month directly without asking for a day', () => {
    render(<BankTransferHistoryPageContent />);

    fireEvent.click(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 07/2026' }));
    fireEvent.click(screen.getByRole('button', { name: 'Chọn tháng 08 năm 2026' }));

    expect(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 08/2026' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Chọn tháng 08 năm 2026' })).not.toBeInTheDocument();
  });

  it('scopes daisyUI card behavior to the admin variant', () => {
    const { container, rerender } = render(<BankTransferHistoryPageContent />);

    expect(container.querySelector('.admin-payment-history-filters')).not.toBeInTheDocument();
    expect(container.querySelector('.admin-payment-history-workspace')).not.toBeInTheDocument();

    rerender(<BankTransferHistoryPageContent variant="admin" />);

    expect(container.querySelector('.admin-payment-history-filters')).toHaveClass('ct-card');
    expect(container.querySelector('.admin-payment-history-workspace')).toHaveClass('ct-card');
  });

  it('keeps touch-safe mobile filters while using compact desktop controls', () => {
    render(<BankTransferHistoryPageContent />);

    expect(screen.getByRole('button', { name: /Chọn tháng kỳ lương/ })).toHaveClass('h-11', 'sm:h-8');
    expect(screen.getByRole('combobox')).toHaveClass('h-11', 'sm:h-8');
    expect(screen.getByRole('textbox')).toHaveClass('h-11', 'sm:h-8');
    // Icon sits at left-3 + w-4 (ends 28px in); keep 36px padding at every
    // breakpoint so the placeholder never slides under the search icon.
    expect(screen.getByRole('textbox')).toHaveClass('pl-9', 'sm:pl-9');
  });

  it('groups completed payments into one compact comparison workspace on wide screens', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const workspace = container.querySelector('[data-slot="payment-history-workspace"]');
    const records = container.querySelector('[data-slot="payment-history-records"]');
    const disclosure = screen.getByLabelText(/Chi tiết giao dịch/i);

    expect(workspace).toHaveClass('xl:overflow-hidden', 'xl:border', 'xl:bg-white');
    expect(records).toHaveClass('xl:divide-y');
    expect(disclosure).toHaveClass('xl:min-h-[64px]', 'xl:px-4', 'xl:py-2');
    expect(disclosure.closest('details')).toHaveClass('xl:rounded-none', 'xl:border-0', 'xl:shadow-none');
  });

  it('marks the expanded record with a distinct tint so it stands out from neighbours', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const record = container.querySelector('details');
    const disclosure = screen.getByLabelText(/Chi tiết giao dịch/i);
    const detailsBody = document.getElementById(
      disclosure.getAttribute('aria-controls')!,
    );

    expect(record).toHaveClass('open:bg-emerald-50/60', 'open:border-emerald-300');
    expect(disclosure).toHaveClass('group-open/record:hover:bg-transparent');
    expect(detailsBody).toHaveClass('group-open/record:border-emerald-200');
  });

  it('keeps employee identity readable on narrow screens', () => {
    render(<BankTransferHistoryPageContent />);

    expect(screen.getByText('LÒ THỊ MINH THU')).toHaveClass('break-words', 'xl:truncate');
  });

  it.each(['admin', 'partner'] as const)('keeps the %s header below the iOS safe area at every breakpoint', (variant) => {
    const { container } = render(<BankTransferHistoryPageContent variant={variant} />);

    expect(container.querySelector('.admin-payment-history-page')).toHaveClass(
      'pt-[calc(env(safe-area-inset-top,0px)+0.75rem)]',
      'sm:pt-[calc(env(safe-area-inset-top,0px)+1rem)]',
      'md:pt-[calc(env(safe-area-inset-top,0px)+1.5rem)]',
    );
  });
});
