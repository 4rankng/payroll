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

  it('shows every bank posting and the record total with no expansion step', () => {
    render(<BankTransferHistoryPageContent />);

    // Flat ledger: both postings are plain table content — nothing behind a
    // disclosure, no hidden panels.
    expect(screen.queryByLabelText(/Chi tiết giao dịch/i)).not.toBeInTheDocument();

    for (const reference of ['FT26198846619959', 'FT26198940380850']) {
      expect(screen.getAllByText(reference).length).toBeGreaterThan(0);
    }
    for (const code of ['VFIC6d037214', 'VFIC7a193042']) {
      expect(screen.getAllByText(code).length).toBeGreaterThan(0);
    }
    // Posting amounts + the record total are all rendered.
    expect(screen.getAllByText(/1\.548\.000\s*₫/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/450\.000\s*₫/).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/1\.998\.000\s*₫/).length).toBeGreaterThan(0);

    // Record identity renders once per record on the desktop ledger.
    expect(screen.getAllByText('LÒ THỊ MINH THU')).toHaveLength(2); // desktop + mobile block
    expect(screen.getAllByText(/17\/07\/2026 · 20:24/).length).toBeGreaterThan(0);
    expect(screen.getAllByText('Kỳ 2').length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 07/2026' })).toHaveTextContent('07/2026');
  });

  it('spans record identity across its postings and numbers every ledger line', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const ledger = container.querySelector('[data-slot="payment-history-records"]')!;
    const bodyRows = Array.from(ledger.querySelectorAll('tbody tr'));

    // Two postings → two ledger lines.
    expect(bodyRows).toHaveLength(2);

    const [first, second] = bodyRows;
    // First line anchors the identity cells for the whole record…
    const identityCells = Array.from(first.querySelectorAll('td[rowspan]'));
    expect(identityCells).toHaveLength(4); // nhân viên, kỳ, ngày TT, thực nhận
    for (const cell of identityCells) {
      expect(cell).toHaveAttribute('rowspan', '2');
    }
    expect(within(first as HTMLElement).getByText('LÒ THỊ MINH THU')).toBeInTheDocument();
    expect(within(first as HTMLElement).getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();

    // …continuation lines carry only posting cells, never repeated identity.
    expect(second.querySelectorAll('td[rowspan]')).toHaveLength(0);
    expect(within(second as HTMLElement).queryByText('LÒ THỊ MINH THU')).not.toBeInTheDocument();

    // STT numbers each ledger line.
    expect(within(first as HTMLElement).getByText('01')).toBeInTheDocument();
    expect(within(second as HTMLElement).getByText('02')).toBeInTheDocument();
  });

  it('renders the API summary aggregate even when it differs from the page rows', () => {
    const result = useBankTransferHistories();
    // Summary covers the full filtered set; page rows are only a slice.
    result.data.summary = { total_amount: 2_367_588_332, transfer_count: 827, employee_count: 816 };
    useBankTransferHistories.mockReturnValue(result);

    const { container } = render(<BankTransferHistoryPageContent />);

    const summary = within(container.querySelector('[data-slot="payment-history-summary"]')!);
    expect(summary.getByText(/2\.367\.588\.332\s*₫/)).toBeInTheDocument();
    expect(summary.getByText('827')).toBeInTheDocument();
    expect(summary.getByText('816')).toBeInTheDocument();
  });

  it('keeps a legacy record with no postings on the ledger with em-dash cells', () => {
    const result = useBankTransferHistories();
    result.data.data[0].transfers = [];
    useBankTransferHistories.mockReturnValue(result);

    const { container } = render(<BankTransferHistoryPageContent />);

    const ledger = container.querySelector('[data-slot="payment-history-records"]')!;
    const rows = ledger.querySelectorAll('tbody tr');
    expect(rows).toHaveLength(1);
    expect(within(rows[0] as HTMLElement).getByText('LÒ THỊ MINH THU')).toBeInTheDocument();
    // No posting to show — every posting cell falls back to —.
    expect(within(rows[0] as HTMLElement).getAllByText('—').length).toBeGreaterThan(2);
  });

  it('shows an em-dash fallback when a legacy posting lacks code, bank ref time, or CCCD', () => {
    const result = useBankTransferHistories();
    result.data.data[0].employee_cccd = '';
    result.data.data[0].transfers = [
      { transfer_code: '', bank_reference: 'FT-LEGACY-001', amount: 1_998_000 },
    ];
    useBankTransferHistories.mockReturnValue(result);

    render(<BankTransferHistoryPageContent />);

    expect(screen.getAllByText('FT-LEGACY-001').length).toBeGreaterThan(0);
    // Missing transfer code and paid_at fall back to —, CCCD shows — inline.
    expect(screen.getAllByText('—').length).toBeGreaterThan(0);
    expect(screen.getAllByText('CCCD —').length).toBeGreaterThan(0);
  });

  it('selects a month directly without asking for a day', () => {
    render(<BankTransferHistoryPageContent />);

    fireEvent.click(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 07/2026' }));
    fireEvent.click(screen.getByRole('button', { name: 'Chọn tháng 08 năm 2026' }));

    expect(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 08/2026' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Chọn tháng 08 năm 2026' })).not.toBeInTheDocument();
  });

  it('scopes the admin card treatment to the admin variant', () => {
    const { container, rerender } = render(<BankTransferHistoryPageContent />);

    expect(container.querySelector('.admin-payment-history-workspace')).not.toBeInTheDocument();

    rerender(<BankTransferHistoryPageContent variant="admin" />);

    expect(container.querySelector('.admin-payment-history-workspace')).toHaveClass('rounded-2xl');
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

  it('answers the month total in one slim band and paginates without leaving the module', () => {
    const result = useBankTransferHistories();
    result.data.pagination = { page: 1, pageSize: 20, totalPages: 11, totalRecords: 207 };
    useBankTransferHistories.mockReturnValue(result);

    const { container } = render(<BankTransferHistoryPageContent />);

    // Summary is one slim line inside the workspace — no KPI cards.
    const summary = within(container.querySelector('[data-slot="payment-history-summary"]')!);
    expect(summary.getByText('Tổng đã chuyển')).toBeInTheDocument();
    expect(summary.getByText('bút toán ngân hàng')).toBeInTheDocument();
    expect(summary.getByText('nhân viên')).toBeInTheDocument();
    expect(summary.getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();
    expect(summary.getByText('07/2026')).toBeInTheDocument();
    expect(container.querySelector('.shadow-soft')).not.toBeInTheDocument();

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

  it('renders a semantic ledger table with a sticky header and flat hairline rows', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const ledger = container.querySelector('[data-slot="payment-history-records"]')!;
    const table = ledger.querySelector('table')!;

    // Real table semantics — not a grid of divs.
    expect(table).toBeInTheDocument();
    expect(table.querySelector('thead')).toBeInTheDocument();
    expect(table.querySelector('thead')!).toHaveClass('sticky', 'top-0', 'z-10');

    // Every column header is scoped and labeled.
    const headers = Array.from(table.querySelectorAll<HTMLTableCellElement>('thead th'));
    expect(headers.map((th) => th.textContent)).toEqual([
      'STT', 'Nhân viên', 'Kỳ thanh toán', 'Ngày thanh toán',
      'Ghi chú CK', 'Mã GD ngân hàng', 'Thời gian xử lý', 'Số tiền', 'Thực nhận (kỳ)',
    ]);
    for (const th of headers) {
      expect(th).toHaveAttribute('scope', 'col');
    }

    // Rows are hairline ledger lines — no per-record cards, no accordion.
    const firstRow = table.querySelector('tbody tr')!;
    expect(firstRow.className).not.toContain('rounded-');
    expect(firstRow.className).not.toContain('shadow-');
    expect(ledger.querySelector('details')).not.toBeInTheDocument();
  });

  it('mirrors the ledger as dense record blocks below the table breakpoint', () => {
    const { container } = render(<BankTransferHistoryPageContent />);

    const mobile = container.querySelector<HTMLElement>('[data-slot="payment-history-mobile-records"]')!;
    expect(mobile).toHaveClass('xl:hidden');

    // Same data as the table: name+total line, metadata line, posting lines.
    expect(within(mobile).getByText('LÒ THỊ MINH THU')).toBeInTheDocument();
    expect(within(mobile).getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();
    expect(within(mobile).getByText('FT26198846619959')).toBeInTheDocument();
    expect(within(mobile).getByText('FT26198940380850')).toBeInTheDocument();
    expect(within(mobile).getByText(/031189014251/)).toBeInTheDocument();
  });

  it('requests a denser default page now that rows are compact', () => {
    render(<BankTransferHistoryPageContent />);

    expect(useBankTransferHistories).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 1, pageSize: 50 }),
    );
  });

  it('renders loading placeholders as flat ledger lines', () => {
    useBankTransferHistories.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      refetch: vi.fn(),
    });

    const { container } = render(<BankTransferHistoryPageContent />);

    const skeleton = container.querySelector('[aria-label="Đang tải lịch sử trả lương"]');
    expect(skeleton).toHaveClass('divide-y', 'divide-utility-gray-300');
    // rounded-none is load-bearing: it overrides the Skeleton base rounding so
    // placeholder rows match the flat ledger lines they stand in for.
    const rows = Array.from(skeleton!.children);
    expect(rows).toHaveLength(8);
    for (const row of rows) {
      expect(row).toHaveClass('h-16', 'rounded-none', 'xl:h-10');
    }
  });

  it('keeps the partner header below the iOS safe area at every breakpoint', () => {
    const { container } = render(<BankTransferHistoryPageContent variant="partner" />);

    expect(container.querySelector('.admin-payment-history-page')).toHaveClass(
      'pt-[calc(env(safe-area-inset-top,0px)+0.75rem)]',
      'sm:pt-[calc(env(safe-area-inset-top,0px)+1rem)]',
      'md:pt-[calc(env(safe-area-inset-top,0px)+1.5rem)]',
    );
  });

  it('rides the standard admin shell (canvas + glass header) on the admin variant', () => {
    const { container } = render(<BankTransferHistoryPageContent variant="admin" />);

    // Admin pages share the ambient canvas and glass header card idiom; the
    // legacy standalone safe-area root belongs to the partner variant only.
    expect(container.querySelector('.admin-payment-history-page')).not.toBeInTheDocument();
    expect(container.querySelector('[data-mobile-header]')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: 'Lịch sử trả lương' })).toBeInTheDocument();
  });
});
