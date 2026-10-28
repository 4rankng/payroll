import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { LedgerPageHeader } from "./LedgerPageHeader";

vi.mock("./CashFlowChart", () => ({ CashFlowChart: () => null }));

describe("LedgerPageHeader action menu (PAYR-20)", () => {
  const handlers = {
    onAddEntry: vi.fn(),
    onAddDoubleEntry: vi.fn(),
    onRecalculateBalance: vi.fn(),
    onSendSaoKePayroll: vi.fn(),
    onSendSaoKeAdvance: vi.fn(),
    onViewSaoKeHistory: vi.fn(),
    onImportOnePayFeeReport: vi.fn(),
    onExportSaoKePayroll: vi.fn(),
    onExportSaoKeAdvance: vi.fn(),
    onUploadSettlementResult: vi.fn(),
    onSimulateSettlement: vi.fn(),
    onRunWalletSettlement: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("sheet 'Tùy chọn khác' có đủ nghiệp vụ export sao kê / phí OnePay / settlement", () => {
    render(<LedgerPageHeader {...handlers} />);

    const expected: Array<[string, () => void]> = [
      ["Xuất sao kê lương", handlers.onExportSaoKePayroll],
      ["Xuất sao kê ứng lương", handlers.onExportSaoKeAdvance],
      ["Nhập phí OnePay", handlers.onImportOnePayFeeReport],
      ["Upload kết quả settlement", handlers.onUploadSettlementResult],
      ["Mô phỏng settlement", handlers.onSimulateSettlement],
    ];
    for (const [label, fn] of expected) {
      // The action sheet closes after each pick — reopen it for the next one.
      fireEvent.click(screen.getByRole("button", { name: "Tùy chọn khác" }));
      fireEvent.click(screen.getByRole("button", { name: label }));
      expect(fn).toHaveBeenCalledTimes(1);
    }
  });
});
