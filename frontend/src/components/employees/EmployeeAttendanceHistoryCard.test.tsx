import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAttendanceHistory } from "@/hooks/api/useAttendance";
import type { AttendanceRecord } from "@/services/attendance";
import { EmployeeAttendanceHistoryCard } from "./EmployeeAttendanceHistoryCard";

vi.mock("@/hooks/api/useAttendance", () => ({
  useAttendanceHistory: vi.fn(),
}));

const attendanceRecords: AttendanceRecord[] = [
  {
    id: 1,
    project_id: 10,
    employee_id: 20,
    date: "2026-07-09",
    check_in_time: "2026-07-09T08:00:00+07:00",
    check_in_gate: "Cổng chính",
    check_out_time: "2026-07-09T17:00:00+07:00",
    check_out_gate: "Cổng chính",
    earning_amount: 282_400,
    salary_status: "recorded",
    status: "completed",
  },
  {
    id: 2,
    project_id: 10,
    employee_id: 20,
    date: "2026-07-08",
    check_in_time: "2026-07-08T07:58:00+07:00",
    check_in_gate: "Cổng chính",
    salary_status: "not_recorded",
    salary_reject_reason: "Đã hết hạn tan ca (07:58 17:00 21:00)",
    status: "rejected",
  },
];

describe("EmployeeAttendanceHistoryCard", () => {
  beforeEach(() => {
    vi.mocked(useAttendanceHistory).mockReturnValue({
      data: { data: attendanceRecords },
      isLoading: false,
    } as ReturnType<typeof useAttendanceHistory>);
  });

  it("starts collapsed and exposes month summary counts", () => {
    render(<EmployeeAttendanceHistoryCard />);

    const toggle = screen.getByRole("button", { name: /Lịch sử chấm công/i });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.getByText("1 ngày công")).toBeInTheDocument();
    expect(screen.getByText("1 cần kiểm tra")).toBeInTheDocument();
    expect(screen.queryByText("Ngày 9/7/2026")).not.toBeInTheDocument();
  });

  it("opens compact rows and expands only the problem detail", () => {
    render(<EmployeeAttendanceHistoryCard />);

    fireEvent.click(screen.getByRole("button", { name: /Lịch sử chấm công/i }));
    expect(screen.getByText("Ngày 9/7/2026")).toBeInTheDocument();
    expect(screen.getByText("+282.400đ")).toBeInTheDocument();
    expect(screen.queryByText("Quá hạn tan ca")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Chi tiết" }));
    expect(screen.getByText("Quá hạn tan ca")).toBeInTheDocument();
    expect(screen.getByText("21:00")).toBeInTheDocument();
  });
});
