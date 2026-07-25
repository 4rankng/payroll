import { render, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import EmployeesPageMobile from "./index";

const updateFilters = vi.fn();

vi.mock("@/hooks/employees/useEmployeeDataInfinite", () => ({
  useEmployeeDataInfinite: () => ({
    employees: [],
    loading: false,
    searchEmployees: vi.fn(),
    searchTerm: "",
    updateFilters,
    clearFilters: vi.fn(),
    currentFilters: {},
    pagination: {
      page: 1,
      pageSize: 20,
      totalPages: 1,
      totalRecords: 0,
    },
    hasMore: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
}));

vi.mock("@/hooks/api/useEmployees", () => ({
  useEmployeesSummary: () => ({ data: undefined, isLoading: false }),
}));

vi.mock("@/hooks/api/useProjects", () => ({
  useAssignableProjects: () => ({ data: { data: [] } }),
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

vi.mock("@/components/employees/EmployeeEmptyStates", () => ({
  EmployeeEmptyStates: () => <div>Không có nhân viên</div>,
}));

vi.mock("@/components/employees/MissingBankDetailsSection", () => ({
  MissingBankDetailsSection: () => null,
}));

vi.mock("@/components/modals/ExportEmployeesModal", () => ({
  ExportEmployeesModal: () => null,
}));

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location-search">{location.search}</output>;
}

describe("EmployeesPageMobile deep links", () => {
  beforeEach(() => {
    updateFilters.mockReset();
  });

  it("opens the add modal and still applies a status filter from one URL", async () => {
    const { getByTestId } = render(
      <MemoryRouter
        initialEntries={["/admin/employees?action=add&status=working"]}
      >
        <Routes>
          <Route
            path="/admin/employees"
            element={
              <>
                <EmployeesPageMobile />
                <LocationProbe />
              </>
            }
          />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(updateFilters).toHaveBeenCalledWith({ status: "working" });
      expect(getByTestId("location-search")).toHaveTextContent(
        "?modal=add_employee",
      );
    });
  });
});
