import { describe, expect, it } from "vitest";
import {
  formatPayrollMonthRange,
  formatMonthShort,
  getAdvanceQuotaSummary,
  getAdvanceQuotaSummaryForMonth,
  getDefaultAdvanceMonth,
} from "./advancePaymentHelpers";
import type {
  AdvancePaymentHistoryItem,
  AdvancePaymentInfo,
} from "@/types/api/advance-payment.types";

const baseInfo: AdvancePaymentInfo = {
  forMonth: "2026-06",
  maxAdvanceAmount: 12_360_000,
  completedAmount: 0,
  pendingAmount: 0,
  remainingAmount: 3_000_000,
  canRequest: true,
  feePercentage: 2,
  minFee: 10_000,
  hasFlexible: true,
  quotas: [
    {
      forMonth: "2026-06",
      maxAdvanceAmount: 12_360_000,
      completedAmount: 0,
      pendingAmount: 0,
      remainingAmount: 3_000_000,
    },
  ],
};

const completedHistory: AdvancePaymentHistoryItem[] = [
  {
    id: 4,
    requestAmount: 1_500_000,
    netAmount: 1_470_000,
    fee: 30_000,
    status: "COMPLETED",
    forMonth: "2026-06",
    createdAt: "2026-07-08T00:00:00Z",
  },
  {
    id: 3,
    requestAmount: 860_000,
    netAmount: 842_800,
    fee: 17_200,
    status: "COMPLETED",
    forMonth: "2026-06",
    createdAt: "2026-06-29T00:00:00Z",
  },
  {
    id: 2,
    requestAmount: 3_000_000,
    netAmount: 2_940_000,
    fee: 60_000,
    status: "COMPLETED",
    forMonth: "2026-06",
    createdAt: "2026-06-26T00:00:00Z",
  },
  {
    id: 1,
    requestAmount: 4_000_000,
    netAmount: 3_948_000,
    fee: 52_000,
    status: "COMPLETED",
    forMonth: "2026-06",
    createdAt: "2026-06-25T00:00:00Z",
  },
];

describe("getAdvanceQuotaSummary", () => {
  it("uses completed request history when quota usage is stale at zero", () => {
    const summary = getAdvanceQuotaSummary(baseInfo, completedHistory);

    expect(summary.forMonth).toBe("2026-06");
    expect(summary.usedAmount).toBe(9_360_000);
    expect(summary.remainingAmount).toBe(3_000_000);
    expect(summary.usedPercentage).toBe(76);
  });

  it("prefers an actual quota month over a stale top-level month", () => {
    const summary = getAdvanceQuotaSummary({
      ...baseInfo,
      forMonth: "2026-07",
      quotas: [
        {
          forMonth: "2026-06",
          maxAdvanceAmount: 12_360_000,
          completedAmount: 9_360_000,
          pendingAmount: 0,
          remainingAmount: 3_000_000,
        },
      ],
    });

    expect(summary.forMonth).toBe("2026-06");
    expect(summary.usedPercentage).toBe(76);
  });

  it("skips a zero current-month quota while waiting for the next payroll upload", () => {
    const summary = getAdvanceQuotaSummary({
      ...baseInfo,
      forMonth: "2026-07",
      maxAdvanceAmount: 0,
      completedAmount: 0,
      pendingAmount: 0,
      remainingAmount: 0,
      canRequest: false,
      quotas: [
        {
          forMonth: "2026-06",
          maxAdvanceAmount: 12_360_000,
          completedAmount: 9_360_000,
          pendingAmount: 0,
          remainingAmount: 3_000_000,
        },
        {
          forMonth: "2026-07",
          maxAdvanceAmount: 0,
          completedAmount: 0,
          pendingAmount: 0,
          remainingAmount: 0,
        },
      ],
    });

    expect(summary.forMonth).toBe("2026-06");
    expect(summary.maxAdvanceAmount).toBe(12_360_000);
    expect(summary.completedAmount).toBe(9_360_000);
    expect(summary.remainingAmount).toBe(3_000_000);
  });

  it("uses history as the minimum known limit when quota rows are missing", () => {
    const summary = getAdvanceQuotaSummary(
      {
        ...baseInfo,
        forMonth: "2026-07",
        maxAdvanceAmount: 0,
        completedAmount: 0,
        pendingAmount: 0,
        remainingAmount: 0,
        canRequest: false,
        quotas: [],
      },
      completedHistory.map(({ forMonth: _forMonth, ...item }) => item)
    );

    expect(summary.completedAmount).toBe(9_360_000);
    expect(summary.maxAdvanceAmount).toBe(9_360_000);
    expect(summary.remainingAmount).toBe(0);
  });
});

