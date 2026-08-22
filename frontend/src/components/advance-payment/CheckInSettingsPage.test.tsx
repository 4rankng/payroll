import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CheckInSettingsPage from "./CheckInSettingsPage";

const {
  useCheckInConfigurableProjectsMock,
  useCheckInConfigurationMock,
  disableInactiveMutateMock,
} = vi.hoisted(() => ({
  useCheckInConfigurableProjectsMock: vi.fn(),
  useCheckInConfigurationMock: vi.fn(),
  disableInactiveMutateMock: vi.fn(),
}));

vi.mock("@/hooks/api/useProjectEmployees", () => ({
  useCheckInConfigurableProjects: (...args: unknown[]) =>
    useCheckInConfigurableProjectsMock(...args),
  useCheckInConfiguration: (...args: unknown[]) =>
    useCheckInConfigurationMock(...args),
  useDisableInactiveCheckInEmployees: () => ({
    isPending: false,
    mutate: disableInactiveMutateMock,
  }),
  useToggleCheckInEnabled: () => ({
    isPending: false,
    mutate: vi.fn(),
  }),
}));

const configuration = {
  employees: [
    {
      assignment_id: 1,
      project_id: 7,
      employee_id: 101,
      employee_name: "Nguyễn Hoàng An",
      employee_cccd: "001201000101",
      employee_code: "NV-101",
      check_in_enabled: true,
      pending_check_in_enable: false,
      attendance_count: 0,
      last_check_in_at: null,
    },
  ],
  summary: {
    enabled: 20,
    active: 8,
    inactive: 12,
    pending: 3,
  },
  month: "2026-08",
  pagination: {
    page: 1,
    pageSize: 50,
    totalPages: 1,
    totalRecords: 12,
  },
};

function renderPage(initialPath = "/admin/advance-payments/check-in-settings") {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/admin/advance-payments/check-in-settings" element={<CheckInSettingsPage />} />
        <Route path="/admin/advance-payments" element={<div>Trang ứng lương</div>} />
        <Route path="/adv-partner/advance-payments/check-in-settings" element={<CheckInSettingsPage />} />
        <Route path="/adv-partner/advance-payments" element={<div>Trang ứng lương đối tác</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("CheckInSettingsPage", () => {
  beforeEach(() => {
    useCheckInConfigurableProjectsMock.mockReturnValue({
      data: [{ id: 7, name: "LGD", code: "LGD", status: "active", is_flexible: true }],
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    });
    useCheckInConfigurationMock.mockReturnValue({
      data: configuration,
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    });
  });

  it("uses the exact short Active and Inactive labels", async () => {
    renderPage();

    expect(useCheckInConfigurableProjectsMock).toHaveBeenCalledWith();

    expect(await screen.findByRole("tab", { name: /Active/ })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Inactive/ })).toBeInTheDocument();
    expect(screen.queryByText("Đã điểm danh tháng này")).not.toBeInTheDocument();
    expect(screen.queryByText("Chưa điểm danh tháng này")).not.toBeInTheDocument();
  });

  it("filters to Inactive and confirms disabling the entire cohort", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("tab", { name: /Inactive/ }));

    await waitFor(() => {
      expect(useCheckInConfigurationMock).toHaveBeenLastCalledWith(
        7,
        { page: 1, pageSize: 50, status: "inactive" },
        true,
      );
    });

    fireEvent.click(
      screen.getByRole("button", { name: "Tắt 12 nhân viên Inactive" }),
    );
    expect(
      screen.getByRole("heading", { name: "Tắt điểm danh cho 12 nhân viên?" }),
    ).toBeInTheDocument();
    const confirmation = screen.getByRole("alertdialog");
    expect(within(confirmation).getByText(/LGD/)).toBeInTheDocument();
    expect(within(confirmation).getByText(/tháng 08\/2026/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Xác nhận tắt" }));
    expect(disableInactiveMutateMock).toHaveBeenCalledWith(
      { projectId: 7 },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it("returns to the role-specific advance payment page", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Quay lại" }));
    expect(screen.getByText("Trang ứng lương")).toBeInTheDocument();
  });

  it("returns Advance Partner to the matching role route", async () => {
    renderPage("/adv-partner/advance-payments/check-in-settings");

    fireEvent.click(await screen.findByRole("button", { name: "Quay lại" }));
    expect(screen.getByText("Trang ứng lương đối tác")).toBeInTheDocument();
  });

  it("does not expose stale employee actions while switching projects", async () => {
    useCheckInConfigurationMock.mockReturnValue({
      data: configuration,
      isLoading: false,
      isFetching: true,
      isPlaceholderData: true,
      isError: false,
      refetch: vi.fn(),
    });

    renderPage();

    expect(await screen.findByText("Cấu hình điểm danh")).toBeInTheDocument();
    expect(screen.queryByText("Nguyễn Hoàng An")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Tắt điểm danh cho Nguyễn Hoàng An" }),
    ).not.toBeInTheDocument();
  });

  it("distinguishes a project catalogue error from an empty project list", async () => {
    const refetch = vi.fn();
    useCheckInConfigurableProjectsMock.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      refetch,
    });

    renderPage();

    expect(await screen.findByText("Không thể tải danh sách dự án")).toBeInTheDocument();
    expect(screen.queryByText("Không có dự án linh động")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Thử lại" }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });
});
