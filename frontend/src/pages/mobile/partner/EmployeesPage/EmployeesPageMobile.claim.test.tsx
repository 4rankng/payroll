import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import EmployeesPageMobile from "./index";

const mocks = vi.hoisted(() => ({
  mutate: vi.fn(),
  openEmployeeDetails: vi.fn(),
}));

const poolEmployee = {
  id: 101,
  fullname: "Nguyễn Pool",
  is_accessible: false,
};
const managedEmployee = {
  id: 102,
  fullname: "Trần Managed",
  is_accessible: true,
};

vi.mock("@/hooks/partner-employees/usePartnerEmployeesData", () => ({
  usePartnerEmployeesData: () => ({
    employees: [poolEmployee, managedEmployee],
    isLoading: false,
    searchEmployees: vi.fn(),
    clearSearch: vi.fn(),
    searchTerm: "",
    projectId: undefined,
    filterByProject: vi.fn(),
    statusFilter: undefined,
    month: undefined,
    updateStatusFilter: vi.fn(),
    updateMonth: vi.fn(),
    clearAllFilters: vi.fn(),
    sortBy: undefined,
    sortOrder: undefined,
    handleSortChange: vi.fn(),
    pagination: { page: 1, pageSize: 20, totalPages: 1, totalRecords: 2 },
    handlePageChange: vi.fn(),
  }),
}));
vi.mock("@/hooks/api/useEmployees", () => ({
  useEmployeesSummary: () => ({ data: undefined, isLoading: false }),
  useRequestEmployeeAccess: () => ({ mutate: mocks.mutate, isPending: false }),
}));
vi.mock("@/hooks/useModalNavigation", () => ({
  useEmployeeModals: () => ({ openEmployeeDetails: mocks.openEmployeeDetails, openAddEmployee: vi.fn() }),
}));
vi.mock("@/hooks/api/useProjects", () => ({
  useAssignableProjects: () => ({ data: { data: [] } }),
}));
vi.mock("@/hooks/employees/useEmployeeExport", () => ({
  useEmployeeExport: () => ({ exportEmployees: vi.fn(), isExporting: false }),
}));
vi.mock("@/components/modals/ExportEmployeesModal", () => ({ ExportEmployeesModal: () => null }));
vi.mock("@/components/shared/MobilePageHeader", () => ({
  MobilePageHeader: () => <div />,
}));
vi.mock("@/components/shared/MobileStatStrip", () => ({ MobileStatStrip: () => null }));
vi.mock("@/components/shared/MobileSearchInput", () => ({ MobileSearchInput: () => <div /> }));
vi.mock("@/components/shared/MobilePagination", () => ({ MobilePagination: () => null }));
vi.mock("@/components/employees/MissingBankDetailsSection", () => ({
  MissingBankDetailsSection: () => null,
}));
vi.mock("@/components/employees/EmployeeEmptyStates", () => ({ EmployeeEmptyStates: () => null }));
vi.mock("@/components/ui/searchable-select", () => ({ SearchableSelect: () => null }));
vi.mock("@/components/employees/EmployeeMobileCard", () => ({
  EmployeeMobileCard: ({
    employee,
    onClick,
    action,
  }: {
    employee: { id: number; fullname: string };
    onClick: (e: unknown) => void;
    action?: React.ReactNode;
  }) => (
    <div>
      <button type="button" onClick={() => onClick(employee)}>
        card-{employee.id}
      </button>
      {action}
    </div>
  ),
}));

describe("EmployeesPageMobile partner claim (PAYR-22)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("pool dùng chung có 'Yêu cầu quản lý' → requestEmployeeAccess rồi mở detail; row managed mở thẳng detail", () => {
    render(
      <MemoryRouter>
        <EmployeesPageMobile />
      </MemoryRouter>,
    );

    const claimButtons = screen.getAllByRole("button", { name: /Yêu cầu quản lý/ });
    expect(claimButtons).toHaveLength(1);

    fireEvent.click(claimButtons[0]);
    expect(mocks.mutate).toHaveBeenCalledWith(101, expect.objectContaining({ onSuccess: expect.any(Function) }));

    fireEvent.click(screen.getByText("card-102"));
    expect(mocks.openEmployeeDetails).toHaveBeenCalledWith("102");
    // Managed row must not claim again
    expect(mocks.mutate).toHaveBeenCalledTimes(1);
  });
});
