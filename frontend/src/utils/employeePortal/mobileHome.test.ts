import { describe, expect, it } from "vitest";
import {
  createFlexibleEmployeeHomeModel,
  createRegularEmployeeHomeModel,
} from "./mobileHome";
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
        forMonth: "2026-07",
        maxAdvanceAmount: 0,
        completedAmount: 0,
        pendingAmount: 0,
        remainingAmount: 0,
        canRequest: false,
        canRequestTitle: "Kỳ ứng lương 06/2026 kết thúc",
        canRequestReason: "Xin chờ bảng lương 07/2026 để tiếp tục",
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
      },
      history: [],
      hasCheckIn: false,
      hasBankInfo: true,
    });

    expect(model.amountLabel).toBe("Hạn mức còn lại");
    expect(model.amount).toContain("3.000.000");
    expect(model.amountDescription).toBe("Xin chờ bảng lương 07/2026 để tiếp tục");
    expect(model.periodLabel).toBeUndefined();
    expect(model.metricsTitle).toBe("Ứng lương 06/2026");
    expect(model.metrics).toEqual([
      expect.objectContaining({ label: "Đã nhận", value: expect.stringContaining("9.360.000") }),
      expect.objectContaining({ label: "Đang chờ", value: expect.stringContaining("0") }),
      expect.objectContaining({ label: "Hạn mức", value: expect.stringContaining("12.360.000") }),
    ]);
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
    expect(model.metricsTitle).toBeUndefined();
    expect(model.quickActions[0]).toMatchObject({
      label: "Ứng lương",
      icon: "advance",
      targetId: "employee-advance-request",
    });
    expect(model.quickActions).not.toContainEqual(
      expect.objectContaining({ targetId: "employee-limit" })
    );
  });

  it("labels history-only blocked data with the ended payroll month", () => {
    const model = createFlexibleEmployeeHomeModel({
      info: {
        ...baseInfo,
        forMonth: "2026-07",
        maxAdvanceAmount: 0,
        completedAmount: 0,
        pendingAmount: 0,
        remainingAmount: 0,
        canRequest: false,
        canRequestTitle: "Kỳ ứng lương 06/2026 kết thúc",
        canRequestReason: "Xin chờ bảng lương 07/2026 để tiếp tục",
        quotas: [],
      },
      history: [
        {
          id: 1,
          requestAmount: 9_360_000,
          netAmount: 9_172_800,
          fee: 187_200,
          status: "COMPLETED",
          createdAt: "2026-07-08T00:00:00Z",
        },
      ],
      hasCheckIn: false,
      hasBankInfo: true,
    });

    expect(model.amountDescription).toBe("Xin chờ bảng lương 07/2026 để tiếp tục");
    expect(model.metricsTitle).toBe("Ứng lương 06/2026");
    expect(model.metrics).toEqual([
      expect.objectContaining({ label: "Đã nhận", value: expect.stringContaining("9.360.000") }),
      expect.objectContaining({ label: "Đang chờ", value: expect.stringContaining("0") }),
      expect.objectContaining({ label: "Hạn mức", value: expect.stringContaining("9.360.000") }),
    ]);
  });
});

describe("createRegularEmployeeHomeModel", () => {
  const baseInput = {
    monthLabel: "Tháng 07/2026",
    monthlyTotalSalary: 2_772_000,
    monthlyTotalHours: 61,
    totalPayable: 2_772_000,
    totalPaid: 2_772_000,
    workDayCount: 6,
    totalRecords: 10,
    hasBankInfo: true,
    unreadCount: 1,
  };

  it("shows a fully received salary amount only once", () => {
    const model = createRegularEmployeeHomeModel(baseInput);

    expect(model.amountDescriptionLabel).toBe("Ngày làm việc");
    expect(model.amountDescription).toBe("6 ngày");
    expect(model.metrics).toEqual([
      expect.objectContaining({ label: "Tổng công", value: "61 giờ" }),
    ]);
  });

  it("keeps monetary metrics when their values add distinct information", () => {
    const model = createRegularEmployeeHomeModel({
      ...baseInput,
      monthlyTotalSalary: 3_000_000,
      totalPayable: 2_700_000,
      totalPaid: 1_200_000,
    });

    expect(model.metrics.map((metric) => metric.label)).toEqual([
      "Tổng công",
      "Có thể trả",
      "Đã nhận",
    ]);
  });

  it("does not repeat equal payable and received amounts", () => {
    const model = createRegularEmployeeHomeModel({
      ...baseInput,
      monthlyTotalSalary: 3_000_000,
      totalPayable: 1_200_000,
      totalPaid: 1_200_000,
    });

    expect(model.metrics.map((metric) => metric.label)).toEqual([
      "Tổng công",
      "Đã nhận",
    ]);
  });
});
