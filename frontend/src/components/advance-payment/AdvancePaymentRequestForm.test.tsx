import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AdvancePaymentHistoryItem, AdvancePaymentInfo } from "@/types/api/advance-payment.types";
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

const legacyJuneHistory: AdvancePaymentHistoryItem[] = [
  {
    id: 66,
    requestAmount: 2_000_000,
    fee: 0,
    netAmount: 2_000_000,
    status: "COMPLETED",
    createdAt: "2026-06-25T18:51:24+08:00",
  },
  {
    id: 58,
    requestAmount: 2_200_000,
    fee: 0,
    netAmount: 2_200_000,
    status: "COMPLETED",
    createdAt: "2026-06-20T18:54:34+08:00",
  },
  {
    id: 52,
    requestAmount: 2_880_000,
    fee: 0,
    netAmount: 2_880_000,
    status: "COMPLETED",
    createdAt: "2026-06-06T16:00:35+08:00",
  },
  {
    id: 28,
    requestAmount: 960_000,
    fee: 0,
    netAmount: 960_000,
    status: "COMPLETED",
    createdAt: "2026-05-26T10:47:45+08:00",
  },
];

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
    expect(screen.getByText("Đang mở")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "0");

    fireEvent.click(screen.getByRole("button", { name: "50%" }));
    fireEvent.click(screen.getByRole("button", { name: "Yêu cầu ứng lương" }));

    expect(onSubmit).toHaveBeenCalledWith({ amount: 500_000, forMonth: "2026-07" });
  });

  it("replaces the request controls with the submitted-request acknowledgement", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={info}
        requestConfirmation={{
          amount: 500_000,
          forMonth: "2026-07",
          submittedAt: "2026-07-12T03:00:00.000Z",
        }}
      />
    );

    expect(screen.getByRole("status")).toHaveTextContent("Yêu cầu đã gửi");
    expect(screen.getByText(/500\.000 ₫/)).toBeInTheDocument();
    expect(screen.getByText(/Gửi ngày/)).toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Yêu cầu ứng lương" })).not.toBeInTheDocument();
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
    expect(screen.getByRole("button", { name: /Xem tài khoản nhận tiền/ })).toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Xem tài khoản nhận tiền/ }));
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
    expect(screen.getByText("Chưa mở")).toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Yêu cầu ứng lương" })).not.toBeInTheDocument();
  });

  it("uses the check-in eligibility reason instead of waiting for a timesheet", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        isSelfCheckInFlow
        info={{
          ...info,
          maxAdvanceAmount: 0,
          remainingAmount: 0,
          canRequest: false,
          canRequestTitle: "Chưa đủ hạn mức",
          canRequestReason: "Bạn chưa có tiền công được ứng còn lại. Vui lòng chấm công thêm ca làm việc.",
          quotas: [{ ...info.quotas[0], maxAdvanceAmount: 0, remainingAmount: 0 }],
        }}
      />
    );

    expect(screen.getByText("Chưa đủ hạn mức")).toBeInTheDocument();
    expect(screen.getByText("Bạn chưa có tiền công được ứng còn lại. Vui lòng chấm công thêm ca làm việc.")).toBeInTheDocument();
    expect(screen.queryByText(/Chờ bảng lương tháng/)).not.toBeInTheDocument();
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

    expect(screen.getAllByText("Đã dùng hết hạn mức").length).toBeGreaterThan(0);
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "100");
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Yêu cầu ứng lương" })).not.toBeInTheDocument();
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

  it("shows July as waiting without leaking the closed June quota from top-level totals", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        history={legacyJuneHistory}
        info={{
          ...info,
          forMonth: "2026-07",
          maxAdvanceAmount: 6_060_000,
          completedAmount: 4_200_000,
          remainingAmount: 1_860_000,
          canRequest: false,
          canRequestTitle: "Kỳ ứng lương 06/2026 kết thúc",
          canRequestReason: "Xin chờ bảng lương 07/2026 để tiếp tục",
          quotas: [
            {
              forMonth: "2026-06",
              maxAdvanceAmount: 6_060_000,
              completedAmount: 4_200_000,
              pendingAmount: 0,
              remainingAmount: 1_860_000,
            },
          ],
        }}
        viewMonth="2026-07"
      />
    );

    // Primary card reflects July — never June's closed amount as withdrawable.
    expect(screen.getByText("01/07 – 31/07/2026")).toBeInTheDocument();
    expect(screen.getByText("Chưa mở")).toBeInTheDocument();
    expect(screen.getByText("Chưa có hạn mức")).toBeInTheDocument();
    expect(screen.getByText("Chờ bảng lương tháng 07/2026")).toBeInTheDocument();
    // No title, progress, or amount from the closed June cycle may appear.
    expect(screen.queryByText("Kỳ ứng lương 06/2026 kết thúc")).not.toBeInTheDocument();
    expect(screen.queryByText("Đã dùng 8.040.000 ₫")).not.toBeInTheDocument();
    expect(screen.queryByText("Hạn mức 9.900.000 ₫")).not.toBeInTheDocument();
    expect(screen.queryByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).not.toBeInTheDocument();
    expect(screen.queryByText("Có thể ứng")).not.toBeInTheDocument();
    expect(screen.queryByText("Còn hạn mức")).not.toBeInTheDocument();
    expect(screen.queryByText("Đang mở")).not.toBeInTheDocument();
    expect(screen.queryByText("Tháng 06/2026")).not.toBeInTheDocument();
  });

  it("treats an empty past month as closed instead of waiting for attendance", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        info={info}
        viewMonth="2026-05"
        isPastMonth
      />
    );

    expect(screen.getByText("Đã đóng")).toBeInTheDocument();
    expect(screen.getByText("Kỳ ứng lương 05/2026 kết thúc")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Ứng lương/ })).toBeDisabled();
    expect(screen.queryByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).not.toBeInTheDocument();
    expect(screen.queryByText("Kỳ ứng lương này đã kết thúc")).not.toBeInTheDocument();
    expect(screen.queryByText("Đang chờ bảng công tháng 05/2026")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
  });

  it("shows June quota and used amount when the closed June month is selected", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        history={legacyJuneHistory}
        info={{
          ...info,
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
        }}
        viewMonth="2026-06"
        isPastMonth
      />
    );

    expect(screen.getByText("01/06 – 30/06/2026")).toBeInTheDocument();
    expect(screen.getByText("Kỳ ứng lương 06/2026 kết thúc")).toBeInTheDocument();
    expect(screen.getByText("Đã dùng 8.040.000 ₫")).toBeInTheDocument();
    expect(screen.getByText("Hạn mức 9.900.000 ₫")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "81");
    expect(screen.getByRole("button", { name: /Ứng lương/ })).toBeDisabled();
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
    expect(screen.getAllByText("Đã dùng hết hạn mức").length).toBeGreaterThan(0);
    expect(screen.getByRole("progressbar", { name: "Hạn mức ứng lương đã sử dụng" })).toHaveAttribute("aria-valuenow", "100");
    expect(screen.queryByLabelText("Số tiền muốn ứng")).not.toBeInTheDocument();
  });

  it("does not display previous-period disbursements for the selected month", () => {
    render(
      <AdvancePaymentRequestForm
        {...baseProps}
        history={[
          {
            id: 1,
            requestAmount: 1_860_000,
            status: "COMPLETED",
            forMonth: "2026-06",
            createdAt: "2026-06-15T03:00:00.000Z",
          } as AdvancePaymentHistoryItem,
        ]}
        info={{
          ...info,
          forMonth: "2026-07",
          maxAdvanceAmount: 0,
          remainingAmount: 0,
          canRequest: false,
          quotas: [
            {
              forMonth: "2026-06",
              maxAdvanceAmount: 1_860_000,
              completedAmount: 1_860_000,
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

    // The selected card remains July-only, even when June has completed data.
    expect(screen.getByText("Chưa mở")).toBeInTheDocument();
    expect(screen.queryByText("Đang mở")).not.toBeInTheDocument();
    expect(screen.queryByText("Tháng 06/2026")).not.toBeInTheDocument();
    expect(screen.queryByText("1.860.000 ₫")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Kỳ lương trước tháng 06/2026")).not.toBeInTheDocument();
  });
});