describe("getAdvanceQuotaSummaryForMonth", () => {
  it("does not reuse top-level June totals when July has no quota row", () => {
    const summary = getAdvanceQuotaSummaryForMonth(
      {
        ...baseInfo,
        forMonth: "2026-07",
        maxAdvanceAmount: 9_900_000,
        completedAmount: 8_040_000,
        remainingAmount: 1_860_000,
        canRequest: false,
        quotas: [
          {
            forMonth: "2026-06",
            maxAdvanceAmount: 9_900_000,
            completedAmount: 8_040_000,
            pendingAmount: 0,
            remainingAmount: 1_860_000,
          },
        ],
      },
      "2026-07",
    );

    expect(summary).toEqual({
      forMonth: "2026-07",
      maxAdvanceAmount: 0,
      completedAmount: 0,
      pendingAmount: 0,
      remainingAmount: 0,
      usedAmount: 0,
      usedPercentage: 0,
    });
  });

  it("attributes legacy untagged history only to the resolved June quota", () => {
    const info: AdvancePaymentInfo = {
      ...baseInfo,
      forMonth: "2026-07",
      maxAdvanceAmount: 6_060_000,
      completedAmount: 4_200_000,
      remainingAmount: 1_860_000,
      canRequest: false,
      quotas: [
        {
          forMonth: "2026-06",
          maxAdvanceAmount: 6_060_000,
          completedAmount: 4_200_000,
          pendingAmount: 0,
          remainingAmount: 1_860_000,
        },
      ],
    };
    const history = completedHistory.map(({ forMonth: _forMonth, ...item }) => item);

    const june = getAdvanceQuotaSummaryForMonth(info, "2026-06", history);
    const july = getAdvanceQuotaSummaryForMonth(info, "2026-07", history);

    expect(june.usedAmount).toBe(9_360_000);
    expect(june.maxAdvanceAmount).toBe(11_220_000);
    expect(july.usedAmount).toBe(0);
    expect(july.maxAdvanceAmount).toBe(0);
  });
});

describe("formatPayrollMonthRange", () => {
  it("renders a payroll month as its full calendar-month range", () => {
    expect(formatPayrollMonthRange("2026-07")).toBe("01/07 – 31/07/2026");
  });

  it("uses the correct final day for leap-year February", () => {
    expect(formatPayrollMonthRange("2024-02")).toBe("01/02 – 29/02/2024");
  });

  it("preserves invalid values instead of inventing a period", () => {
    expect(formatPayrollMonthRange("invalid")).toBe("invalid");
  });
});

describe("formatMonthShort", () => {
  it("renders YYYY-MM as MM/YYYY", () => {
    expect(formatMonthShort("2026-07")).toBe("07/2026");
  });
});

describe("getDefaultAdvanceMonth", () => {
  it("returns the previous month before the request window opens (day < 10)", () => {
    expect(getDefaultAdvanceMonth(new Date(2026, 6, 9, 12))).toBe("2026-06");
    expect(getDefaultAdvanceMonth(new Date(2026, 6, 1, 12))).toBe("2026-06");
  });

  it("returns the current month once the request window opens (day >= 10)", () => {
    // Day 10 is the first day a self-check-in advance request lands in the
    // current month — the admin list must open here, not on the previous month.
    expect(getDefaultAdvanceMonth(new Date(2026, 6, 10, 12))).toBe("2026-07");
    expect(getDefaultAdvanceMonth(new Date(2026, 6, 20, 12))).toBe("2026-07");
    expect(getDefaultAdvanceMonth(new Date(2026, 6, 31, 12))).toBe("2026-07");
  });

  it("crosses the year boundary when the previous month is December", () => {
    expect(getDefaultAdvanceMonth(new Date(2026, 0, 9, 12))).toBe("2025-12");
    expect(getDefaultAdvanceMonth(new Date(2026, 0, 10, 12))).toBe("2026-01");
  });

  it("returns a YYYY-MM string when called with no argument", () => {
    expect(getDefaultAdvanceMonth()).toMatch(/^\d{4}-\d{2}$/);
  });
});
