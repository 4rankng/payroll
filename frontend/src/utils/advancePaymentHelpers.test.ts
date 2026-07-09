import { describe, expect, it } from "vitest";
import { getAdvanceQuotaSummary } from "./advancePaymentHelpers";
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
});
