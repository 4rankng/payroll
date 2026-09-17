import { useState } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import type { Timesheet } from '@/types/api/timesheet.types';
import { TimesheetGroupedTable } from './TimesheetGroupedTable';

const timesheet: Timesheet = {
  id: 1, project_id: 10, employee_id: 20, date: '2026-07-14',
  hours_worked: 8, paytype: 'Có Tay Nghề.ngày thường.ca ngày',
  hour_type: 'ca ngày', day_type: 'ngày thường', payrate_id: 1,
  payrate: 43_750, amount: 350_000, status: 'approved', payment_status: 'paid',
  created_by: 1, created_at: '2026-07-14T00:00:00Z', updated_at: '2026-07-14T00:00:00Z',
  projectName: 'PQC Hải Phòng', projectCode: 'PQC-HP', employeeName: 'Lê Thị Hằng', employeeCode: '031306002190',
};

describe.each(['admin', 'partner'] as const)('%s grouped timesheet table', (userRole) => {
  it('uses a named button to expand a semantic table row without double toggling', () => {
    const onRowClick = vi.fn();
    function TableHarness() {
      const [expandedGroups, setExpandedGroups] = useState(new Set<string>());
      return <TimesheetGroupedTable
        timesheets={[timesheet]}
        userRole={userRole}
        expandedGroups={expandedGroups}
        onRowClick={onRowClick}
        onToggleGroup={(key) => setExpandedGroups((current) => current.has(key) ? new Set() : new Set([key]))}
      />;
    }
    render(<MemoryRouter><TableHarness /></MemoryRouter>);
    const toggle = screen.getByRole('button', { name: 'Xem bảng công của Lê Thị Hằng' });
    const row = toggle.closest('tr')!;
    expect(row).not.toHaveAttribute('aria-expanded');
    expect(row).not.toHaveAttribute('type');
    expect(toggle).toHaveAttribute('aria-expanded', 'false');
    const collapsedRowCount = screen.getAllByRole('row').length;
    fireEvent.click(toggle);
    expect(screen.getByRole('button', { name: 'Thu gọn bảng công của Lê Thị Hằng' })).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getAllByRole('row').length).toBeGreaterThan(collapsedRowCount);
    expect(onRowClick).not.toHaveBeenCalled();
    fireEvent.click(row);
    expect(screen.getByRole('button', { name: 'Xem bảng công của Lê Thị Hằng' })).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getAllByRole('row')).toHaveLength(collapsedRowCount);
  });
});
