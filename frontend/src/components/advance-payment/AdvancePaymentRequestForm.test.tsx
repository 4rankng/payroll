import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AdvancePaymentInfo } from "@/types/api/advance-payment.types";
import { AdvancePaymentRequestForm } from "./AdvancePaymentRequestForm";

const info: AdvancePaymentInfo = {
  forMonth: "2026-07",
  maxAdvanceAmount: 1_000_000,
  completedAmount: 0,
  pendingAmount: 0,
  remainingAmount: 1_000_000,
  canRequest: true,
  feePercentage: 2,
  minFee: 10_000,
  hasFlexible: true,
  quotas: [
    {
      forMonth: "2026-07",
      maxAdvanceAmount: 1_000_000,
      completedAmount: 0,
      pendingAmount: 0,
      remainingAmount: 1_000_000,
    },
  ],
};

const baseProps = {
  history: [],
  feeDetails: { fee: 10_000, netAmount: 490_000 },
  hasBankDestination: true,
  onSubmit: vi.fn(),
  onAmountChange: vi.fn(),
  onBankAction: vi.fn(),
  isPending: false,
};

describe("AdvancePaymentRequestForm", () => {
  it("shows the compact quick choices and submits a valid selected amount", () => {
    const onSubmit = vi.fn();
    render(<AdvancePaymentRequestForm {...baseProps} info={info} onSubmit={onSubmit} />);

    expect(screen.getByRole("button", { name: "25%" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "50%" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tối đa" })).toBeInTheDocument();
    expect(screen.queryByRole("slider")).not.toBeInTheDocument();
    expect(screen.queryByText("Bùi Nguyễn Duy Anh")).not.toBeInTheDocument();
    expect(screen.queryByText("0366178061")).not.toBeInTheDocument();
    expect(screen.getByText("01/07 – 31/07/2026")).toBeInTheDocument();
    expect(screen.getByText("Còn hạn mức")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "0");

    fireEvent.click(screen.getByRole("button", { name: "50%" }));
    fireEvent.click(screen.getByRole("button", { name: "Yêu cầu ứng lương" }));

    expect(onSubmit).toHaveBeenCalledWith({ amount: 500_000, forMonth: "2026-07" });
  });

  it("blocks the form when receiving bank details are missing", () => {
    const onBankAction = vi.fn();
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={info}
        hasBankDestination={false}
        onBankAction={onBankAction}
      />
    );

    expect(screen.getByText("Chưa có tài khoản nhận tiền")).toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Xem tài khoản nhận tiền" }));
    expect(onBankAction).toHaveBeenCalledOnce();
  });

  it("shows the server reason when requests are unavailable", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          canRequest: false,
          canRequestTitle: "Ngoài kỳ ứng lương",
          canRequestReason: "Kỳ tiếp theo mở vào ngày 15.",
        }}
      />
    );

    expect(screen.getByText("Ngoài kỳ ứng lương")).toBeInTheDocument();
    expect(screen.getByText("Kỳ tiếp theo mở vào ngày 15.")).toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Xem lịch sử yêu cầu" })).not.toBeInTheDocument();
  });

  it("deduplicates quick choices after the minimum amount clamp", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          maxAdvanceAmount: 10_000,
          remainingAmount: 10_000,
          quotas: [
            {
              forMonth: "2026-07",
              maxAdvanceAmount: 10_000,
              completedAmount: 0,
              pendingAmount: 0,
              remainingAmount: 10_000,
            },
          ],
        }}
      />
    );

    expect(screen.queryByRole("button", { name: "25%" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "50%" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tối đa" })).toBeInTheDocument();
  });

  it("explains a zero remaining quota instead of showing an unusable form", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          remainingAmount: 0,
          quotas: [{ ...info.quotas[0], remainingAmount: 0 }],
        }}
      />
    );

    expect(screen.getByText("Hạn mức kỳ này đã sử dụng hết")).toBeInTheDocument();
    expect(screen.getByText("Đã dùng hết hạn mức")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "100");
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Xem lịch sử yêu cầu" })).not.toBeInTheDocument();
  });

  it("shows a proportional allowance summary for partially used quota", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          maxAdvanceAmount: 1_000_000,
          completedAmount: 250_000,
          remainingAmount: 750_000,
          quotas: [{
            ...info.quotas[0],
            maxAdvanceAmount: 1_000_000,
            completedAmount: 250_000,
            remainingAmount: 750_000,
          }],
        }}
      />
    );

    expect(screen.getByText("Đã dùng 250.000 ₫")).toBeInTheDocument();
    expect(screen.getByText("Hạn mức 1.000.000 ₫")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "25");
  });

  it("keeps a large available amount readable without changing its value", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          maxAdvanceAmount: 999_999_999,
          remainingAmount: 999_999_999,
          quotas: [{ ...info.quotas[0], maxAdvanceAmount: 999_999_999, remainingAmount: 999_999_999 }],
        }}
      />
    );

    expect(screen.getByText("999.999.999 ₫")).toBeInTheDocument();
  });

  it("shows the viewed July payroll month as waiting instead of falling back to exhausted June", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          forMonth: "2026-06",
          maxAdvanceAmount: 1_000_000,
          completedAmount: 1_000_000,
          remainingAmount: 0,
          canRequest: false,
          quotas: [
            {
              forMonth: "2026-06",
              maxAdvanceAmount: 1_000_000,
              completedAmount: 1_000_000,
              pendingAmount: 0,
              remainingAmount: 0,
            },
            {
              forMonth: "2026-07",
              maxAdvanceAmount: 0,
              completedAmount: 0,
              pendingAmount: 0,
              remainingAmount: 0,
            },
          ],
        }}
        viewMonth="2026-07"
      />
    );

    expect(screen.getByText("01/07 – 31/07/2026")).toBeInTheDocument();
    expect(screen.getByText("Đang chờ bảng công tháng 07/2026")).toBeInTheDocument();
    expect(screen.queryByText("Hạn mức kỳ này đã sử dụng hết")).not.toBeInTheDocument();
  });

  it("keeps exhausted June messaging when June is the viewed month", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={{
          ...info,
          forMonth: "2026-06",
          maxAdvanceAmount: 1_000_000,
          completedAmount: 1_000_000,
          remainingAmount: 0,
          canRequest: false,
          quotas: [{
            forMonth: "2026-06",
            maxAdvanceAmount: 1_000_000,
            completedAmount: 1_000_000,
            pendingAmount: 0,
            remainingAmount: 0,
          }],
        }}
        viewMonth="2026-06"
      />
    );

    expect(screen.getByText("01/06 – 30/06/2026")).toBeInTheDocument();
    expect(screen.getByText("Hạn mức kỳ này đã sử dụng hết")).toBeInTheDocument();
  });
});
