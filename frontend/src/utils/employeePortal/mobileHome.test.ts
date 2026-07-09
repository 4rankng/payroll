import { describe, expect, it } from "vitest";
import { createFlexibleEmployeeHomeModel } from "./mobileHome";
import type { AdvancePaymentInfo } from "@/types/api/advance-payment.types";

const baseInfo: AdvancePaymentInfo = {
  forMonth: "2026-06",
  maxAdvanceAmount: 12_360_000,
  completedAmount: 9_360_000,
  pendingAmount: 0,
  remainingAmount: 3_000_000,
  canRequest: true,
  feePercentage: 2,
  minFee: 10_000,
  hasFlexible: true,
  quotas: [],
};

describe("createFlexibleEmployeeHomeModel", () => {
  it("shows quota-only copy when the backend says requests are closed", () => {
    const model = createFlexibleEmployeeHomeModel({
      info: {
        ...baseInfo,
        canRequest: false,
        canRequestTitle: "Kỳ ứng lương 06/2026 kết thúc",
        canRequestReason: "Xin chờ bảng lương 07/2026 để tiếp tục",
      },
      history: [],
      hasCheckIn: false,
      hasBankInfo: true,
    });

    expect(model.amountLabel).toBe("Hạn mức còn lại");
    expect(model.amountDescription).toBe("Xin chờ bảng lương 07/2026 để tiếp tục");
    expect(model.periodLabel).toBeUndefined();
    expect(model.quickActions).toEqual([
      {
        id: "history",
        label: "Lịch sử",
        icon: "history",
        targetId: "employee-history",
      },
      {
        id: "account",
        label: "Tài khoản",
        icon: "account",
        targetId: "employee-bank",
      },
    ]);
  });

  it("keeps request copy when requests are open", () => {
    const model = createFlexibleEmployeeHomeModel({
      info: baseInfo,
      history: [],
      hasCheckIn: false,
      hasBankInfo: true,
    });

    expect(model.amountLabel).toBe("Có thể ứng");
    expect(model.amountDescription).toBe("Sẵn sàng gửi yêu cầu ứng lương.");
    expect(model.periodLabel).toBeUndefined();
    expect(model.quickActions[0]).toMatchObject({
      label: "Ứng lương",
      icon: "advance",
      targetId: "employee-advance-request",
    });
    expect(model.quickActions).not.toContainEqual(
      expect.objectContaining({ targetId: "employee-limit" })
    );
  });
});
