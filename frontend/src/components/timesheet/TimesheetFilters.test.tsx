import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { TimesheetFilters } from './TimesheetFilters';

const useTimesheetContextMock = vi.hoisted(() => vi.fn());
const useIsMobileMock = vi.hoisted(() => vi.fn(() => false));

vi.mock('@/hooks/useBreakpoint', () => ({
  useIsMobile: useIsMobileMock,
}));

vi.mock('@/components/timesheet/TimesheetContext', () => ({
  useTimesheetContext: useTimesheetContextMock,
}));

vi.mock('@/components/ui/searchable-dropdown', () => ({
  SearchableDropdown: ({ className }: { className?: string }) => (
    <button data-testid="project-filter" className={className}>Dự án</button>
  ),
}));

vi.mock('@/components/timesheet/EmployeeDropdownAdapter', () => ({
  EmployeeDropdownAdapter: ({ className }: { className?: string }) => (
    <button data-testid="employee-filter" className={className}>Tất cả nhân viên</button>
  ),
}));

describe('TimesheetFilters', () => {
  beforeEach(() => {
    useIsMobileMock.mockReturnValue(false);
    useTimesheetContextMock.mockReturnValue({
      filters: {
        selectedMonth: '2026-07',
        onMonthChange: vi.fn(),
        selectedProject: 'all',
        onProjectChange: vi.fn(),
        selectedEmployee: 'all',
        onEmployeeChange: vi.fn(),
        statusFilter: 'all',
        onStatusChange: vi.fn(),
        projects: [],
        projectEmployees: [],
        searchTerm: '',
        onSearchChange: vi.fn(),
        userRole: 'admin',
      },
    });
  });

  it('uses the same white 44px surface for every desktop dropdown trigger', () => {
    render(<TimesheetFilters />);

    const triggers = [
      screen.getByRole('combobox', { name: 'Tháng' }),
      screen.getByRole('combobox', { name: 'Trạng thái' }),
      screen.getByTestId('project-filter'),
      screen.getByTestId('employee-filter'),
    ];

    for (const trigger of triggers) {
      expect(trigger).toHaveClass('min-h-11');
      expect(trigger).toHaveClass('bg-card');
    }
  });

  it('keeps the mobile month and filter controls at least 44px tall', () => {
    useIsMobileMock.mockReturnValue(true);
    render(<TimesheetFilters />);

    expect(screen.getByRole('combobox')).toHaveClass('h-11');
    expect(screen.getByRole('button', { name: 'Bộ lọc' })).toHaveClass('h-11', 'w-11');
  });
});
