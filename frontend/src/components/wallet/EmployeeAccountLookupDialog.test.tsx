import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { EmployeeAccountLookupDialog } from './EmployeeAccountLookupDialog';

const lookupMutation = vi.hoisted(() => ({
  data: null as unknown,
  error: null as unknown,
  isError: false,
  isPending: false,
  mutate: vi.fn(),
  reset: vi.fn(),
}));

const customLookupMutation = vi.hoisted(() => ({
  data: null as unknown,
  error: null as unknown,
  isError: false,
  isPending: false,
  mutate: vi.fn(),
  reset: vi.fn(),
}));

vi.mock('@/hooks/api/useManualDisbursement', () => ({
  useEmployeeAccountLookup: () => lookupMutation,
  useVerifyManualDisbursementAccount: () => customLookupMutation,
  useManualDisbursementBanks: () => ({
    data: [{ bank_code: 'VCB', swift_code: 'BFTVVNVX', bank_name: 'Vietcombank' }],
    isLoading: false,
  }),
}));

vi.mock('@/components/ui/EmployeeSingleSelector', () => ({
  EmployeeSingleSelector: ({ onSelect, disabled }: { onSelect: (employee: unknown) => void; disabled?: boolean }) => (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onSelect({ id: 21, fullname: 'Nguyễn Thị An', cccd: '001002003004' })}
    >
      Chọn Nguyễn Thị An
    </button>
  ),
}));

describe('EmployeeAccountLookupDialog', () => {
  beforeEach(() => {
    lookupMutation.data = null;
    lookupMutation.error = null;
    lookupMutation.isError = false;
    lookupMutation.isPending = false;
    lookupMutation.mutate.mockClear();
    lookupMutation.reset.mockClear();
    customLookupMutation.data = null;
    customLookupMutation.error = null;
    customLookupMutation.isError = false;
    customLookupMutation.isPending = false;
    customLookupMutation.mutate.mockClear();
    customLookupMutation.reset.mockClear();
  });

  it('sends only the selected employee ID after explicit confirmation', () => {
    const onOpenChange = vi.fn();
    render(<EmployeeAccountLookupDialog open onOpenChange={onOpenChange} />);

    const lookupButton = screen.getByRole('button', { name: 'Tra cứu' });
    expect(lookupButton).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: 'Chọn Nguyễn Thị An' }));
    fireEvent.click(lookupButton);

    expect(lookupMutation.mutate).toHaveBeenCalledWith({ employee_id: 21 });
    expect(lookupMutation.reset).toHaveBeenCalled();
  });

  it('renders a provider-confirmed name mismatch separately from stored data', () => {
    lookupMutation.data = {
      employee: { id: 21, fullname: 'Nguyễn Thị An' },
      stored_bank: {
        bank_id: 1,
        bank_name: 'Vietcombank',
        bank_code: 'VCB',
        swift_code: 'BFTVVNVX',
        account_number: '123456789',
        account_name: 'NGUYEN THI AN',
      },
      outcome: 'name_mismatch',
      provider_result: {
        Valid: false,
        BankCode: 'VCB',
        AccountNo: '123456789',
        AccountName: 'NGUYEN THI ANH',
        AccountType: '0',
        RawErrorCode: '12',
        RawMessage: 'Tên không khớp',
      },
    };

    render(<EmployeeAccountLookupDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByText('Tên chủ tài khoản không khớp')).toBeInTheDocument();
    expect(screen.getByText('NGUYEN THI AN')).toBeInTheDocument();
    expect(screen.getByText('NGUYEN THI ANH')).toBeInTheDocument();
  });

  it('resets lookup state and closes deterministically', () => {
    const onOpenChange = vi.fn();
    render(<EmployeeAccountLookupDialog open onOpenChange={onOpenChange} />);

    fireEvent.click(screen.getByRole('button', { name: 'Chọn Nguyễn Thị An' }));
    fireEvent.click(screen.getByRole('button', { name: 'Đóng' }));

    expect(lookupMutation.reset).toHaveBeenCalled();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('shows an actionable provider error and permits retry', () => {
    lookupMutation.isError = true;
    lookupMutation.error = {
      code: 'account_verification_provider_error',
      message: 'provider failed',
    };
    render(<EmployeeAccountLookupDialog open onOpenChange={vi.fn()} />);

    expect(screen.getByRole('alert')).toHaveTextContent('Nhà cung cấp không thể xác minh tài khoản lúc này');
    fireEvent.click(screen.getByRole('button', { name: 'Chọn Nguyễn Thị An' }));
    fireEvent.click(screen.getByRole('button', { name: 'Tra cứu' }));

    expect(lookupMutation.mutate).toHaveBeenCalledWith({ employee_id: 21 });
  });

  it('allows a one-off custom lookup without an employee ID', () => {
    render(<EmployeeAccountLookupDialog open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByRole('tab', { name: 'Nhập thủ công' }));
    fireEvent.change(screen.getByLabelText('Mã SWIFT'), { target: { value: 'bftvvnvx' } });
    fireEvent.change(screen.getByLabelText('Số tài khoản'), { target: { value: '123 456abc789' } });
    fireEvent.change(screen.getByLabelText('Tên chủ tài khoản'), { target: { value: 'NGUYEN VAN B' } });
    fireEvent.click(screen.getByRole('button', { name: 'Tra cứu' }));

    expect(customLookupMutation.mutate).toHaveBeenCalledWith({
      bank_code: 'BFTVVNVX',
      account_no: '123456789',
      account_name: 'NGUYEN VAN B',
      account_type: '0',
    });
    expect(lookupMutation.mutate).not.toHaveBeenCalled();
  });

  it('blocks malformed custom values with inline guidance', () => {
    render(<EmployeeAccountLookupDialog open onOpenChange={vi.fn()} />);

    fireEvent.click(screen.getByRole('tab', { name: 'Nhập thủ công' }));
    fireEvent.change(screen.getByLabelText('Mã SWIFT'), { target: { value: 'bad' } });
    fireEvent.change(screen.getByLabelText('Số tài khoản'), { target: { value: '12345' } });
    fireEvent.change(screen.getByLabelText('Tên chủ tài khoản'), { target: { value: 'A' } });

    expect(screen.getByText('Mã SWIFT phải có 8–11 ký tự chữ hoặc số.')).toBeInTheDocument();
    expect(screen.getByText('Số tài khoản phải có 6–20 chữ số.')).toBeInTheDocument();
    expect(screen.getByText('Tên chủ tài khoản phải có 3–100 ký tự.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Tra cứu' })).toBeDisabled();
    expect(customLookupMutation.mutate).not.toHaveBeenCalled();
  });
});
