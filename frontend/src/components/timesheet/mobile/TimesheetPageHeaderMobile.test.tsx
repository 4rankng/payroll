import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import { TimesheetPageHeaderMobile } from './TimesheetPageHeaderMobile';

describe('TimesheetPageHeaderMobile', () => {
  it('exposes the production OnePay export in the Admin mobile actions', () => {
    const onOnePayExport = vi.fn();

    render(
      <TimesheetPageHeaderMobile
        onAddTimesheet={vi.fn()}
        onApprovedTimesheetsExport={vi.fn()}
        onOnePayExport={onOnePayExport}
        monthValue="2026-07"
        onMonthChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Thêm tùy chọn' }));
    fireEvent.click(screen.getByRole('button', { name: 'Chuyển OnePay' }));

    expect(onOnePayExport).toHaveBeenCalledOnce();
  });

  it('exposes the reject-unpaid action for Admin and closes the action sheet', () => {
    const onRejectUnpaid = vi.fn();

    render(
      <TimesheetPageHeaderMobile
        onAddTimesheet={vi.fn()}
        onApprovedTimesheetsExport={vi.fn()}
        onRejectUnpaid={onRejectUnpaid}
        monthValue="2026-07"
        onMonthChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Thêm tùy chọn' }));
    fireEvent.click(screen.getByRole('button', { name: 'Loại công' }));

    expect(onRejectUnpaid).toHaveBeenCalledOnce();
    expect(screen.queryByRole('button', { name: 'Loại công' })).not.toBeInTheDocument();
  });

  it('does not expose the Admin reject-unpaid action to Partner', () => {
    render(
      <TimesheetPageHeaderMobile
        onAddTimesheet={vi.fn()}
        onApprovedTimesheetsExport={vi.fn()}
        onRejectUnpaid={vi.fn()}
        userRole="partner"
        monthValue="2026-07"
        onMonthChange={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Thêm tùy chọn' }));

    expect(screen.queryByRole('button', { name: 'Loại công' })).not.toBeInTheDocument();
  });
});
