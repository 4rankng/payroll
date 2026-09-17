import type { ReactNode } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useEmployeeTimesheetsInfinite } from "@/hooks/api/useEmployeePortal";
import { useSettingByKey } from "@/hooks/api/useSettings";
import EmployeePage from ".";

vi.mock("@/contexts/AuthContext", () => ({ useAuth: () => ({ logout: vi.fn() }) }));
vi.mock("@/hooks/api/useEmployeePortal", () => ({
  useEmployeeProfile: () => ({ data: { id: 1, fullname: "Nguyễn An" }, isLoading: false }),
  useEmployeeTimesheetsInfinite: vi.fn(),
  useUpdateEmployeePassword: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));
vi.mock("@/hooks/api/useSettings", () => ({ useSettingByKey: vi.fn() }));
vi.mock("@/hooks/api/useNotifications", () => ({ useUnreadNotifications: () => ({ data: { count: 0 } }) }));
vi.mock("@/hooks/use-infinite-scroll", () => ({ useInfiniteScroll: () => ({ observerRef: vi.fn() }) }));
vi.mock("@/components/employees/EmployeeMobileShell", () => ({ EmployeeMobileShell: ({ children, canopy }: { children: ReactNode; canopy?: ReactNode }) => <>{canopy}{children}</> }));
vi.mock("@/components/employees/EmployeeAdBanner", () => ({ EmployeeAdBanner: () => null }));
vi.mock("@/components/employees/EmployeeBankInfoCard", () => ({ EmployeeBankInfoCard: () => null }));
vi.mock("@/components/employees/ChangePasswordSheet", () => ({ ChangePasswordSheet: () => null }));
vi.mock("@/components/notifications/NotificationSheet", () => ({ NotificationSheet: () => null }));

const refetchTimesheets = vi.fn();
const refetchSetting = vi.fn();
const timesheetQuery = {
  data: { pages: [{ data: [], pagination: { totalRecords: 0 } }] },
  isLoading: false,
  isError: false,
  isFetching: false,
  isFetchingNextPage: false,
  hasNextPage: false,
  refetch: refetchTimesheets,
  fetchNextPage: vi.fn(),
};
const settingQuery = { data: { value: "0.7" }, isLoading: false, isError: false, isFetching: false, refetch: refetchSetting };

describe("EmployeePage data recovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useEmployeeTimesheetsInfinite).mockReturnValue(timesheetQuery as unknown as ReturnType<typeof useEmployeeTimesheetsInfinite>);
    vi.mocked(useSettingByKey).mockReturnValue(settingQuery as unknown as ReturnType<typeof useSettingByKey>);
  });

  it.each(["timesheets", "payment setting"])("does not show zero wages or empty records when %s fails", (failedQuery) => {
    if (failedQuery === "timesheets") {
      vi.mocked(useEmployeeTimesheetsInfinite).mockReturnValue({ ...timesheetQuery, data: undefined, isError: true } as unknown as ReturnType<typeof useEmployeeTimesheetsInfinite>);
    } else {
      vi.mocked(useSettingByKey).mockReturnValue({ ...settingQuery, data: undefined, isError: true } as unknown as ReturnType<typeof useSettingByKey>);
    }
    render(<MemoryRouter><EmployeePage /></MemoryRouter>);

    expect(screen.getByRole("alert")).toHaveTextContent("Chưa tải được bảng công");
    expect(screen.queryByText("Chưa có bảng công")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Ẩn số tiền" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Xem tháng trước" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Tải lại" }));
    expect(refetchTimesheets).toHaveBeenCalledOnce();
    expect(refetchSetting).toHaveBeenCalledOnce();
  });

  it("shows an empty month only after both dependencies succeed", () => {
    render(<MemoryRouter><EmployeePage /></MemoryRouter>);
    expect(screen.getByText("Chưa có bảng công")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ẩn số tiền" })).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
});
