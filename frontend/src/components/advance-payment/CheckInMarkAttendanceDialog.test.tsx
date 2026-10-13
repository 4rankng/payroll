import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CheckInMarkAttendanceDialog } from "./CheckInMarkAttendanceDialog";

const mockUseAdminAttendances = vi.fn();
const mockUseAdminCheckInShifts = vi.fn();
const mockUseAdminCreateCheckIn = vi.fn();
const mockUseApproveAttendance = vi.fn();

vi.mock("@/hooks/api/useAdminAttendance", () => ({
  useAdminAttendances: (...args: unknown[]) => mockUseAdminAttendances(...args),
  useAdminCheckInShifts: (...args: unknown[]) => mockUseAdminCheckInShifts(...args),
  useAdminCreateCheckIn: () => mockUseAdminCreateCheckIn(),
  useApproveAttendance: (...args: unknown[]) => mockUseApproveAttendance(...args),
}));

describe("CheckInMarkAttendanceDialog", () => {
  const defaultEmployee = { id: 612, name: "Quan Quốc Khánh" };
  const defaultProjectId = 58;

  beforeEach(() => {
    vi.clearAllMocks();

    mockUseAdminAttendances.mockReturnValue({
      data: { data: [] },
      isLoading: false,
    });
    mockUseAdminCheckInShifts.mockReturnValue({
      data: { shifts: [] },
      isLoading: false,
      error: null,
    });
    mockUseAdminCreateCheckIn.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    });
    mockUseApproveAttendance.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    });
  });

  it("prioritizes an open attendance record from yesterday and offers checkout approval", () => {
    const mutateApprove = vi.fn();
    mockUseApproveAttendance.mockReturnValue({
      mutate: mutateApprove,
      isPending: false,
      error: null,
    });

    mockUseAdminAttendances.mockReturnValue({
      data: {
        data: [
          {
            id: 69,
            employee_id: 612,
            project_id: 58,
            date: "2026-10-12",
            check_in_time: "2026-10-12T19:30:53+07:00",
            check_out_time: null,
            status: "checked_in",
          },
        ],
      },
      isLoading: false,
    });

    render(
      <CheckInMarkAttendanceDialog
        open
        onOpenChange={vi.fn()}
        employee={defaultEmployee}
        projectId={defaultProjectId}
        projectName="LGD"
      />,
    );

    expect(screen.getByText(/Đã điểm danh lúc 19:30/)).toBeInTheDocument();
    expect(screen.getByText(/\(ngày 12\/10\)/)).toBeInTheDocument();
    const approveBtn = screen.getByRole("button", { name: "Xác nhận đã tan ca" });
    expect(approveBtn).toBeInTheDocument();

    fireEvent.click(approveBtn);
    expect(mutateApprove).toHaveBeenCalledWith(
      { id: 69 },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it("shows completed status when today's attendance is completed", () => {
    const today = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Ho_Chi_Minh",
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    }).format(new Date());

    mockUseAdminAttendances.mockReturnValue({
      data: {
        data: [
          {
            id: 70,
            employee_id: 612,
            project_id: 58,
            date: today,
            check_in_time: `${today}T08:00:00+07:00`,
            check_out_time: `${today}T17:00:00+07:00`,
            status: "completed",
          },
        ],
      },
      isLoading: false,
    });

    render(
      <CheckInMarkAttendanceDialog
        open
        onOpenChange={vi.fn()}
        employee={defaultEmployee}
        projectId={defaultProjectId}
        projectName="LGD"
      />,
    );

    expect(screen.getByText("Nhân viên đã hoàn thành chấm công hôm nay.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Xác nhận đã tan ca" })).not.toBeInTheDocument();
  });

  it("disables creation and shows warning if the selected shift is in the future", () => {
    mockUseAdminAttendances.mockReturnValue({
      data: { data: [] },
      isLoading: false,
    });

    // Future shift starting 2 hours from now
    const futureStart = new Date(Date.now() + 2 * 3600 * 1000).toISOString();
    const futureEnd = new Date(Date.now() + 10 * 3600 * 1000).toISOString();

    mockUseAdminCheckInShifts.mockReturnValue({
      data: {
        shifts: [
          {
            index: 0,
            label: "20:00 - 06:00",
            position: "Phổ Thông",
            start: futureStart,
            end: futureEnd,
            amount: 315000,
          },
        ],
      },
      isLoading: false,
      error: null,
    });

    render(
      <CheckInMarkAttendanceDialog
        open
        onOpenChange={vi.fn()}
        employee={defaultEmployee}
        projectId={defaultProjectId}
        projectName="LGD"
      />,
    );

    expect(screen.getByText("Ca làm việc hôm nay")).toBeInTheDocument();
  });

  it("allows switching work date to yesterday to mark a missed shift", () => {
    mockUseAdminAttendances.mockReturnValue({
      data: { data: [] },
      isLoading: false,
    });

    render(
      <CheckInMarkAttendanceDialog
        open
        onOpenChange={vi.fn()}
        employee={defaultEmployee}
        projectId={defaultProjectId}
        projectName="LGD"
      />,
    );

    const yesterdayBtn = screen.getByRole("button", { name: /Hôm qua/ });
    expect(yesterdayBtn).toBeInTheDocument();

    fireEvent.click(yesterdayBtn);
    expect(screen.getByText("Ca làm việc hôm qua")).toBeInTheDocument();
  });
});
