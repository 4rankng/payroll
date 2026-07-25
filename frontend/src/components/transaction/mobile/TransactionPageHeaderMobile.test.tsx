import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { TransactionPageHeaderMobile } from "./TransactionPageHeaderMobile";

describe("TransactionPageHeaderMobile", () => {
  it("exposes settlement simulation in the mobile action sheet", () => {
    const onSimulateSettlement = vi.fn();

    render(
      <TransactionPageHeaderMobile
        onAddTransaction={vi.fn()}
        onImportTransactions={vi.fn()}
        onViewHistory={vi.fn()}
        onSimulateSettlement={onSimulateSettlement}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Tùy chọn khác" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Mô phỏng đối soát" }),
    );

    expect(onSimulateSettlement).toHaveBeenCalledOnce();
  });
});
