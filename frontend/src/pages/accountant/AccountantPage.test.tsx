import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import AccountantPage from "./AccountantPage";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  logout: vi.fn(),
  timesheets: {} as Record<string, unknown>,
  bulkApprove: { mutate: vi.fn(), isPending: false },
  exportBulk: { mutate: vi.fn(), isPending: false },
  exportSaoKe: { mutate: vi.fn(), isPending: false },
}));

vi.mock("react-router-dom", () => ({ useNavigate: () => mocks.navigate }));
vi.mock("@/contexts/AuthContext", () => ({
  useAuth: () => ({ user: { name: "Kế Toán QA" }, logout: mocks.logout }),
}));
vi.mock("@/hooks/api/useTimesheets", () => ({
  useTimesheets: () => mocks.timesheets,
  useBulkApproveTimesheets: () => mocks.bulkApprove,
}));
vi.mock("@/hooks/api/usePayrolls", () => ({
  useExportBulkTransfer: () => mocks.exportBulk,
  useExportPayrollReport: () => mocks.exportSaoKe,
}));
vi.mock("@/components/timesheet/BulkTransferExportDialog", () => ({
  BulkTransferExportDialog: () => null,
}));
vi.mock("@/components/timesheet/BulkTransferResultUploadDialog", () => ({
  BulkTransferResultUploadDialog: () => null,
}));
vi.mock("@/components/timesheet/PayrollReportExportDialog", () => ({
  PayrollReportExportDialog: () => null,
}));

const rows = [
  { id: 11, employeeName: "Nguyễn Văn An", employeeCCCD: "000000000001", projectName: "Dự án A", date: "2026-09-28", hours_worked: 8, amount: 800000 },
  { id: 12, employeeName: "Trần Thị Bính", employeeCode: "EMP02", projectName: "Dự án A", date: "2026-09-29", hours_worked: 6, amount: 600000 },
];

beforeEach(() => {
  vi.clearAllMocks();
  mocks.bulkApprove.isPending = false;
  mocks.timesheets = { data: { data: rows, pagination: { totalRecords: 2 } }, isLoading: false, isError: false };
});

describe("Accountant console", () => {
  it("logs out from the header", () => {
    render(<AccountantPage />);
    fireEvent.click(screen.getByRole("button", { name: "Đăng xuất" }));
    expect(mocks.logout).toHaveBeenCalledOnce();
    expect(mocks.navigate).toHaveBeenCalledWith("/login", { replace: true });
  });

  it("bulk-approves exactly the rows ticked, all or individually", () => {
    render(<AccountantPage />);
    expect(screen.getByRole("region", { name: "Chấm công chờ duyệt" })).toBeInTheDocument();
    expect(screen.getByText("Nguyễn Văn An")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("checkbox", { name: "Chọn tất cả" }));
    expect(screen.getByText(/Đã chọn 2\/2/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Duyệt (2)" }));
    expect(mocks.bulkApprove.mutate).toHaveBeenCalledWith({ timesheet_ids: [11, 12] }, expect.anything());

    // Untick one row → only the remaining id may be submitted.
    fireEvent.click(screen.getByRole("checkbox", { name: "Chọn Nguyễn Văn An" }));
    fireEvent.click(screen.getByRole("button", { name: "Duyệt (1)" }));
    expect(mocks.bulkApprove.mutate).toHaveBeenLastCalledWith({ timesheet_ids: [12] }, expect.anything());
  });

  it("clears stale selections when the week changes and warns about hidden rows", () => {
    mocks.timesheets = { data: { data: rows, pagination: { totalRecords: 3 } }, isLoading: false, isError: false };
    render(<AccountantPage />);
    fireEvent.click(screen.getByRole("checkbox", { name: "Chọn tất cả" }));
    expect(screen.getByText(/Đã chọn 2\/2/)).toBeInTheDocument();
    expect(screen.getByText(/Còn 1 dòng chưa hiển thị/)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Từ ngày"), { target: { value: "2026-09-01" } });
    expect(screen.getByText("Đã chọn 0/2")).toBeInTheDocument();
    expect(mocks.bulkApprove.mutate).not.toHaveBeenCalled();
  });
});
