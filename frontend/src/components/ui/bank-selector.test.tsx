import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { BankSelector } from './bank-selector';

const bank = { id: 1, branch_name: 'Ngân hàng kiểm thử' };

vi.mock('@/hooks/useBreakpoint', () => ({ useIsMobile: vi.fn() }));
vi.mock('@/hooks/api/useBanks', () => ({
  useAllBanks: () => ({ data: [bank], isLoading: false }),
  useCreateBank: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

describe.each([false, true])('BankSelector clear control (mobile: %s)', (mobile) => {
  beforeEach(() => vi.mocked(useIsMobile).mockReturnValue(mobile));

  it('clears the selection without opening the picker or nesting interactive controls', () => {
    const onSelect = vi.fn();
    render(<BankSelector value={bank} onSelect={onSelect} />);
    const clear = screen.getByRole('button', { name: 'Xóa ngân hàng đã chọn' });
    expect(screen.getByRole('combobox', { name: `Ngân hàng: ${bank.branch_name}` })).not.toContainElement(clear);
    fireEvent.click(clear);
    expect(onSelect).toHaveBeenCalledWith(null);
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('keeps both selection and clearing disabled when the field is disabled', () => {
    const onSelect = vi.fn();
    render(<BankSelector value={bank} onSelect={onSelect} disabled />);
    expect(screen.getByRole('combobox')).toBeDisabled();
    const clear = screen.getByRole('button', { name: 'Xóa ngân hàng đã chọn' });
    expect(clear).toBeDisabled();
    fireEvent.click(clear);
    expect(onSelect).not.toHaveBeenCalled();
  });
});

it('opens the picker, searches, clears the search, and selects a bank', () => {
  vi.mocked(useIsMobile).mockReturnValue(false);
  const onSelect = vi.fn();
  render(<BankSelector onSelect={onSelect} canCreateBank={false} />);
  fireEvent.click(screen.getByRole('combobox', { name: 'Chọn ngân hàng...' }));
  const search = screen.getByRole('textbox', { name: 'Tìm kiếm ngân hàng' });
  fireEvent.change(search, { target: { value: 'Không có' } });
  expect(screen.getByText('Không tìm thấy ngân hàng nào')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Xóa tìm kiếm ngân hàng' }));
  expect(search).toHaveValue('');
  fireEvent.click(screen.getByRole('button', { name: bank.branch_name }));
  expect(onSelect).toHaveBeenCalledWith(bank);
  expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
});
