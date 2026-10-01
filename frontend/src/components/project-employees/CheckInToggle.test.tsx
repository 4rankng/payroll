import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CheckInToggle } from "./CheckInToggle";
import type { ProjectEmployeeAssignment } from "@/types/api/project-employee.types";

const { toggleMutateMock, cancelMutateMock } = vi.hoisted(() => ({
  toggleMutateMock: vi.fn(),
  cancelMutateMock: vi.fn(),
}));

vi.mock("@/hooks/api/useProjectEmployees", () => ({
  useToggleCheckInEnabled: () => ({ isPending: false, mutate: toggleMutateMock }),
  useCancelPendingCheckInEnable: () => ({ isPending: false, mutate: cancelMutateMock }),
}));

const baseAssignment: ProjectEmployeeAssignment = {
  id: 1,
  project_id: 7,
  employee_id: 101,
  employee_name: "Nguyễn Hoàng An",
  employee_cccd: "001201000101",
  employee_code: "NV-101",
  position: "Công nhân",
  start_date: "2026-01-01",
  status: "current",
  created_by: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

describe("CheckInToggle", () => {
  beforeEach(() => {
    toggleMutateMock.mockClear();
    cancelMutateMock.mockClear();
  });

  it("asks for the start month before enabling, instead of assuming one", () => {
    render(<CheckInToggle assignment={{ ...baseAssignment, check_in_enabled: false }} />);

    fireEvent.click(screen.getByRole("switch"));

    expect(toggleMutateMock).not.toHaveBeenCalled();
    fireEvent.click(screen.getByLabelText(/Tháng này/));
    fireEvent.click(screen.getByRole("button", { name: /^Xác nhận từ/ }));

    expect(toggleMutateMock).toHaveBeenCalledWith(
      { projectId: 7, employeeId: 101, enabled: true, startMonth: "this_month" },
      expect.anything(),
    );
  });

  it("lets the admin move a queued next-month activation to this month", async () => {
    render(
      <CheckInToggle
        assignment={{
          ...baseAssignment,
          check_in_enabled: false,
          pending_check_in_enabled: true,
          check_in_effective_from: "2026-11-01",
        }}
      />,
    );

    expect(screen.getByText(/Kích hoạt 01\/11/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Đổi tháng" }));

    expect(await screen.findByText("Đổi tháng kích hoạt")).toBeInTheDocument();
    fireEvent.click(screen.getByLabelText(/Tháng này/));
    fireEvent.click(screen.getByRole("button", { name: "Xác nhận đổi tháng" }));

    expect(toggleMutateMock).toHaveBeenCalledWith(
      { projectId: 7, employeeId: 101, enabled: true, startMonth: "this_month" },
      expect.anything(),
    );
    expect(cancelMutateMock).not.toHaveBeenCalled();
  });

  it("still turns the service off immediately, with no month question", () => {
    render(<CheckInToggle assignment={{ ...baseAssignment, check_in_enabled: true }} />);

    fireEvent.click(screen.getByRole("switch"));

    expect(toggleMutateMock).toHaveBeenCalledWith({
      projectId: 7,
      employeeId: 101,
      enabled: false,
    });
  });
});
