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
  it("starts collapsed and keeps cancellation keyboard focus deterministic", () => {
    render(
      <AdvancePaymentHistoryCard
        history={[pendingRequest]}
        isLoading={false}
        onCancel={vi.fn()}
      />
    );

    const disclosure = screen.getByRole("button", { name: /Lịch sử yêu cầu/i });
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

    fireEvent.click(screen.getByRole("button", { name: /Lịch sử yêu cầu/i }));
    fireEvent.click(screen.getByRole("button", { name: "Hủy yêu cầu" }));
    fireEvent.click(screen.getByRole("button", { name: "Xác nhận" }));

    expect(onCancel).toHaveBeenCalledWith(42);
  });
});
