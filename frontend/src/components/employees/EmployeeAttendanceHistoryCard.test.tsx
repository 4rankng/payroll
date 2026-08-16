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
  {
    id: 3,
    project_id: 10,
    employee_id: 20,
    date: "2026-07-07",
    check_in_time: "2026-07-07T08:05:00+07:00",
    check_out_time: "2026-07-07T17:01:00+07:00",
    earning_amount: 282_400,
    salary_status: "recorded",
    status: "completed",
  },
  {
    id: 4,
    project_id: 10,
    employee_id: 20,
    date: "2026-07-06",
    check_in_time: "2026-07-06T08:02:00+07:00",
    check_out_time: "2026-07-06T17:04:00+07:00",
    earning_amount: 282_400,
    salary_status: "recorded",
    status: "completed",
  },
];

describe("EmployeeAttendanceHistoryCard", () => {
  beforeEach(() => {
    vi.mocked(useAttendanceHistory).mockReturnValue({
      data: { data: attendanceRecords },
      isLoading: false,
    } as ReturnType<typeof useAttendanceHistory>);
  });

  it("shows a three-record preview and exposes month summary counts", () => {
    render(<EmployeeAttendanceHistoryCard />);

    expect(screen.getByText("3 ngày công")).toBeInTheDocument();
    expect(screen.getByText("1 cần kiểm tra")).toBeInTheDocument();
    expect(screen.getByText("Ngày 9/7/2026")).toBeInTheDocument();
    expect(screen.getByText("Ngày 7/7/2026")).toBeInTheDocument();
    expect(screen.getAllByText((_, element) => element?.tagName === "P" && /^\d{2}:\d{2} — \d{2}:\d{2}$/.test(element.textContent ?? ""))).toHaveLength(2);
    expect(screen.getByText((_, element) => element?.tagName === "P" && /^\d{2}:\d{2} — --:--$/.test(element.textContent ?? ""))).toBeInTheDocument();
    expect(screen.queryByText("Ngày 6/7/2026")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Xem toàn bộ lịch chấm công" })).toHaveAttribute("aria-expanded", "false");
  });

  it("reveals the current month history and expands only the problem detail", () => {
    render(<EmployeeAttendanceHistoryCard />);

    fireEvent.click(screen.getByRole("button", { name: "Xem toàn bộ lịch chấm công" }));
    expect(screen.getByText("Ngày 6/7/2026")).toBeInTheDocument();
    expect(screen.getAllByText((_, node) => node?.textContent === "+282.400₫")).toHaveLength(3);
    expect(screen.queryByText("Quá hạn tan ca")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Xem lý do" }));
    expect(screen.getByText("Quá hạn tan ca")).toBeInTheDocument();
    expect(screen.getByText("21:00")).toBeInTheDocument();
  });
});
