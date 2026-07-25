import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { WalletBalanceCard } from './WalletBalanceCard';

vi.mock('@tanstack/react-query', () => ({
  useQueryClient: () => ({
    refetchQueries: vi.fn().mockResolvedValue(undefined),
    invalidateQueries: vi.fn(),
  }),
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
});
