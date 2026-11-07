import { fireEvent, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import { TimesheetPageHeaderMobile } from './TimesheetPageHeaderMobile';

// The header wires its console kicker via useLocation — render inside a router.
describe('TimesheetPageHeaderMobile', () => {
  it('exposes the production OnePay export in the Admin mobile actions', () => {
    const onOnePayExport = vi.fn();

    render(
      <MemoryRouter>
        <TimesheetPageHeaderMobile
          onAddTimesheet={vi.fn()}
          onApprovedTimesheetsExport={vi.fn()}
          onOnePayExport={onOnePayExport}
          monthValue="2026-07"
          onMonthChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Thêm tùy chọn' }));
    fireEvent.click(screen.getByRole('button', { name: 'Chuyển OnePay' }));

    expect(onOnePayExport).toHaveBeenCalledOnce();
  });

  it('exposes the reject-unpaid action for Admin and closes the action sheet', () => {
    const onRejectUnpaid = vi.fn();

    render(
      <MemoryRouter>
        <TimesheetPageHeaderMobile
          onAddTimesheet={vi.fn()}
          onApprovedTimesheetsExport={vi.fn()}
          onRejectUnpaid={onRejectUnpaid}
          monthValue="2026-07"
          onMonthChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Thêm tùy chọn' }));
    fireEvent.click(screen.getByRole('button', { name: 'Loại công' }));

    expect(onRejectUnpaid).toHaveBeenCalledOnce();
    expect(screen.queryByRole('button', { name: 'Loại công' })).not.toBeInTheDocument();
  });

  it('does not expose the Admin reject-unpaid action to Partner', () => {
    render(
      <MemoryRouter>
        <TimesheetPageHeaderMobile
          onAddTimesheet={vi.fn()}
          onApprovedTimesheetsExport={vi.fn()}
          onRejectUnpaid={vi.fn()}
          userRole="partner"
          monthValue="2026-07"
          onMonthChange={vi.fn()}
        />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Thêm tùy chọn' }));

    expect(screen.queryByRole('button', { name: 'Loại công' })).not.toBeInTheDocument();
  });
});
