import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Employee } from "@/types/api/employee.types";
import { EmployeeMobileCard } from "./EmployeeMobileCard";

const employee = {
  id: 17,
  fullname: "Nguyễn Văn An",
  cccd: "001234567890",
  status: "active",
  timesheet_summary: {
    pending_timesheets: 3,
  },
} as Employee;

describe("EmployeeMobileCard", () => {
  it("opens pending timesheets without opening employee details", () => {
    const onClick = vi.fn();
    const onPendingTimesheets = vi.fn();

    render(
      <EmployeeMobileCard
        employee={employee}
        onClick={onClick}
        onPendingTimesheets={onPendingTimesheets}
      />,
    );

    const pendingButton = screen.getByRole("button", {
      name: "3 bảng công chờ duyệt",
    });
    expect(pendingButton).toHaveClass("min-h-11");

    fireEvent.click(pendingButton);

    expect(onPendingTimesheets).toHaveBeenCalledWith(employee);
    expect(onClick).not.toHaveBeenCalled();

    fireEvent.keyDown(pendingButton, { key: " " });
    expect(onClick).not.toHaveBeenCalled();
  });
});
