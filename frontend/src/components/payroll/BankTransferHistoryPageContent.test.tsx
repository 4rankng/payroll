import { fireEvent, render, screen, within } from '@testing-library/react';
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
        summary: { total_amount: 1_998_000, transfer_count: 2, employee_count: 1 },
        pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 1 },
      },
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
  });

  it('shows every bank posting and the employee-cycle total', () => {
    render(<BankTransferHistoryPageContent />);

    expect(screen.getAllByText(/1\.998\.000\s*₫/)).toHaveLength(2);
    expect(screen.getByText('2 bút toán ngân hàng')).toBeInTheDocument();
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

    expect(container.querySelector('.admin-payment-history-workspace')).not.toBeInTheDocument();

    rerender(<BankTransferHistoryPageContent variant="admin" />);

    expect(container.querySelector('.admin-payment-history-workspace')).toHaveClass('ct-card');
  });

  it('keeps touch-safe mobile filters while using compact desktop controls', () => {
    render(<BankTransferHistoryPageContent />);

    expect(screen.getByRole('button', { name: /Chọn tháng kỳ lương/ })).toHaveClass('h-11', 'sm:h-9');
    expect(screen.getByRole('combobox')).toHaveClass('min-h-11', 'sm:min-h-9');
    // The shared SearchBar keeps the icon and input as flex siblings (sizing on
    // the wrapper), so the placeholder can never slide under the search icon.
    expect(screen.getByRole('textbox').parentElement).toHaveClass('min-h-11', 'sm:min-h-9');
    expect(screen.getByRole('textbox')).toHaveAttribute('placeholder', 'Tên nhân viên, mã chuyển khoản hoặc mã ngân hàng');
  });

  it('groups completed payments into one compact comparison workspace on wide screens', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const workspace = container.querySelector('[data-slot="payment-history-workspace"]');
    const records = container.querySelector('[data-slot="payment-history-records"]');
    const disclosure = screen.getByLabelText(/Chi tiết giao dịch/i);

    expect(workspace).toHaveClass('overflow-hidden', 'bg-white', 'rounded-xl', 'xl:overflow-visible', 'xl:rounded-none');
    // Records are flat hairline rows of the workspace module at every
    // breakpoint — no nested per-record cards.
    expect(records).toHaveClass('divide-y', 'divide-slate-200/80');
    expect(records.className).not.toContain('space-y-');
    expect(records.className).not.toContain('gap-');
    expect(records.className).not.toContain('p-2');
    const recordRow = disclosure.closest('details')!;
    expect(recordRow.className).not.toContain('rounded-');
    expect(recordRow.className).not.toContain('bg-white');
    expect(recordRow.className).not.toContain('border-');
    expect(recordRow.className).not.toContain('shadow-[0_');
    expect(disclosure).toHaveClass('xl:min-h-[48px]', 'xl:px-4', 'xl:py-1.5');
    // Sticky column header keeps the dense table scannable mid-scroll.
    expect(container.querySelector('[data-slot="payment-history-header"]')).toHaveClass(
      'xl:sticky',
      'xl:top-0',
      'xl:z-10',
      'xl:bg-slate-50/95',
      'xl:backdrop-blur',
    );
    // The identity badge shrinks from a card-style avatar to a table-style dot.
    expect(disclosure.firstElementChild!.firstElementChild).toHaveClass('xl:h-5', 'xl:w-5');
  });

  it('answers the month total at a glance and paginates without leaving the module', () => {
    const result = useBankTransferHistories();
    result.data.pagination = { page: 1, pageSize: 20, totalPages: 11, totalRecords: 207 };
    useBankTransferHistories.mockReturnValue(result);

    const { container } = render(<BankTransferHistoryPageContent />);

    const statStrip = within(container.querySelector('.shadow-soft')!);
    expect(statStrip.getByText('Tổng đã chuyển')).toBeInTheDocument();
    expect(statStrip.getByText('Bút toán ngân hàng')).toBeInTheDocument();
    expect(statStrip.getByText('Nhân viên')).toBeInTheDocument();
    expect(statStrip.getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();
    expect(within(container.querySelector('[data-slot="payment-history-records"]')!).getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();

    // The toolbar lives inside the workspace module, not on a floating card.
    const toolbar = container.querySelector('[data-slot="filter-bar"]');
    expect(toolbar).toBeInTheDocument();
    expect(toolbar!.closest('[data-slot="payment-history-workspace"]')).toBeInTheDocument();

    // Identical per-row status is noise; the module carries the completed state.
    expect(screen.queryByText('Đã chuyển')).not.toBeInTheDocument();

    expect(screen.getByRole('button', { name: 'Trang 2' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Trang cuối' })).toBeInTheDocument();
    expect(screen.getByText('1–20')).toBeInTheDocument();
    // Desktop counter and the mobile pager both report the totals.
    expect(screen.getAllByText(/207/)).toHaveLength(2);
  });

  it('marks the expanded record with a single restrained accent, not a green wash', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const record = container.querySelector('details');
    const disclosure = screen.getByLabelText(/Chi tiết giao dịch/i);
    const detailsBody = document.getElementById(
      disclosure.getAttribute('aria-controls')!,
    );
    const chevron = Array.from(disclosure.querySelectorAll('svg')).find((icon) =>
      icon.classList.contains('group-open/record:rotate-180'),
    );
    const panel = detailsBody!.firstElementChild;

    // Ownership is signaled by one left accent bar on a flat row, not a
    // full-row/panel green wash or a per-record card.
    expect(record).toHaveClass('open:shadow-[inset_4px_0_0_0_#059669]');
    expect(record.className).not.toContain('open:bg-emerald-50/80');
    expect(record.className).not.toContain('open:border-emerald-300');
    expect(record.className).not.toContain('open:border-slate-300');
    expect(record.className).not.toContain('rounded-');
    expect(disclosure).toHaveClass('group-open/record:hover:bg-transparent');
    // The wrapper drops its border-t when open so the panel's own hairline is
    // the only seam — summary and expanded rows keep exactly one divider.
    expect(detailsBody).toHaveClass('group-open/record:border-t-0');
    // The expanded panel is a flat neutral surface, not a nested card or a green slab.
    expect(panel).toHaveClass('bg-slate-50/80', 'border-t', 'border-slate-200/80');
    expect(panel!.firstElementChild).toHaveClass('xl:grid', 'border-slate-200/80');
    expect(panel!.querySelector('[role="list"]')).toHaveClass('divide-y', 'divide-slate-200/70');
    // Emerald is reserved for the money value and the open-state chevron.
    expect(chevron).toHaveClass('group-open/record:text-emerald-600');
  });

  it('keeps employee identity readable on narrow screens', () => {
    render(<BankTransferHistoryPageContent />);

    const name = screen.getByText('LÒ THỊ MINH THU');
    expect(name).toHaveClass('break-words', 'xl:truncate');
    // Desktop runs name, CCCD and projects on a single baseline row.
    expect(name.parentElement).toHaveClass('xl:flex', 'xl:items-baseline', 'xl:gap-x-1.5');
    expect(screen.getByText('CCCD 031189014251')).toHaveClass('xl:mt-0');
  });

  it('expands only one record at a time and collapses it on page change', () => {
    const result = useBankTransferHistories();
    result.data.data.push({
      ...result.data.data[0],
      employee_id: 83,
      employee_name: 'TRẦN VĂN B',
      employee_cccd: '031199001234',
    });
    result.data.pagination = { page: 1, pageSize: 50, totalPages: 2, totalRecords: 2 };
    useBankTransferHistories.mockReturnValue(result);

    render(<BankTransferHistoryPageContent />);

    const [first, second] = screen.getAllByLabelText(/Chi tiết giao dịch/i);
    fireEvent.click(first);
    expect(first.closest('details')).toHaveAttribute('open');
    expect(second.closest('details')).not.toHaveAttribute('open');

    fireEvent.click(second);
    expect(second.closest('details')).toHaveAttribute('open');
    expect(first.closest('details')).not.toHaveAttribute('open');

    // Leaving the page collapses the open record instead of stranding it.
    fireEvent.click(screen.getByRole('button', { name: 'Trang 2' }));
    expect(second.closest('details')).not.toHaveAttribute('open');
  });

  it('requests a denser default page now that rows are compact', () => {
    render(<BankTransferHistoryPageContent />);

    expect(useBankTransferHistories).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 1, pageSize: 50 }),
    );
  });

  it('renders loading placeholders as flat rows of the workspace module', () => {
    useBankTransferHistories.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      refetch: vi.fn(),
    });

    const { container } = render(<BankTransferHistoryPageContent />);

    const skeleton = container.querySelector('[aria-label="Đang tải lịch sử trả lương"]');
    expect(skeleton).toHaveClass('divide-y', 'divide-slate-200/80');
    // rounded-none is load-bearing: it overrides the Skeleton base rounded-md
    // so placeholder rows match the flat record rows they stand in for.
    const rows = Array.from(skeleton!.children);
    expect(rows).toHaveLength(6);
    for (const row of rows) {
      expect(row).toHaveClass('h-36', 'rounded-none', 'xl:h-16');
    }
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
