import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { WalletBalanceCard } from './WalletBalanceCard';
import type { WalletBalance } from '@/types/api/wallet.types';

const MOCK_BALANCE = vi.hoisted(
  () =>
    ({
      available: 5_000_000,
      pending_in: 0,
      pending_out: 1_250_000,
      limbo: 0,
      currency: 'VND',
      as_of: '2026-09-05T10:00:00+07:00',
    }) as WalletBalance,
);

vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({
    refetchQueries: vi.fn().mockResolvedValue(undefined),
    invalidateQueries: vi.fn(),
  }),
  // The card only consumes the wallet-balance query's data (as_of,
  // pending_out); the query never runs under this mock.
  useQuery: () => ({ data: MOCK_BALANCE }),
}));

vi.mock('@/hooks/useDisbursementSettings', () => ({
  useDisbursementSettings: () => ({
    isLoading: false,
    data: {
      internal_balance: { available: 5_000_000 },
      active_provider: { capabilities: { balance_inquiry: false } },
    },
  }),
}));

describe('WalletBalanceCard', () => {
  it('keeps the mobile balance-sync action at least 44px', () => {
    render(<WalletBalanceCard />);

    expect(screen.getByRole('button', { name: 'Đồng bộ số dư' })).toHaveClass(
      'h-11',
      'w-11',
      'touch-manipulation',
    );
  });

  it('shows the pending-out companion rail when fee props are absent (wallet page)', () => {
    render(<WalletBalanceCard />);

    expect(screen.getByText('Đang chi trả')).toBeInTheDocument();
    expect(screen.getByText(/1\.250\.000/)).toBeInTheDocument();
    expect(screen.queryByText('Phí tháng này')).not.toBeInTheDocument();
  });

  it('shows the as-of meta line in non-compact mode', () => {
    render(<WalletBalanceCard />);

    expect(screen.getByText(/Cập nhật/)).toBeInTheDocument();
  });

  it('keeps the fee rail when fee props are provided (band usage)', () => {
    render(<WalletBalanceCard compact monthlyProviderFee={80_000} totalProviderFee={960_000} />);

    expect(screen.getByText('Phí tháng này')).toBeInTheDocument();
    expect(screen.queryByText('Đang chi trả')).not.toBeInTheDocument();
    expect(screen.queryByText(/Cập nhật/)).not.toBeInTheDocument();
  });
});
