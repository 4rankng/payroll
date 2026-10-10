import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { Project } from '@/types/api/project.types';
import { getDefaultWeeklyDates, useBulkTransferExportForm } from './useBulkTransferExportForm';

describe('getDefaultWeeklyDates', () => {
  it('defaults to the ky 4 custom range (22 to month end) from day 28', () => {
    const result = getDefaultWeeklyDates(new Date(2026, 9, 29));
    expect(result.from.toISOString().slice(0, 10)).toBe('2026-10-22');
    expect(result.to.toISOString().slice(0, 10)).toBe('2026-10-31');
    expect(result.selectedCustomRange).toBe('22 - 31 Tháng 10');
  });

  it('uses the 30-day month end for September', () => {
    const result = getDefaultWeeklyDates(new Date(2026, 8, 30));
    expect(result.to.toISOString().slice(0, 10)).toBe('2026-09-30');
    expect(result.selectedCustomRange).toBe('22 - 30 Tháng 09');
  });

  it('keeps the week 3 default on days 22-27', () => {
    const result = getDefaultWeeklyDates(new Date(2026, 9, 25));
    expect(result.from.toISOString().slice(0, 10)).toBe('2026-10-15');
    expect(result.to.toISOString().slice(0, 10)).toBe('2026-10-21');
    expect(result.selectedCustomRange).toBe('');
  });

  it('defaults to 01-07 of the current month in the early days', () => {
    const result = getDefaultWeeklyDates(new Date(2026, 10, 1));
    expect(result.from.toISOString().slice(0, 10)).toBe('2026-11-01');
    expect(result.to.toISOString().slice(0, 10)).toBe('2026-11-07');
    expect(result.selectedCustomRange).toBe('01 - 07 Tháng 11');
  });

  it('defaults to 01-07 through day 10', () => {
    const result = getDefaultWeeklyDates(new Date(2026, 9, 10));
    expect(result.selectedCustomRange).toBe('01 - 07 Tháng 10');
  });
});

describe('useBulkTransferExportForm', () => {
  it('builds the exact selected weekly period with the preselected project', () => {
    const projects = [
      { id: 42, code: 'DA-42', name: 'Dự án 42' },
      { id: 84, code: 'DA-84', name: 'Dự án 84' },
    ] as Project[];

    const { result } = renderHook(() => useBulkTransferExportForm({
      isOpen: true,
      onOpenChange: vi.fn(),
      projects,
      initialProjectIds: [42],
    }));

    const selectedPeriod = result.current.weekPeriods.availablePeriods[0];

    act(() => {
      result.current.applyWeekPeriod(selectedPeriod);
    });

    expect(result.current.buildExportParams()).toEqual({
      fromDate: selectedPeriod.from,
      toDate: selectedPeriod.to,
      project_ids: [42],
      employee_ids: [],
    });
  });

  it('keeps a preselected project scoped while project options are still loading', () => {
    const { result } = renderHook(() => useBulkTransferExportForm({
      isOpen: true,
      onOpenChange: vi.fn(),
      projects: [],
      initialProjectIds: [42],
    }));

    const selectedPeriod = result.current.weekPeriods.availablePeriods[0];

    act(() => {
      result.current.applyWeekPeriod(selectedPeriod);
    });

    expect(result.current.buildExportParams()).toMatchObject({
      fromDate: selectedPeriod.from,
      toDate: selectedPeriod.to,
      project_ids: [42],
    });
  });

  it('keeps explicitly selected employee IDs when the search result page changes', () => {
    const { result } = renderHook(() => useBulkTransferExportForm({
      isOpen: true,
      onOpenChange: vi.fn(),
      projects: [],
    }));

    const selectedPeriod = result.current.weekPeriods.availablePeriods[0];

    act(() => {
      result.current.applyWeekPeriod(selectedPeriod);
      result.current.setSelectedEmployees([11156]);
    });

    expect(result.current.buildExportParams()).toMatchObject({
      employee_ids: [11156],
    });
  });
});
