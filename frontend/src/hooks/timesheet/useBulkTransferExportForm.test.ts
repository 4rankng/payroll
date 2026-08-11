import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { Project } from '@/types/api/project.types';
import { useBulkTransferExportForm } from './useBulkTransferExportForm';

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
