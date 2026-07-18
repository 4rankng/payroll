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
          project_ids: [1],
          project_names: ['Công trường Việt Nam'],
          work_month: '2026-07',
          cycle: 2,
          from_date: '2026-07-08',
          to_date: '2026-07-14',
          payment_date: '2026-07-17',
          total_amount: 1_998_000,
          transfers: [
            { bank_reference: 'FT26198846619959', amount: 1_548_000 },
            { bank_reference: 'FT26198940380850', amount: 450_000 },
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
    expect(screen.getByText(/1\.548\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText(/450\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText(/1\.998\.000\s*₫/)).toBeInTheDocument();
    expect(screen.getByText('Kỳ 2 · ngày 8–14')).toBeInTheDocument();
    expect(screen.getByText('2 giao dịch')).toBeInTheDocument();
  });
});
