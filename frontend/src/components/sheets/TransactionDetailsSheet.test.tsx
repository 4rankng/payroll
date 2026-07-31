import type { ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type { Transaction } from '@/services/api/transaction.service';
import TransactionDetailsSheet from './TransactionDetailsSheet';

const transaction = {
  id: 123,
  description: 'ChatGPT 20X Upgrade (Initially 5X)',
  transaction_code: 'TXN-123',
  transaction_type: 'expense',
  amount: 3_600_000,
  settled_amount: 3_600_000,
  remaining_amount: 0,
  party: 'Frank Nguyen',
  status: 'settled',
  settlement_method: 'cash',
  created_by: 1,
  created_at: '2026-07-23T16:45:00+07:00',
  updated_at: '2026-07-31T12:00:00+07:00',
  settlements: [
    {
      id: 9,
      transaction_id: 123,
      amount: 3_600_000,
      settlement_date: '2026-07-31',
      created_by: 1,
      created_at: '2026-07-31T12:00:00+07:00',
    },
  ],
} satisfies Transaction;

vi.mock('react-router-dom', () => ({
  useSearchParams: () => [new URLSearchParams('modal=transaction_details&id=123')],
}));

vi.mock('@/hooks/useModalNavigation', () => ({
  useModalNavigation: () => ({
    openModal: vi.fn(),
    closeModal: vi.fn(),
  }),
}));

vi.mock('@/hooks/transactions/useTransactions', () => ({
  useTransaction: () => ({ data: transaction, isLoading: false }),
}));

vi.mock('@/hooks/transactions/useTransactionMetadata', () => ({
  useTransactionMetadata: () => ({ data: undefined }),
}));

vi.mock('@/hooks/api/useUsers', () => ({
  useUsersByIds: () => ({
    data: new Map([[1, { id: 1, fullname: 'Frank Nguyen' }]]),
    isLoading: false,
  }),
}));

vi.mock('@/hooks/api/useAssets', () => ({
  useUploadAsset: () => ({ isPending: false, mutateAsync: vi.fn() }),
  useDownloadAsset: () => ({ mutate: vi.fn() }),
}));

vi.mock('@/hooks/transactions/useUpdateTransactionEvidence', () => ({
  useUpdateTransactionEvidence: () => ({ isPending: false, mutateAsync: vi.fn() }),
}));

vi.mock('@/hooks/transactions/useDeleteTransaction', () => ({
  useDeleteTransaction: () => ({ isPending: false, mutate: vi.fn() }),
}));

vi.mock('@/components/sheets/templates/SlideSheetTemplate', () => ({
  SlideSheetTemplate: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

describe('TransactionDetailsSheet', () => {
  it('renders the settlement business date instead of the transaction creation time', () => {
    render(<TransactionDetailsSheet isOpen onClose={vi.fn()} />);

    const paymentDateBlock = screen.getByText('Ngày thanh toán').parentElement;

    expect(paymentDateBlock).toHaveTextContent('31 tháng 07 2026');
    expect(paymentDateBlock).not.toHaveTextContent('23 tháng 07 2026 lúc 16:45');
  });
});
