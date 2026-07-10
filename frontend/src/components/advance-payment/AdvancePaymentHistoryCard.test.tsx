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
  it("uses transaction-shaped skeletons while history is loading", () => {
    render(<AdvancePaymentHistoryCard history={[]} isLoading />);

    expect(screen.getByLabelText("Đang tải lịch sử yêu cầu")).toBeInTheDocument();
    expect(screen.queryByText("Chưa có yêu cầu ứng lương")).not.toBeInTheDocument();
  });

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

  it("shows a compact all-time empty state", () => {
    render(
      <AdvancePaymentHistoryCard
        history={[]}
        isLoading={false}
      />
    );

    expect(screen.getByText("Chưa có yêu cầu ứng lương")).toBeInTheDocument();
    expect(screen.getByText("Yêu cầu mới sẽ xuất hiện tại đây.")).toBeInTheDocument();
    expect(screen.getByText("0 yêu cầu")).toBeInTheDocument();
  });

  it("shows five rows initially and makes older all-time requests scrollable", () => {
    const history = Array.from({ length: 6 }, (_, index) => ({
      ...pendingRequest,
      id: index + 1,
      status: "COMPLETED" as const,
      requestAmount: 500_000 + index * 10_000,
      createdAt: `2026-07-${String(10 - index).padStart(2, "0")}T08:00:00+07:00`,
    }));

    render(
      <AdvancePaymentHistoryCard
        history={history}
        totalCount={12}
        isLoading={false}
      />
    );

    expect(screen.getByText("12 yêu cầu")).toBeInTheDocument();
    const scrollRegion = screen.getByLabelText("Lịch sử yêu cầu, cuộn để xem thêm");
    expect(scrollRegion).toHaveClass("max-h-[390px]", "overflow-y-auto");
    const rows = screen.getAllByRole("button", { name: /Xem phí và chi tiết/i });
    expect(rows).toHaveLength(6);
    expect(rows[0]).toHaveTextContent("10/07/2026");
    expect(rows[5]).toHaveTextContent("05/07/2026");
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
