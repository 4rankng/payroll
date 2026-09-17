import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AuditLogDetailSheet } from './AuditLogDetailSheet';
const mocks = vi.hoisted(() => ({ mobile: false, refetch: vi.fn() }));
vi.mock('@/hooks/useBreakpoint', () => ({ useIsMobile: () => mocks.mobile }));
vi.mock('@/hooks/api/useAuditLogs', () => ({ useAuditLogDetail: () => ({ isLoading: false, isError: true, refetch: mocks.refetch }) }));
describe.each([false, true])('audit detail mobile=%s', (mobile) => {
  it('provides a retry and visible close after a failed detail read', () => {
    vi.clearAllMocks();
    mocks.mobile = mobile;
    const onClose = vi.fn();
    render(<AuditLogDetailSheet logId={1} onClose={onClose} />);
    expect(screen.getByRole('alert')).toHaveTextContent('Không thể tải dữ liệu');
    fireEvent.click(screen.getByRole('button', { name: 'Thử lại' }));
    expect(mocks.refetch).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole('button', { name: 'Đóng chi tiết nhật ký' }));
    expect(onClose).toHaveBeenCalledOnce();
  });
});
