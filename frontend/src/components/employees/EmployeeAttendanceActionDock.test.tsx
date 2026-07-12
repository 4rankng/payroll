import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { EmployeeAttendanceActionDock } from "./EmployeeAttendanceActionDock";

describe("EmployeeAttendanceActionDock", () => {
  it("renders compact advance and attendance actions", () => {
    const onAdvanceRequest = vi.fn();
    const onAttendanceAction = vi.fn();

    render(
      <EmployeeAttendanceActionDock
        action="check_out"
        actionLabel="Tan ca"
        onAdvanceRequest={onAdvanceRequest}
        onAttendanceAction={onAttendanceAction}
      />
    );

    const toolbar = screen.getByRole("toolbar", { name: "Hành động nhân viên" });
    expect(toolbar).toBeInTheDocument();
    expect(toolbar).toHaveClass("employee-attendance-action-dock");
    expect(toolbar).not.toHaveClass("lg:hidden");
    const advanceButton = screen.getByRole("button", { name: "Ứng lương" });
    const attendanceButton = screen.getByRole("button", { name: "Tan ca" });
    expect(advanceButton).toHaveClass("h-11");
    expect(attendanceButton).toHaveClass("h-11");

    fireEvent.click(advanceButton);
    fireEvent.click(attendanceButton);
    expect(onAdvanceRequest).toHaveBeenCalledOnce();
    expect(onAttendanceAction).toHaveBeenCalledOnce();
  });

  it("keeps advance available while attendance is not ready", () => {
    const onAdvanceRequest = vi.fn();

    render(
      <EmployeeAttendanceActionDock
        action="loading"
        actionLabel="Đang kiểm tra GPS…"
        actionDisabled
        onAdvanceRequest={onAdvanceRequest}
      />
    );

    expect(screen.getByRole("button", { name: "Đang kiểm tra GPS…" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Ứng lương" }));
    expect(onAdvanceRequest).toHaveBeenCalledOnce();
  });
});
