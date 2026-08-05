import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";
import { AttendanceMobileCard } from "./index";

const attendance = {
  id: 9,
  project_id: 2,
  project_name: "Dự án A",
  employee_id: 3,
  employee_name: "Trần Thị Bình",
  date: "2026-07-25",
  check_in_time: "2026-07-25T08:00:00+07:00",
  check_in_lat: 10.7,
  check_in_lng: 106.7,
  check_in_gate: "Cổng A",
  earning_amount: 450000,
  status: "checked_in",
} as AdminAttendanceResponse;

describe("AttendanceMobileCard", () => {
  it("surfaces an admin-approved attendance with no checkout for repair", () => {
    render(
      <AttendanceMobileCard
        row={{ ...attendance, review_action: "approved" }}
        onViewMap={vi.fn()}
        onApprove={vi.fn()}
        onReject={vi.fn()}
      />,
    );

    expect(screen.getByText("Cần duyệt lại")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Duyệt lại" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Từ chối" })).not.toBeInTheDocument();
  });

  it("keeps map, approve, and reject review actions available", () => {
    const onViewMap = vi.fn();
    const onApprove = vi.fn();
    const onReject = vi.fn();

    render(
      <AttendanceMobileCard
        row={attendance}
        onViewMap={onViewMap}
        onApprove={onApprove}
        onReject={onReject}
      />,
    );

    const actions = [
      ["Bản đồ", onViewMap],
      ["Duyệt", onApprove],
      ["Từ chối", onReject],
    ] as const;

    actions.forEach(([label, callback]) => {
      const button = screen.getByRole("button", { name: label });
      expect(button).toHaveClass("min-h-11");
      fireEvent.click(button);
      expect(callback).toHaveBeenCalledWith(attendance);
    });
  });

  it("hides terminal review actions using the desktop rules", () => {
    render(
      <AttendanceMobileCard
        row={{ ...attendance, status: "completed", review_action: "rejected" }}
        onViewMap={vi.fn()}
        onApprove={vi.fn()}
        onReject={vi.fn()}
      />,
    );

    expect(screen.queryByRole("button", { name: "Duyệt" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Từ chối" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Bản đồ" })).toBeInTheDocument();
  });

  it("offers repair when an approval has no persisted checkout", () => {
    const onApprove = vi.fn();
    const corrupted = { ...attendance, status: "completed", review_action: "approved" as const };

    render(
      <AttendanceMobileCard
        row={corrupted}
        onViewMap={vi.fn()}
        onApprove={onApprove}
        onReject={vi.fn()}
      />,
    );

    const repairButton = screen.getByRole("button", { name: "Duyệt lại" });
    expect(screen.getByText("Cần duyệt lại")).toBeInTheDocument();
    expect(screen.queryByText("Hoàn thành")).not.toBeInTheDocument();
    fireEvent.click(repairButton);
    expect(onApprove).toHaveBeenCalledWith(corrupted);
  });
});
