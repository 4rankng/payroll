import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import TimesheetsPageMobile from "./index";

const navigate = vi.fn();
const setSearchParams = vi.fn();
const setStatusFilter = vi.fn();
const setSelectedProject = vi.fn();
const openEmployeeDetails = vi.fn();
let routeSearchParams = new URLSearchParams();

vi.mock("react-router-dom", () => ({
  useNavigate: () => navigate,
  useSearchParams: () => [routeSearchParams, setSearchParams],
}));

vi.mock("@tanstack/react-query", () => ({
  useQueryClient: () => ({ setQueryData: vi.fn() }),
}));

vi.mock("@/components/timesheet/mobile/TimesheetPageHeaderMobile", () => ({
  TimesheetPageHeaderMobile: ({
    onPayrollReportExport,
    onBccUpload,
    onBccHistory,
    onPaymentHistory,
  }: {
    onPayrollReportExport?: () => void;
    onBccUpload?: () => void;
    onBccHistory?: () => void;
    onPaymentHistory?: () => void;
  }) => (
    <div>
      {onPayrollReportExport && (
        <button onClick={onPayrollReportExport}>Xuất sao kê</button>
      )}
      {onBccUpload && <button onClick={onBccUpload}>Tải lên BCC</button>}
      {onBccHistory && <button onClick={onBccHistory}>Lịch sử BCC</button>}
      {onPaymentHistory && (
        <button onClick={onPaymentHistory}>Lịch sử trả lương</button>
      )}
    </div>
  ),
}));

vi.mock("@/components/timesheet/TimesheetDisplaySection", () => ({
  TimesheetDisplaySection: () => <div />,
}));

vi.mock("@/components/timesheet/TimesheetsExportDialog", () => ({
  TimesheetsExportDialog: () => null,
}));

vi.mock("@/components/transaction/ExportSaoKeDialog", () => ({
  ExportSaoKeDialog: ({ open }: { open: boolean }) =>
    open ? <div>Hộp thoại xuất sao kê</div> : null,
}));

vi.mock("@/components/timesheet/BCCUploadModal", () => ({
  BCCUploadModal: ({ open }: { open: boolean }) =>
    open ? <div>Hộp thoại tải BCC</div> : null,
}));

vi.mock("@/components/timesheet/UploadHistorySheet", () => ({
  UploadHistorySheet: ({ open }: { open: boolean }) =>
    open ? <div>Lịch sử tải BCC</div> : null,
}));

vi.mock("@/components/payroll/PaymentHistorySheet", () => ({
  PaymentHistorySheet: ({ isOpen }: { isOpen: boolean }) =>
    isOpen ? <div>Danh sách lịch sử trả lương</div> : null,
}));

vi.mock("@/components/employees/MissingBankDetailsSection", () => ({
  MissingBankDetailsSection: ({
    onEmployeeClick,
  }: {
    onEmployeeClick?: (employee: { id: number }) => void;
  }) => (
    <button onClick={() => onEmployeeClick?.({ id: 42 })}>
      Nhân viên thiếu thông tin ngân hàng
    </button>
  ),
}));

vi.mock("@/components/shared/GroupedStatCard", () => ({
  GroupedStatCard: ({
    stats,
  }: {
    stats: Array<{ label: string; onClick?: () => void }>;
  }) => (
    <div>
      {stats.map((stat) =>
        stat.onClick ? (
          <button key={stat.label} onClick={stat.onClick}>
            {stat.label}
          </button>
        ) : null,
      )}
    </div>
  ),
}));

vi.mock("@/components/shared/MobilePageShell", () => ({
  MobilePageShell: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
  MobileSurface: ({ children }: { children: React.ReactNode }) => (
    <div>{children}</div>
  ),
}));

vi.mock("@/components/ui/skeleton", () => ({
  Skeleton: () => <div />,
}));

vi.mock("@/components/sheets/timesheet-entry/mobile/MobileTimesheetEntry", () => ({
  MobileTimesheetEntry: () => <div />,
}));

vi.mock("@/hooks/timesheet/useTimesheetManagement", () => ({
  useTimesheetManagement: () => ({
    selectedProject: "all",
    selectedMonth: "2026-07",
    selectedEmployee: "all",
    statusFilter: "all",
    setSelectedEmployee: vi.fn(),
    setSelectedProject,
    setSelectedMonth: vi.fn(),
    setStatusFilter,
    timesheets: [],
    projects: [],
    isLoading: false,
    handleEdit: vi.fn(),
    handleDelete: vi.fn(),
  }),
}));

vi.mock("@/hooks/useModalNavigation", () => ({
  useTimesheetModals: () => ({
    openTimesheetEntry: vi.fn(),
    openTimesheetDetails: vi.fn(),
  }),
  useEmployeeModals: () => ({
    openEmployeeDetails,
  }),
}));

vi.mock("@/hooks/api/usePayrolls", () => ({
  useExportApprovedTimesheets: () => ({
    isPending: false,
    mutateAsync: vi.fn(),
  }),
}));

vi.mock("@/hooks/api/useSettings", () => ({
  useSettingByKey: () => ({ data: { value: "0" } }),
}));

vi.mock("@/hooks/api/useTimesheetEditRequests", () => ({
  useCreateEditRequest: () => ({
    isPending: false,
    mutateAsync: vi.fn(),
  }),
}));

vi.mock("@/hooks/useTimesheetStatsConfig", () => ({
  useTimesheetStatsConfig: () => ({
    summary: {
      totalEmployees: 2,
      totalEntries: 3,
      approvedEntries: 1,
      pendingApproval: 0,
      pendingEmployees: 0,
    },
  }),
}));

describe("partner mobile timesheet actions", () => {
  beforeEach(() => {
    navigate.mockClear();
    setSearchParams.mockClear();
    setStatusFilter.mockClear();
    setSelectedProject.mockClear();
    openEmployeeDetails.mockClear();
    routeSearchParams = new URLSearchParams();
  });

  it("opens the employee profile drawer from a bank-warning row", () => {
    render(<TimesheetsPageMobile />);

    fireEvent.click(
      screen.getByRole("button", { name: "Nhân viên thiếu thông tin ngân hàng" }),
    );

    expect(openEmployeeDetails).toHaveBeenCalledWith("42");
  });

  it("opens the restored partner workflows and applies the approved filter", () => {
    render(<TimesheetsPageMobile />);

    fireEvent.click(screen.getByRole("button", { name: "Xuất sao kê" }));
    expect(screen.getByText("Hộp thoại xuất sao kê")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Tải lên BCC" }));
    expect(screen.getByText("Hộp thoại tải BCC")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Lịch sử BCC" }));
    expect(screen.getByText("Lịch sử tải BCC")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Lịch sử trả lương" }));
    expect(screen.getByText("Danh sách lịch sử trả lương")).toBeInTheDocument();
    expect(navigate).not.toHaveBeenCalledWith("payment-history");

    fireEvent.click(screen.getByRole("button", { name: "Đã duyệt" }));
    expect(setStatusFilter).toHaveBeenCalledWith("approved");
  });

  it("applies the project scope from the project-list shortcut", () => {
    routeSearchParams = new URLSearchParams("project=42");

    render(<TimesheetsPageMobile />);

    expect(setSelectedProject).toHaveBeenCalledWith("42");
  });
});
