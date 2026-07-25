import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import EmployeesPageMobile from "./index";

const mocks = vi.hoisted(() => ({
  handlePageChange: vi.fn(),
  navigate: vi.fn(),
}));

vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return {
    ...actual,
    useNavigate: () => mocks.navigate,
  };
});

vi.mock("@/components/shared/MobilePageHeader", () => ({
  MobilePageHeader: ({
    title,
    actions,
  }: {
    title: string;
    actions?: React.ReactNode;
  }) => (
    <header>
      <h1>{title}</h1>
      {actions}
    </header>
  ),
}));

vi.mock("@/components/shared/MobilePageShell", () => ({
  MobilePageShell: ({ children }: { children: React.ReactNode }) => (
    <main>{children}</main>
  ),
  MobileSurface: ({ children }: { children: React.ReactNode }) => (
    <section>{children}</section>
  ),
}));

vi.mock("@/components/shared/MobileSearchInput", () => ({
  MobileSearchInput: () => <input aria-label="Tìm kiếm nhân viên" />,
}));

vi.mock("@/components/employees/EmployeeMobileCard", () => ({
  EmployeeMobileCard: ({
    employee,
  }: {
    employee: { id: number; fullname: string };
  }) => <article>{employee.fullname}</article>,
}));

vi.mock("@/components/employees/EmployeeEmptyStates", () => ({
  EmployeeEmptyStates: () => <div>Không có nhân viên</div>,
}));

vi.mock("@/components/employees/MissingBankDetailsSection", () => ({
  MissingBankDetailsSection: () => null,
}));

vi.mock("@/components/modals/ExportEmployeesModal", () => ({
  ExportEmployeesModal: () => null,
}));

vi.mock("@/hooks/partner-employees/usePartnerEmployeesData", () => ({
  usePartnerEmployeesData: () => ({
    employees: [{ id: 1, fullname: "Nguyễn Văn An" }],
    pagination: {
      page: 1,
      pageSize: 20,
      totalPages: 3,
      totalRecords: 45,
    },
    isLoading: false,
    searchEmployees: vi.fn(),
    clearSearch: vi.fn(),
    searchTerm: "",
    projectId: null,
    filterByProject: vi.fn(),
    statusFilter: undefined,
    month: undefined,
    updateStatusFilter: vi.fn(),
    updateMonth: vi.fn(),
    clearAllFilters: vi.fn(),
    sortBy: "created_at",
    sortOrder: "desc",
    handleSortChange: vi.fn(),
    handlePageChange: mocks.handlePageChange,
  }),
}));

vi.mock("@/hooks/api/useEmployees", () => ({
  useEmployeesSummary: () => ({ data: undefined, isLoading: false }),
}));

vi.mock("@/hooks/api/useProjects", () => ({
  useAssignableProjects: () => ({ data: undefined }),
}));

vi.mock("@/hooks/employees/useEmployeeExport", () => ({
  useEmployeeExport: () => ({
    exportEmployees: vi.fn(),
    isExporting: false,
  }),
}));

vi.mock("@/hooks/useModalNavigation", () => ({
  useEmployeeModals: () => ({
    openEmployeeDetails: vi.fn(),
    openAddEmployee: vi.fn(),
  }),
}));

describe("partner mobile employee pagination", () => {
  beforeEach(() => {
    mocks.handlePageChange.mockClear();
    mocks.navigate.mockClear();
  });

  it("renders the server page range and requests the next server page", () => {
    render(<EmployeesPageMobile />);

    expect(screen.getByText("Nguyễn Văn An")).toBeInTheDocument();
    expect(screen.getByText("1–20 / 45")).toBeInTheDocument();

    const nextPageButton = screen.getByRole("button", { name: "Trang tiếp" });
    expect(nextPageButton).toHaveClass("h-11", "w-11");

    fireEvent.click(nextPageButton);

    expect(mocks.handlePageChange).toHaveBeenCalledTimes(1);
    expect(mocks.handlePageChange).toHaveBeenCalledWith(2);
  });
});
