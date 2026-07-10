import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AdvancePaymentHistoryItem } from "@/types/api/advance-payment.types";
import { AdvancePaymentHistoryCard } from "./AdvancePaymentHistoryCard";

const pendingRequest: AdvancePaymentHistoryItem = {
  id: 42,
  requestAmount: 500_000,
  fee: 10_000,
  netAmount: 490_000,
  status: "PENDING",
  forMonth: "2026-07",
  createdAt: "2026-07-10T08:00:00+07:00",
};

describe("AdvancePaymentHistoryCard", () => {
  it("starts each transaction collapsed and keeps cancellation keyboard focus deterministic", () => {
    render(
      <AdvancePaymentHistoryCard
        history={[pendingRequest]}
        isLoading={false}
        onCancel={vi.fn()}
      />
    );

    expect(screen.getByText("1 yêu cầu")).toBeInTheDocument();
    const disclosure = screen.getByRole("button", { name: /Xem phí và chi tiết/i });
    expect(disclosure).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByRole("button", { name: "Hủy yêu cầu" })).not.toBeInTheDocument();

    fireEvent.click(disclosure);
    const cancelButton = screen.getByRole("button", { name: "Hủy yêu cầu" });
    fireEvent.click(cancelButton);

    const dismissButton = screen.getByRole("button", { name: "Không" });
    expect(dismissButton).toHaveFocus();
    fireEvent.click(dismissButton);
    expect(screen.getByRole("button", { name: "Hủy yêu cầu" })).toHaveFocus();
  });

  it("confirms cancellation with the pending request id", () => {
    const onCancel = vi.fn();
    render(
      <AdvancePaymentHistoryCard
        history={[pendingRequest]}
        isLoading={false}
        onCancel={onCancel}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: /Xem phí và chi tiết/i }));
    fireEvent.click(screen.getByRole("button", { name: "Hủy yêu cầu" }));
    fireEvent.click(screen.getByRole("button", { name: "Xác nhận" }));

    expect(onCancel).toHaveBeenCalledWith(42);
  });

  it("shows a compact month-aware empty state", () => {
    render(
      <AdvancePaymentHistoryCard
        history={[]}
        isLoading={false}
        monthLabel="07/2026"
      />
    );

    expect(screen.getByText("Chưa có yêu cầu trong tháng 07/2026")).toBeInTheDocument();
    // The header badge prefixes the salary period so month navigation has feedback
    // even when the selected month is empty.
    expect(
      screen.getByText((_, element) =>
        element?.tagName === "SPAN" && element.textContent === "Kỳ 07/2026 · 0 yêu cầu"
      )
    ).toBeInTheDocument();
  });

  it("renders a retry action for history errors", () => {
    const onRetry = vi.fn();
    render(
      <AdvancePaymentHistoryCard
        history={[]}
        isLoading={false}
        isError
        onRetry={onRetry}
      />
    );

    fireEvent.click(screen.getByRole("button", { name: "Tải lại" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it.each([
    ["APPROVED", "Đã duyệt"],
    ["COMPLETED", "Hoàn tất"],
    ["FAILED", "Thất bại"],
    ["CANCELLED", "Đã hủy"],
  ] as const)("renders the %s status using the existing API wording", (status, label) => {
    render(
      <AdvancePaymentHistoryCard
        history={[{ ...pendingRequest, status }]}
        isLoading={false}
      />
    );

    expect(screen.getByText(label)).toBeInTheDocument();
  });
});
