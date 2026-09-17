import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { HealthDrilldownSheet } from './HealthDrilldownSheet';
import type { HealthDrilldownTarget } from './CheckInHealthStrip';

const mocks = vi.hoisted(() => ({ mobile: false, query: {} as Record<string, unknown>, refetch: vi.fn() }));
vi.mock('@/hooks/useBreakpoint', () => ({ useIsMobile: () => mocks.mobile }));
vi.mock('@/hooks/api/useDashboard', () => ({ useFailedAttempts: () => mocks.query, useQuotaAnomalies: () => mocks.query }));
vi.mock('@/hooks/api/useAdminAttendance', () => ({ useAdminAttendances: () => mocks.query }));
vi.mock('./FailedAttemptLocationMap', () => ({ FailedAttemptLocationMap: () => null, formatGpsAccuracy: () => '' }));
vi.mock('./AttendanceLocationMap', () => ({ AttendanceLocationMap: () => null }));

const targets: HealthDrilldownTarget[] = [
  { type: 'failed-attempts' },
  { type: 'quota-anomaly', anomalyType: 'drift' },
  { type: 'attendance-list', label: 'Đang chấm công', status: 'checked_in', emptyLabel: 'Không có ca đang chấm công' },
  { type: 'successful-checkouts' },
];
beforeEach(() => {
  vi.clearAllMocks();
  mocks.query = { isLoading: false, isFetching: false, isError: true, data: undefined, refetch: mocks.refetch };
});
describe.each([false, true])('health drilldown mobile=%s', (mobile) => {
  it.each(targets)('shows failure and retries instead of reporting no records for $type', (target) => {
    mocks.mobile = mobile;
    render(<HealthDrilldownSheet target={target} month="2026-09" onClose={vi.fn()} />);
    expect(screen.getByRole('alert')).toHaveTextContent('Không thể tải dữ liệu');
    expect(screen.queryByText(/^Không có (lần|bất|ca|ai)/)).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Thử lại' }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
  });
  it('can close a failed drilldown', () => {
    mocks.mobile = mobile;
    const onClose = vi.fn();
    render(<HealthDrilldownSheet target={targets[0]} onClose={onClose} />);
    fireEvent.click(screen.getByRole('button', { name: 'Đóng' }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
