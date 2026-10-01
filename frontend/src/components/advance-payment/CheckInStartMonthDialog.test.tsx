import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { CheckInStartMonthDialog, getCheckInStartDate } from "./CheckInStartMonthDialog";

// The day is never the admin's to choose: it is always the 1st of the month
// they picked, and the dialog must say so before the request is sent.
describe("getCheckInStartDate", () => {
  const now = new Date(2026, 9, 20, 15, 30); // 20/10/2026 local

  it("resolves both choices to the first of the month", () => {
    expect(getCheckInStartDate("this_month", now).getDate()).toBe(1);
    expect(getCheckInStartDate("next_month", now).getDate()).toBe(1);
    expect(getCheckInStartDate("this_month", now).getMonth()).toBe(9);
    expect(getCheckInStartDate("next_month", now).getMonth()).toBe(10);
  });

  it("rolls December's next month into January of the next year", () => {
    const december = new Date(2026, 11, 31);
    const start = getCheckInStartDate("next_month", december);
    expect(start.getFullYear()).toBe(2027);
    expect(start.getMonth()).toBe(0);
    expect(start.getDate()).toBe(1);
  });
});

describe("CheckInStartMonthDialog", () => {
  function renderDialog(overrides: Partial<React.ComponentProps<typeof CheckInStartMonthDialog>> = {}) {
    const onConfirm = vi.fn();
    render(
      <CheckInStartMonthDialog
        open
        onOpenChange={vi.fn()}
        employeeName="Nguyễn Hoàng An"
        onConfirm={onConfirm}
        {...overrides}
      />,
    );
    return onConfirm;
  }

  it("confirms the chosen month and never fires on open", () => {
    const onConfirm = renderDialog();

    expect(onConfirm).not.toHaveBeenCalled();
    fireEvent.click(screen.getByLabelText(/Tháng này/));
    fireEvent.click(screen.getByRole("button", { name: /^Xác nhận từ/ }));

    expect(onConfirm).toHaveBeenCalledWith("this_month");
  });

  it("offers the reschedule wording and the current month for a queued enable", () => {
    const onConfirm = renderDialog({ isReschedule: true });

    fireEvent.click(screen.getByLabelText(/Tháng này/));
    fireEvent.click(screen.getByRole("button", { name: "Xác nhận đổi tháng" }));

    expect(onConfirm).toHaveBeenCalledWith("this_month");
  });
});
