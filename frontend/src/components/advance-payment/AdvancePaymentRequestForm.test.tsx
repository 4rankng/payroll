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
  bankAccountNumber: "0366178061",
  bankName: "Quân đội (MB)",
  bankAccountName: "Bùi Nguyễn Duy Anh",
  onSubmit: vi.fn(),
  onAmountChange: vi.fn(),
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
    expect(screen.getByText(/\*\*\*\* \*\*\*\* 8061/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "50%" }));
    fireEvent.click(screen.getByRole("button", { name: "Tiếp tục" }));

    expect(onSubmit).toHaveBeenCalledWith({ amount: 500_000, forMonth: "2026-07" });
  });

  it("blocks the form when receiving bank details are missing", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={info}
        bankAccountNumber={undefined}
        bankName={undefined}
      />
    );

    expect(screen.getByText("Chưa có tài khoản nhận tiền")).toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
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
});
