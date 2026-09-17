import type { ReactNode } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { TransactionMetadata } from '@/services/api/transaction.service';
import AddTransactionSheet from './AddTransactionSheet';

const mocks = vi.hoisted(() => ({ metadata: undefined as TransactionMetadata | undefined, missingProvider: false, loading: false, create: vi.fn() }));
vi.mock('@/contexts', () => ({ useMetadata: () => {
  if (mocks.missingProvider) throw new Error('Provider unavailable');
  return { transactionMetadata: mocks.metadata, isLoadingTransactionMetadata: mocks.loading };
} }));
vi.mock('@/hooks/useModalNavigation', () => ({ useModalNavigation: () => ({ closeModal: vi.fn() }) }));
vi.mock('@/hooks/transactions/useCreateTransaction', () => ({ useCreateTransaction: () => ({ isPending: false, mutateAsync: mocks.create }) }));
vi.mock('@/hooks/api/useAssets', () => ({ useUploadAsset: () => ({ isPending: false, mutateAsync: vi.fn() }) }));
vi.mock('@/components/sheets/templates/SlideSheetTemplate', () => ({ SlideSheetTemplate: ({ children, footer }: { children: ReactNode; footer: ReactNode }) => <div>{children}{footer}</div> }));
vi.mock('@/components/ui/user-selector', () => ({ UserSelector: () => <button>Người góp vốn</button> }));

beforeEach(() => { mocks.metadata = undefined; mocks.missingProvider = false; mocks.loading = false; mocks.create.mockReset(); });
function renderForm() {
  return render(<MemoryRouter initialEntries={['/admin?modal=add_ledger_entry']}><AddTransactionSheet isOpen onClose={vi.fn()} /></MemoryRouter>);
}

describe('AddTransactionSheet metadata readiness', () => {
  it.each(['provider absent', 'metadata unavailable', 'metadata empty'] as const)('renders usable status choices with %s', (state) => {
    mocks.missingProvider = state === 'provider absent';
    if (state === 'metadata empty') mocks.metadata = { transaction_types: [], statuses: [] };
    renderForm();
    expect(screen.getByRole('button', { name: 'Chi phí' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Chờ TT' })).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Đã TT' }));
    expect(screen.getByRole('button', { name: 'Lưu' })).toBeEnabled();
    expect(mocks.create).not.toHaveBeenCalled();
  });
  it('uses loaded metadata and omits partial settlements from new transactions', () => {
    mocks.metadata = { transaction_types: [{ type: 'expense', label: 'Chi phí từ cấu hình' }], statuses: [{ type: 'pending', label: 'Pending' }, { type: 'partially_settled', label: 'Partial' }, { type: 'settled', label: 'Settled' }] };
    renderForm();
    expect(screen.getByRole('button', { name: 'Chi phí từ cấu hình' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Partial' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Chờ TT' })).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
