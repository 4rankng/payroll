import { describe, expect, it } from "vitest";

import {
  getAvailableWeekPeriods,
  getCustomDateRanges,
  getWeeklyPayrollDefaultMonth,
} from "./weekPeriodHelpers";

describe("getWeeklyPayrollDefaultMonth", () => {
  it("holds the previous month for the first three days, then follows the calendar", () => {
    // Weekly batches (weeks of 1-7, 8-14, 15-21, 22-month end) are approved
    // after the fact, so on the 1st-3rd the unapproved batch is still the
    // previous month's final week. From the 4th the current month is the live one.
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

describe("getAvailableWeekPeriods", () => {
  it("extends week 4 to the end of the month (October has 31 days)", () => {
    for (const day of [22, 28, 31]) {
      const { availablePeriods, defaultPeriod } = getAvailableWeekPeriods(new Date(2026, 9, day));
      expect(availablePeriods).toHaveLength(2);
      expect(availablePeriods[0].week).toBe(3);
      expect(availablePeriods[0].from).toBe("2026-10-15");
      expect(availablePeriods[0].to).toBe("2026-10-21");
      expect(availablePeriods[1].week).toBe(4);
      expect(availablePeriods[1].from).toBe("2026-10-22");
      expect(availablePeriods[1].to).toBe("2026-10-31");
      // The in-progress week 4 is selectable but never the default.
      expect(defaultPeriod.week).toBe(3);
    }
  });

  it("ends week 4 on day 30 in a 30-day month", () => {
    const { availablePeriods } = getAvailableWeekPeriods(new Date(2026, 8, 29));
    expect(availablePeriods[1].week).toBe(4);
    expect(availablePeriods[1].to).toBe("2026-09-30");
  });

  it("keeps week 4 at day 28 in a 28-day February", () => {
    const { availablePeriods } = getAvailableWeekPeriods(new Date(2026, 1, 28));
    expect(availablePeriods[1].week).toBe(4);
    expect(availablePeriods[1].to).toBe("2026-02-28");
  });

  it("extends week 4 to day 29 in a leap February", () => {
    const { availablePeriods } = getAvailableWeekPeriods(new Date(2028, 1, 28));
    expect(availablePeriods[1].week).toBe(4);
    expect(availablePeriods[1].to).toBe("2028-02-29");
  });

  it("offers the previous month's week 4 (through month end) in the early days", () => {
    const { availablePeriods, defaultPeriod } = getAvailableWeekPeriods(new Date(2026, 10, 1));
    expect(availablePeriods[0].month).toBe("previous");
    expect(availablePeriods[0].from).toBe("2026-10-22");
    expect(availablePeriods[0].to).toBe("2026-10-31");
    expect(availablePeriods[1].from).toBe("2026-11-01");
    expect(availablePeriods[1].to).toBe("2026-11-07");
    expect(defaultPeriod.week).toBe(1);
  });
});

describe("getCustomDateRanges", () => {
  it("labels the late-month week-4 range with the month's last day (October)", () => {
    const ranges = getCustomDateRanges(new Date(2026, 9, 29));
    expect(ranges).toHaveLength(2);
    expect(ranges[0]).toMatchObject({ from: "2026-10-15", to: "2026-10-21", label: "15 - 21 Tháng 10" });
    expect(ranges[1]).toMatchObject({ from: "2026-10-22", to: "2026-10-31", label: "22 - 31 Tháng 10" });
  });

  it("uses day 30 for a 30-day month", () => {
    const ranges = getCustomDateRanges(new Date(2026, 8, 30));
    expect(ranges[1]).toMatchObject({ from: "2026-09-22", to: "2026-09-30", label: "22 - 30 Tháng 09" });
  });

  it("uses day 29 for a leap February and day 28 for a common one", () => {
    expect(getCustomDateRanges(new Date(2028, 1, 28))[1].label).toBe("22 - 29 Tháng 02");
    expect(getCustomDateRanges(new Date(2026, 1, 28))[1].label).toBe("22 - 28 Tháng 02");
  });

  it("offers the previous month's week 4 alongside 01-07 in the early days", () => {
    const ranges = getCustomDateRanges(new Date(2026, 10, 5));
    expect(ranges[0]).toMatchObject({ from: "2026-10-22", to: "2026-10-31", label: "22 - 31 Tháng 10" });
    expect(ranges[1]).toMatchObject({ from: "2026-11-01", to: "2026-11-07", label: "01 - 07 Tháng 11" });
  });
});
