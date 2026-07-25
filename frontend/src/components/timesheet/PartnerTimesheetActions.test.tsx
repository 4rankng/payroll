import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { PartnerTimesheetActions } from "./PartnerTimesheetActions";

describe("PartnerTimesheetActions", () => {
  it("keeps every timesheet action visible and connected", () => {
    const callbacks = {
      onAddTimesheet: vi.fn(),
      onOpenPaymentHistory: vi.fn(),
      onExportTimesheets: vi.fn(),
      onExportStatement: vi.fn(),
      onOpenBccHistory: vi.fn(),
      onUploadBcc: vi.fn(),
    };

    render(<PartnerTimesheetActions {...callbacks} />);

    const actions = [
      ["Nhập công", callbacks.onAddTimesheet],
      ["Lịch sử trả lương", callbacks.onOpenPaymentHistory],
      ["Xuất bảng công", callbacks.onExportTimesheets],
      ["Xuất sao kê", callbacks.onExportStatement],
      ["Lịch sử BCC", callbacks.onOpenBccHistory],
      ["Tải lên BCC", callbacks.onUploadBcc],
    ] as const;

    actions.forEach(([label, callback]) => {
      fireEvent.click(screen.getByRole("button", { name: new RegExp(label) }));
      expect(callback).toHaveBeenCalledOnce();
    });
  });
});
