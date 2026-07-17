import { describe, expect, it } from 'vitest';

import { buildTimesheetStatusFilters } from './timesheetFilterHelpers';

describe('buildTimesheetStatusFilters', () => {
  it('uses the backend pending-payment cohort token', () => {
    expect(buildTimesheetStatusFilters('pending_payment')).toEqual({
      status: 'pending_payment',
    });
  });

  it('keeps ordinary approval and payment filters unchanged', () => {
    expect(buildTimesheetStatusFilters('pending_approval')).toEqual({
      status: 'pending_approval',
    });
    expect(buildTimesheetStatusFilters('paid')).toEqual({ status: 'paid' });
    expect(buildTimesheetStatusFilters('failed')).toEqual({ status: 'failed' });
    expect(buildTimesheetStatusFilters('cancelled')).toEqual({ status: 'cancelled' });
    expect(buildTimesheetStatusFilters('all')).toEqual({});
  });
});
