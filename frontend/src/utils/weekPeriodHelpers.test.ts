import { describe, expect, it } from "vitest";

import { getWeeklyPayrollDefaultMonth } from "./weekPeriodHelpers";

describe("getWeeklyPayrollDefaultMonth", () => {
  it("holds the previous month for the first three days, then follows the calendar", () => {
    // Weekly batches (weeks of 1-7, 8-14, 15-21, 22-28) are approved after the
    // fact, so on the 1st-3rd the unapproved batch is still the previous month's
    // 22-28 week. From the 4th the current month is the live one.
    expect(getWeeklyPayrollDefaultMonth(new Date(2026, 9, 1))).toBe("2026-09");
    expect(getWeeklyPayrollDefaultMonth(new Date(2026, 9, 3, 23, 59))).toBe("2026-09");
    expect(getWeeklyPayrollDefaultMonth(new Date(2026, 9, 4, 0, 1))).toBe("2026-10");
    expect(getWeeklyPayrollDefaultMonth(new Date(2026, 9, 20))).toBe("2026-10");
  });

  it("does not borrow the FlexPay cycle", () => {
    // The advance period rolls on the 20th. If this ever returned "2026-09" on
    // 2026-10-10 again, the weekly screen had been wired to the wrong calendar.
    expect(getWeeklyPayrollDefaultMonth(new Date(2026, 9, 10))).toBe("2026-10");
    expect(getWeeklyPayrollDefaultMonth(new Date(2026, 9, 19))).toBe("2026-10");
  });

  it("rolls the year over in January", () => {
    expect(getWeeklyPayrollDefaultMonth(new Date(2027, 0, 2))).toBe("2026-12");
    expect(getWeeklyPayrollDefaultMonth(new Date(2027, 0, 4))).toBe("2027-01");
  });
});
