import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import CheckInSettingsPage from "./CheckInSettingsPage";

const {
  useCheckInConfigurableProjectsMock,
  useCheckInConfigurationMock,
  disableInactiveMutateMock,
  disablePendingMutateMock,
} = vi.hoisted(() => ({
  useCheckInConfigurableProjectsMock: vi.fn(),
  useCheckInConfigurationMock: vi.fn(),
  disableInactiveMutateMock: vi.fn(),
  disablePendingMutateMock: vi.fn(),
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
  useDisablePendingCheckInEmployees: () => ({
    isPending: false,
    mutate: disablePendingMutateMock,
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
    pageSize: 20,
    totalPages: 3,
    totalRecords: 50,
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

  it("uses coherent Vietnamese status labels", async () => {
    renderPage();

    expect(useCheckInConfigurableProjectsMock).toHaveBeenCalledWith();

    expect(await screen.findByRole("tab", { name: /Đã điểm danh/ })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /Chưa điểm danh/ })).toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: /Active/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("tab", { name: /Inactive/ })).not.toBeInTheDocument();
  });

  it("keeps row actions compact on desktop and touch-safe on smaller screens", async () => {
    renderPage();

    const actions = await screen.findAllByRole("button", {
      name: "Tắt điểm danh cho Nguyễn Hoàng An",
    });
    expect(actions.some((action) => action.classList.contains("h-8"))).toBe(true);
    expect(actions.some((action) => action.classList.contains("h-11"))).toBe(true);

    const backButton = screen.getByRole("button", { name: "Quay lại" });
    expect(backButton).toHaveClass("h-11", "sm:h-9");
  });

  it("filters to employees without attendance and confirms disabling the entire cohort", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("tab", { name: /Chưa điểm danh/ }));

    await waitFor(() => {
      expect(useCheckInConfigurationMock).toHaveBeenLastCalledWith(
        7,
        { page: 1, pageSize: 20, status: "inactive" },
        true,
      );
    });

    fireEvent.click(
      screen.getByRole("button", { name: "Tắt tất cả 12 nhân viên chưa điểm danh" }),
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

  it("confirms canceling every pending activation in the selected project", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("tab", { name: /Chờ kích hoạt/ }));

    await waitFor(() => {
      expect(useCheckInConfigurationMock).toHaveBeenLastCalledWith(
        7,
        { page: 1, pageSize: 20, status: "pending" },
        true,
      );
    });

    fireEvent.click(screen.getByRole("button", { name: "Hủy chờ tất cả 3 nhân viên" }));
    expect(
      screen.getByRole("heading", { name: "Hủy kích hoạt đang chờ của 3 nhân viên?" }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Xác nhận hủy chờ" }));
    expect(disablePendingMutateMock).toHaveBeenCalledWith(
      { projectId: 7 },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  it("requests the next page from the backend with a bounded page size", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Trang tiếp" }));

    await waitFor(() => {
      expect(useCheckInConfigurationMock).toHaveBeenLastCalledWith(
        7,
        { page: 2, pageSize: 20, status: "enabled" },
        true,
      );
    });
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
