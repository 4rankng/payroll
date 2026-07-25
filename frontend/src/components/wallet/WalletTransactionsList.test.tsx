import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import WalletTransactionsList from './WalletTransactionsList';

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: () => true,
}));

vi.mock('@tanstack/react-query', () => ({
  useQuery: () => ({
    data: { data: [], total: 0 },
    isLoading: false,
    isFetching: false,
    isError: false,
    refetch: vi.fn(),
  }),
}));

describe('WalletTransactionsList', () => {
  it('uses mobile-sized refresh and date controls', () => {
    render(<WalletTransactionsList />);

    expect(screen.getByRole('button', { name: 'Làm mới' })).toHaveClass(
      'h-11',
      'min-h-11',
    );
    expect(screen.getByLabelText('Ngày bắt đầu')).toHaveClass('h-11');
    expect(screen.getByLabelText('Ngày kết thúc')).toHaveClass('h-11');
  });
});
