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
});
