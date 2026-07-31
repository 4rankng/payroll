import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { CheckInBulkDialog } from "./CheckInBulkDialog";

const {
  useProjectsMock,
  useProjectEmployeesMock,
  toggleMutateMock,
  bulkMutateMock,
} = vi.hoisted(() => ({
  useProjectsMock: vi.fn(),
  useProjectEmployeesMock: vi.fn(),
  toggleMutateMock: vi.fn(),
  bulkMutateMock: vi.fn(),
}));

vi.mock("@/hooks/api/useProjects", () => ({
  useProjects: (...args: unknown[]) => useProjectsMock(...args),
}));

vi.mock("@/hooks/api/useProjectEmployees", () => ({
  useProjectEmployees: (...args: unknown[]) => useProjectEmployeesMock(...args),
  useToggleCheckInEnabled: () => ({
    isPending: false,
    mutate: toggleMutateMock,
  }),
  useBulkToggleCheckInEnabled: () => ({
    isPending: false,
    mutate: bulkMutateMock,
  }),
}));

const employees = Array.from({ length: 50 }, (_, index) => ({
  id: index + 1,
  project_id: 7,
  employee_id: index + 1,
  employee_name: index === 0 ? "Việt Duy" : `Nhân viên ${index + 1}`,
  employee_cccd: String(100_000_000_000 + index),
  employee_code: `NV-${index + 1}`,
  position: "phổ thông",
  start_date: "2026-01-01",
  last_date: null,
  check_in_enabled: false,
  created_by: 1,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
}));

describe("CheckInBulkDialog pagination and search", () => {
  beforeEach(() => {
    useProjectsMock.mockReturnValue({
      data: {
        data: [
          {
            id: 7,
            name: "LGD",
            code: "LGD",
            is_flexible: true,
          },
        ],
      },
    });
    useProjectEmployeesMock.mockReturnValue({
      data: {
        status: "success",
        data: employees,
        pagination: {
          page: 1,
          pageSize: 50,
          totalPages: 3,
          totalRecords: 120,
        },
        message: "",
      },
      isLoading: false,
      isFetching: false,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("requests 50 employees per page and displays the backend total", async () => {
    render(<CheckInBulkDialog open onOpenChange={vi.fn()} />);

    await screen.findByText("120 nhân viên");

    expect(useProjectEmployeesMock).toHaveBeenLastCalledWith(
      7,
      {
        page: 1,
        pageSize: 50,
        status: "current",
      },
      true,
    );
    expect(screen.getByRole("button", { name: "Trang trước" })).toHaveClass(
      "h-11",
      "w-11",
    );
    expect(screen.getByRole("button", { name: "Trang sau" })).toHaveClass(
      "h-11",
      "w-11",
    );

    fireEvent.click(screen.getByRole("button", { name: "Trang sau" }));

    expect(useProjectEmployeesMock).toHaveBeenLastCalledWith(
      7,
      {
        page: 2,
        pageSize: 50,
        status: "current",
      },
      true,
    );
  });

  it("debounces the search term into the backend query and resets pagination", async () => {
    vi.useFakeTimers();
    render(<CheckInBulkDialog open onOpenChange={vi.fn()} />);

    await act(async () => {
      await Promise.resolve();
    });
    fireEvent.click(screen.getByRole("button", { name: "Trang sau" }));
    fireEvent.change(screen.getByLabelText("Tìm kiếm nhân viên"), {
      target: { value: "viet duy" },
    });

    await act(async () => {
      vi.advanceTimersByTime(300);
    });

    expect(useProjectEmployeesMock).toHaveBeenLastCalledWith(
      7,
      {
        page: 1,
        pageSize: 50,
        search: "viet duy",
        status: "current",
      },
      true,
    );
  });
});
