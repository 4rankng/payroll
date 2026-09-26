import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import WalletPage from './index';

const employeeAccountLookupDialogProps = vi.fn();

vi.mock('@tanstack/react-query', () => ({
  useQuery: () => ({ data: { available: 12000000, pending_out: 3000000, as_of: null } }),
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock('@/components/wallet/WalletTransactionsList', () => ({ default: () => null }));
vi.mock('@/components/wallet/CreateManualDisbursementDialog', () => ({ default: () => null }));
vi.mock('@/components/wallet/BulkTransferBatchList', () => ({ BulkTransferBatchList: () => null }));
vi.mock('@/components/wallet/BulkTransferUploadDialog', () => ({ BulkTransferUploadDialog: () => null }));
vi.mock('@/components/wallet/EmployeeAccountLookupDialog', () => ({
  EmployeeAccountLookupDialog: (props: { open: boolean }) => {
    employeeAccountLookupDialogProps(props);
    return props.open ? <div>Hộp thoại tra cứu tài khoản</div> : null;
  },
}));

describe('WalletPage', () => {
  it('exposes the employee account lookup action in the desktop wallet header', () => {
    render(<WalletPage />);

    fireEvent.click(screen.getByRole('button', { name: 'Tra cứu' }));
    expect(screen.getByText('Hộp thoại tra cứu tài khoản')).toBeInTheDocument();
    expect(employeeAccountLookupDialogProps).toHaveBeenLastCalledWith(
      expect.objectContaining({ open: true }),
    );
  });
});
