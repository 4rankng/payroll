import { describe, expect, it } from 'vitest';

import { buildTimesheetStatusFilters, parseTimesheetStatusFilter } from './timesheetFilterHelpers';

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


describe('parseTimesheetStatusFilter', () => {
  it('accepts supported deep-link cohorts without broad payment-status casts', () => {
    expect(parseTimesheetStatusFilter('pending_payment')).toBe('pending_payment');
    expect(parseTimesheetStatusFilter('pending_approval')).toBe('pending_approval');
    expect(parseTimesheetStatusFilter('all')).toBe('all');
  });

  it.each([null, '', 'processing', 'disbursing', 'unknown', 'PAID'])('ignores unsupported URL status %s', (value) => {
    expect(parseTimesheetStatusFilter(value)).toBeUndefined();
  });
});
