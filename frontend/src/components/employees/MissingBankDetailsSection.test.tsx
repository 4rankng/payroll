import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useMissingBankDetails } from '@/hooks/employees/useMissingBankDetails';
import type { Employee } from '@/types/api/employee.types';
import { MissingBankDetailsSection } from './MissingBankDetailsSection';

vi.mock('@/hooks/employees/useMissingBankDetails', () => ({
  useMissingBankDetails: vi.fn(),
}));

const employees = [
  {
    id: 1,
    fullname: 'Nguyễn Thiếu Ngân Hàng',
    cccd: '001',
    bank: null,
    bank_account_number: '',
    bank_account_name: '',
    current_projects: [],
  },
  {
    id: 2,
    fullname: 'Nguyễn Thiếu Số',
    cccd: '002',
    bank: { id: 1 },
    bank_account_number: '',
    bank_account_name: 'NGUYEN THIEU SO',
    current_projects: [],
  },
  {
    id: 3,
    fullname: 'Nguyễn Thiếu Tên',
    cccd: '003',
    bank: { id: 1 },
    bank_account_number: '123',
    bank_account_name: '',
    current_projects: [],
  },
  {
    id: 4,
    fullname: 'Nguyễn Sai Tài Khoản',
    cccd: '004',
    bank: { id: 1 },
    bank_account_number: '456',
    bank_account_name: 'NGUYEN SAI TAI KHOAN',
    bank_account_status: 'invalid',
    bank_account_invalid_reason: 'OnePay: invalid account info',
    current_projects: [],
  },
] as Employee[];

describe('MissingBankDetailsSection', () => {
  beforeEach(() => {
    vi.mocked(useMissingBankDetails).mockReturnValue({
      employees,
      totalCount: employees.length,
      isLoading: false,
    });
  });

  it('uses one stable title and shows a safe precise reason for every row', () => {
    render(<MissingBankDetailsSection />);

    expect(screen.getByText('Thông tin ngân hàng không hợp lệ')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Xem danh sách' }));

    expect(screen.getByText('Thiếu ngân hàng')).toBeInTheDocument();
    expect(screen.getByText('Thiếu số tài khoản')).toBeInTheDocument();
    expect(screen.getByText('Thiếu tên chủ tài khoản')).toBeInTheDocument();
    expect(screen.getByText('Tài khoản ngân hàng không hợp lệ')).toBeInTheDocument();
    expect(screen.queryByText(/OnePay|invalid account/i)).not.toBeInTheDocument();
  });

  it('shows split counts for invalid vs missing kinds in the header', () => {
    render(<MissingBankDetailsSection />);

    expect(screen.getByText('Sai thông tin: 1')).toBeInTheDocument();
    expect(screen.getByText('Thiếu thông tin: 3')).toBeInTheDocument();
  });

  it('orders invalid rows first and labels each row with its kind', () => {
    render(<MissingBankDetailsSection />);
    fireEvent.click(screen.getByRole('button', { name: 'Xem danh sách' }));

    const kindBadges = screen.getAllByText(/^(Sai thông tin|Thiếu thông tin)$/);
    // Rows are re-ordered invalid-first: the sole invalid employee (id 4)
    // renders before the three missing-info employees.
    expect(kindBadges).toHaveLength(4);
    expect(kindBadges[0]).toHaveTextContent('Sai thông tin');
    expect(kindBadges.slice(1).every(b => b.textContent === 'Thiếu thông tin')).toBe(true);
  });

  it('keeps optional row activation available to pointer and keyboard users', () => {
    const handleEmployeeClick = vi.fn();
    render(<MissingBankDetailsSection onEmployeeClick={handleEmployeeClick} />);
    fireEvent.click(screen.getByRole('button', { name: 'Xem danh sách' }));

    const row = screen.getByRole('row', {
      name: 'Mở thông tin nhân viên Nguyễn Thiếu Ngân Hàng',
    });
    fireEvent.click(row);
    fireEvent.keyDown(row, { key: 'Enter' });

    expect(handleEmployeeClick).toHaveBeenCalledTimes(2);
    expect(handleEmployeeClick).toHaveBeenLastCalledWith(employees[0]);
  });
});
