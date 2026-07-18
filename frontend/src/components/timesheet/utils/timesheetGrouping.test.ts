import { describe, expect, it } from 'vitest';

import type { Timesheet } from '@/types/api/timesheet.types';
import {
  getPaytypeDetail,
  groupEmployeeEntriesByProject,
} from './timesheetGrouping';

function makeTimesheet(overrides: Partial<Timesheet>): Timesheet {
  return {
    id: 1,
    project_id: 10,
    employee_id: 20,
    date: '2026-07-14',
    hours_worked: 8,
    paytype: 'Có Tay Nghề.ngày thường.ca ngày',
    hour_type: 'ca ngày',
    day_type: 'ngày thường',
    payrate_id: 1,
    payrate: 43_750,
    amount: 350_000,
    status: 'approved',
    payment_status: 'paid',
    created_by: 1,
    created_at: '2026-07-14T00:00:00Z',
    updated_at: '2026-07-14T00:00:00Z',
    projectName: 'PQC Hải Phòng',
    projectCode: 'PQC-HP',
    employeeName: 'Lê Thị Hằng',
    employeeCode: '031306002190',
    ...overrides,
  };
}

describe('timesheet display grouping', () => {
  it('groups entries by project, sorts dates newest first, and totals each section', () => {
    const sections = groupEmployeeEntriesByProject([
      makeTimesheet({ id: 2, date: '2026-07-10', hours_worked: 2, amount: 120_000 }),
      makeTimesheet({ id: 1, date: '2026-07-14' }),
      makeTimesheet({ id: 3, project_id: 11, projectName: 'Cầu Đông', amount: 200_000 }),
    ]);

    expect(sections).toHaveLength(2);
    const project = sections.find((section) => section.projectId === 10);
    expect(project?.entries.map((entry) => entry.id)).toEqual([1, 2]);
    expect(project?.totalHours).toBe(10);
    expect(project?.totalAmount).toBe(470_000);
    expect(project?.positions).toEqual(['Có Tay Nghề']);
  });

  it('removes the repeated position from an expanded entry label', () => {
    expect(getPaytypeDetail('Có Tay Nghề.ngày thường.tăng ca')).toBe('ngày thường · tăng ca');
  });
});
