import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import EmployeeRouter from "./index";

const profileQuery = vi.hoisted(() => ({
  data: undefined as { payment_schedule: string } | undefined,
  isLoading: false,
  isError: false,
  isFetching: false,
  refetch: vi.fn(),
}));

vi.mock("@/hooks/api/useEmployeePortal", () => ({ useEmployeeProfile: () => profileQuery }));
vi.mock("@/pages/employee/EmployeePage", () => ({ default: () => <p>Trang bảng công</p> }));
vi.mock("@/pages/employee/FlexiblePayEmployeePage", () => ({ default: () => <p>Trang ứng lương</p> }));

describe("EmployeeRouter", () => {
  beforeEach(() => {
    profileQuery.data = undefined;
    profileQuery.isLoading = false;
    profileQuery.isError = false;
    profileQuery.isFetching = false;
    profileQuery.refetch.mockReset();
  });

  it("renders non-interactive shell chrome while loading", () => {
    profileQuery.isLoading = true;
    render(<EmployeeRouter />);

    expect(screen.queryByRole("button")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Đang tải thông tin nhân viên")).toBeInTheDocument();
  });

  it("guards retry while a profile refetch is already running", () => {
    profileQuery.isError = true;
    profileQuery.isFetching = true;
    render(<EmployeeRouter />);

    const retryButton = screen.getByRole("button", { name: "Đang tải lại…" });
    expect(retryButton).toBeDisabled();
    fireEvent.click(retryButton);
    expect(profileQuery.refetch).not.toHaveBeenCalled();
  });

  it("routes flexible employees to the advance-pay experience", () => {
    profileQuery.data = { payment_schedule: "flexible" };
    render(<EmployeeRouter />);

    expect(screen.getByText("Trang ứng lương")).toBeInTheDocument();
  });
});
