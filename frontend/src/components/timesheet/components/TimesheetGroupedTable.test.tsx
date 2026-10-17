import { useState } from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import type { Timesheet } from '@/types/api/timesheet.types';
import { TimesheetGroupedTable } from './TimesheetGroupedTable';
import { groupTimesheetsByEmployee } from '../utils/timesheetGrouping';

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

  // Regression guard: the table card must not be painted on the tinted page
  // canvas (read as a bare wireframe) nor on a glaring white surface. It sits
  // on the muted gray surface with a deeper header band.
  it('paints the rows on a muted surface with a deeper header band', () => {
    render(
      <MemoryRouter>
        <TimesheetGroupedTable
          timesheets={[timesheet]}
          userRole={userRole}
          expandedGroups={new Set()}
          onRowClick={() => {}}
          onToggleGroup={() => {}}
        />
      </MemoryRouter>,
    );

    const table = screen.getByRole('table');
    const card = table.parentElement?.parentElement as HTMLElement;
    expect(card.className).toContain('bg-utility-gray-100');
    expect(card.className).not.toContain('bg-card');
    expect((table.querySelector('thead tr') as HTMLElement).className).toContain('bg-utility-gray-200');
  });

  // Regression guard: the expanded entry rows used to be white like the
  // aggregate row, so a group and its entries read as one flat, bright block.
  // The entries must sit on a deeper surface than the aggregate row.
  it('sinks the expanded entry rows below the aggregate row surface', () => {
    const groupKey = groupTimesheetsByEmployee([timesheet])[0].groupKey;
    render(
      <MemoryRouter>
        <TimesheetGroupedTable
          timesheets={[timesheet]}
          userRole={userRole}
          expandedGroups={new Set([groupKey])}
          onRowClick={() => {}}
          onToggleGroup={() => {}}
        />
      </MemoryRouter>,
    );

    const rows = screen.getAllByRole('row').filter((row) => row.querySelector('td'));
    const aggregateRow = rows.find((row) => row.querySelector('button[aria-label$="bảng công của Lê Thị Hằng"]'))!;
    const entryRows = rows.filter((row) => row !== aggregateRow);
    expect(aggregateRow.className.split(/\s+/).filter((token) => token.startsWith('bg-utility-'))).toEqual([]);
    expect(entryRows.length).toBeGreaterThan(0);
    for (const row of entryRows) {
      expect(row.className.split(/\s+/).some((token) => token.startsWith('bg-utility-'))).toBe(true);
    }
  });
});
