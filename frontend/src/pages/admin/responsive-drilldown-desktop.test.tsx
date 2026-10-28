import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import ActivityUsersPage from "./DashboardPage/ActivityUsersPage";
import LendersPage from "./LoansPage/LendersPage";
import AdvancePaymentsEmployeeListPage from "./AdvancePaymentsPage/EmployeeListPage";

const mocks = vi.hoisted(() => ({
  useIsMobile: vi.fn(() => false),
}));

vi.mock("@/hooks/useBreakpoint", () => ({
  useIsMobile: mocks.useIsMobile,
  useMediaQuery: () => true,
}));
vi.mock("@/components/shared/AdminPageFrame", () => ({
  AdminPageCanvas: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminPageHeaderCard: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminSectionCard: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AdminFilterRow: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/shared/PageHeader", () => ({
  PageHeader: ({ title }: { title: string }) => <h1>{title}</h1>,
}));
vi.mock("@/components/employees/details/EmployeeDetailsSheet", () => ({
  EmployeeDetailsSheet: () => null,
}));
vi.mock("@/components/sheets/AddLenderSheet", () => ({ AddLenderSheet: () => null }));
vi.mock("@/components/sheets/EditLenderSheet", () => ({ EditLenderSheet: () => null }));
vi.mock("@/components/advance-payment/EmployeeDetailSheet", () => ({
  EmployeeDetailSheet: () => null,
}));
vi.mock("@/hooks/api/useDashboard", () => ({
  useActiveEmployeesBySchedule: () => ({
    data: [
      {
        user_id: 1,
        employee_id: 11,
        fullname: "Nguyễn Văn A",
        username: "nva",
        role: "employee",
        payment_schedule: "weekly",
        last_login: "2026-10-28T10:00:00Z",
      },
    ],
    isLoading: false,
  }),
}));
vi.mock("@/hooks/api/useEmployees", () => ({
  useEmployee: () => ({ data: undefined }),
  useUpdateEmployee: () => ({ mutate: vi.fn() }),
  useDeleteEmployee: () => ({ mutate: vi.fn() }),
}));
vi.mock("@/hooks/api/useLoans", () => ({
  useLenders: () => ({
    data: {
      data: [
        {
          id: 5,
          name: "Chủ nợ A",
          cccd: "001",
          email: "a@x.com",
          mobile: "0900",
          bank_account_number: "123",
          bank_account_name: "A",
        },
      ],
      pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 1 },
    },
    isLoading: false,
  }),
}));
vi.mock("@/hooks/advance-payment/useAdvancePaymentsPage", () => ({
  useAdvancePaymentsPage: () => ({
    flexPayEmployees: [
      {
        employeeId: 7,
        fullname: "Trần B",
        cccd: "002",
        username: "tb",
        bank: null,
        project: { id: 1, name: "P1" },
        forMonth: "2026-10",
        maxAdvanceAmount: 1,
        utilizedAmount: 0,
        availableAmount: 1,
        totalFeeGenerated: 0,
        pendingAmount: 0,
        completedRequestsCount: 0,
        pendingRequestsCount: 0,
      },
    ],
    flexPaySearchInput: "",
    handleFlexPaySearch: vi.fn(),
    handleExportFlexPayEmployees: vi.fn(),
    exportFlexPayMutation: { isPending: false },
    availableMonths: [],
    selectedViewMonth: undefined,
    setSelectedViewMonth: vi.fn(),
    flexPayFilters: { sortBy: "max_advance_amount", sortOrder: "DESC" },
    handleFlexPaySort: vi.fn(),
    flexPayPagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 1 },
    handleFlexPayPageChange: vi.fn(),
  }),
}));

describe("Drill-down desktop variants render dense tables (PAYR-23)", () => {
  const renderPage = (node: React.ReactNode) =>
    render(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>{node}</MemoryRouter>
      </QueryClientProvider>,
    );

  beforeEach(() => {
    vi.clearAllMocks();
    mocks.useIsMobile.mockReturnValue(false);
  });

  it("ActivityUsersPage desktop render bảng", () => {
    renderPage(<ActivityUsersPage />);
    expect(screen.getByRole("table")).toBeTruthy();
    expect(screen.getByText("Nguyễn Văn A")).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Lịch lương" })).toBeTruthy();
  });

  it("LendersPage desktop render bảng", () => {
    renderPage(<LendersPage />);
    expect(screen.getByRole("table")).toBeTruthy();
    expect(screen.getByText("Chủ nợ A")).toBeTruthy();
    expect(screen.getByRole("columnheader", { name: "Số tài khoản" })).toBeTruthy();
  });

  it("EmployeeListPage desktop render bảng (ResponsiveTable ở desktop width)", () => {
    renderPage(<AdvancePaymentsEmployeeListPage />);
    expect(screen.getByRole("table")).toBeTruthy();
    expect(screen.getByText("Trần B")).toBeTruthy();
  });
});
