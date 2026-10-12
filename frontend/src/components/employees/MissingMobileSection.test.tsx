import { fireEvent, render, screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  useMissingMobile,
  useDuplicateMobiles,
} from '@/hooks/employees/useMissingMobile';
import { useIsMobile } from '@/hooks/useBreakpoint';
import type { Employee, DuplicateMobileGroup } from '@/types/api/employee.types';
import { MissingMobileSection } from './MissingMobileSection';

vi.mock('@/hooks/employees/useMissingMobile', () => ({
  useMissingMobile: vi.fn(),
  useDuplicateMobiles: vi.fn(),
}));

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: vi.fn(),
}));

const missingEmployees = [
  {
    id: 1,
    fullname: 'Nguyễn Thị Không SĐT',
    cccd: '001',
    mobile: '',
    current_projects: [],
  },
  {
    id: 2,
    fullname: 'Trần Văn Chưa Có SĐT',
    cccd: '002',
    mobile: '',
    current_projects: [],
  },
] as Employee[];

const duplicateGroups: DuplicateMobileGroup[] = [
  {
    mobile: '0912345678',
    employees: [
      { id: 3, fullname: 'Lê Văn Trùng Một' },
      { id: 4, fullname: 'Lê Văn Trùng Hai' },
    ],
    hidden_count: 2,
  },
  {
    mobile: '0987654321',
    employees: [{ id: 5, fullname: 'Phạm Thị Năm' }],
    hidden_count: 1,
  },
];

describe('MissingMobileSection', () => {
  beforeEach(() => {
    vi.mocked(useMissingMobile).mockReturnValue({
      employees: missingEmployees,
      totalCount: missingEmployees.length,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    vi.mocked(useDuplicateMobiles).mockReturnValue({
      groups: duplicateGroups,
      totalEmployees: 3,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    vi.mocked(useIsMobile).mockReturnValue(false);
  });

  it('renders nothing while loading (no dead card)', () => {
    vi.mocked(useMissingMobile).mockReturnValue({
      employees: [],
      totalCount: 0,
      isLoading: true,
      error: null,
      refetch: vi.fn(),
    });
    vi.mocked(useDuplicateMobiles).mockReturnValue({
      groups: [],
      totalEmployees: 0,
      isLoading: true,
      error: null,
      refetch: vi.fn(),
    });

    const { container } = render(<MissingMobileSection />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders nothing when both lists are clean (no dead card)', () => {
    vi.mocked(useMissingMobile).mockReturnValue({
      employees: [],
      totalCount: 0,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
    vi.mocked(useDuplicateMobiles).mockReturnValue({
      groups: [],
      totalEmployees: 0,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });

    const { container } = render(<MissingMobileSection />);
    expect(container).toBeEmptyDOMElement();
  });

  it('shows split counts per kind in the header and explains why it matters', () => {
    render(<MissingMobileSection />);

    // Header chips render the label and its count as separate nodes, so match
    // on the chip element rather than one concatenated string.
    const headerChips = screen.getAllByText(
      (_, el) =>
        el?.tagName === 'SPAN' &&
        /^(Thiếu số điện thoại|Số điện thoại trùng)\s*\d+$/.test(el.textContent ?? ''),
    );
    expect(headerChips.map(c => c.textContent)).toEqual(
      expect.arrayContaining(['Thiếu số điện thoại2', 'Số điện thoại trùng3']),
    );
    expect(screen.getByText(/không thể đăng nhập bằng SĐT/)).toBeInTheDocument();
  });

  it('groups duplicate rows under their shared number and names co-holders', () => {
    // Rows only carry activation semantics (and their accessible name) when
    // an onEmployeeClick handler is supplied — same as the cloned section.
    render(<MissingMobileSection onEmployeeClick={vi.fn()} />);
    fireEvent.click(screen.getByRole('button', { name: 'Xem danh sách' }));

    const region = screen.getByRole('region', {
      name: 'Danh sách số điện thoại trùng',
    });

    // The group header counts every holder of the number: in-scope members
    // plus the ones outside the caller's scope.
    expect(within(region).getByText('4 nhân viên dùng chung')).toBeInTheDocument();

    // One row per in-scope member: each names its in-scope co-holders and
    // counts the out-of-scope ones without ever inventing names for them.
    const row = within(region).getByRole('row', {
      name: 'Mở thông tin nhân viên Lê Văn Trùng Một',
    });
    expect(within(row).getByText('Dùng chung với: Lê Văn Trùng Hai')).toBeInTheDocument();
    expect(within(row).getByText('2 nhân viên ngoài phạm vi')).toBeInTheDocument();

    // A member whose only co-holders are out of scope shows just the count.
    const hiddenOnlyRow = within(region).getByRole('row', {
      name: 'Mở thông tin nhân viên Phạm Thị Năm',
    });
    expect(within(hiddenOnlyRow).queryByText(/Dùng chung với/)).not.toBeInTheDocument();
    expect(within(hiddenOnlyRow).getByText('1 nhân viên ngoài phạm vi')).toBeInTheDocument();
  });

  it('keeps optional row activation available to pointer and keyboard users', () => {
    const handleEmployeeClick = vi.fn();
    render(<MissingMobileSection onEmployeeClick={handleEmployeeClick} />);
    fireEvent.click(screen.getByRole('button', { name: 'Xem danh sách' }));

    const row = screen.getByRole('row', {
      name: 'Mở thông tin nhân viên Nguyễn Thị Không SĐT',
    });
    fireEvent.click(row);
    fireEvent.keyDown(row, { key: 'Enter' });

    expect(handleEmployeeClick).toHaveBeenCalledTimes(2);
    expect(handleEmployeeClick).toHaveBeenLastCalledWith(missingEmployees[0]);

    // Duplicate rows only carry id + fullname from the API; the click payload
    // fills the remaining Employee fields with empty values.
    const duplicateRow = screen.getByRole('row', {
      name: 'Mở thông tin nhân viên Lê Văn Trùng Một',
    });
    fireEvent.click(duplicateRow);
    expect(handleEmployeeClick).toHaveBeenLastCalledWith({
      id: 3,
      fullname: 'Lê Văn Trùng Một',
      cccd: '',
      current_projects: [],
    });
  });

  it('switches to card lists on phones but keeps collision groups intact', () => {
    const handleEmployeeClick = vi.fn();
    vi.mocked(useIsMobile).mockReturnValue(true);
    render(<MissingMobileSection onEmployeeClick={handleEmployeeClick} />);
    fireEvent.click(screen.getByRole('button', { name: 'Xem danh sách' }));

    // Phones get card lists; the table is a desktop-density view — only one
    // variant exists in the DOM at a time.
    expect(screen.queryByRole('row')).not.toBeInTheDocument();
    expect(
      screen.getByRole('list', { name: 'Danh sách nhân viên thiếu số điện thoại' }),
    ).toBeInTheDocument();

    // Duplicate cards still lead with the shared number (group header + one
    // card per in-scope member).
    expect(screen.getAllByText('0912345678')).toHaveLength(3);
    expect(screen.getByText('4 nhân viên dùng chung')).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: 'Mở thông tin nhân viên Lê Văn Trùng Một' }),
    );
    expect(handleEmployeeClick).toHaveBeenCalledWith({
      id: 3,
      fullname: 'Lê Văn Trùng Một',
      cccd: '',
      current_projects: [],
    });
  });
});
