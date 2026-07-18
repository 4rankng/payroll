import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { BankTransferHistoryPageContent } from './BankTransferHistoryPageContent';

const useBankTransferHistories = vi.fn();

vi.mock('@/hooks/api/usePayrolls', () => ({
  useBankTransferHistories: (...args: unknown[]) => useBankTransferHistories(...args),
}));

describe('BankTransferHistoryPageContent', () => {
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
          project_names: ['Công trường Việt Nam'],
          work_month: '2026-07',
          cycle: 2,
          from_date: '2026-07-08',
          to_date: '2026-07-14',
          payment_date: '2026-07-17',
          total_amount: 1_998_000,
          transfers: [
            { transfer_code: 'VFIC6d037214', bank_reference: 'FT26198846619959', amount: 1_548_000 },
            { transfer_code: 'VFIC7a193042', bank_reference: 'FT26198940380850', amount: 450_000 },
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

    expect(screen.getByText('FT26198846619959')).toBeInTheDocument();
    expect(screen.getByText('FT26198940380850')).toBeInTheDocument();
    expect(screen.getByText('VFIC6d037214')).toBeInTheDocument();
    expect(screen.getByText('VFIC7a193042')).toBeInTheDocument();
    expect(screen.getAllByText('Ghi chú:')).toHaveLength(2);
    expect(screen.getAllByText('Mã GD:')).toHaveLength(2);
    expect(screen.getByText('CCCD: 031189014251')).toBeInTheDocument();
    expect(screen.getAllByText('Chi tiết thanh toán')).toHaveLength(2);
    expect(screen.getByText(/1\.548\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText(/450\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Chọn tháng kỳ lương, hiện tại 07/2026' })).toHaveTextContent('07/2026');
    expect(screen.queryByText('July 2026')).not.toBeInTheDocument();
    expect(screen.getByText('Kỳ 2')).toBeInTheDocument();
    expect(screen.getByText('08–14/07/2026')).toHaveAttribute('aria-label', 'Từ 08/07/2026 đến 14/07/2026');
    expect(screen.getByText('17/07/2026')).toBeInTheDocument();
    expect(screen.getByText('2 giao dịch')).toBeInTheDocument();
  });

  it('shows an empty-note fallback when a legacy payment has no transfer code', () => {
    const result = useBankTransferHistories();
    result.data.data[0].employee_cccd = '';
    result.data.data[0].transfers = [
      { transfer_code: '', bank_reference: 'FT-LEGACY-001', amount: 1_998_000 },
    ];
    useBankTransferHistories.mockReturnValue(result);

    render(<BankTransferHistoryPageContent />);

    expect(screen.getByText('FT-LEGACY-001')).toBeInTheDocument();
    expect(screen.getByText('Ghi chú:')).toBeInTheDocument();
    expect(screen.getByText('Mã GD:')).toBeInTheDocument();
    expect(screen.getByText('—')).toBeInTheDocument();
    expect(screen.getByText('CCCD: —')).toBeInTheDocument();
  });
});
