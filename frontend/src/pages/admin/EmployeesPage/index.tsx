import { useState, useEffect, useMemo, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { useEmployeeData } from "@/hooks/employees/useEmployeeData";
import { TableLoadingSkeleton } from "@/components/ui/loading-states";
import { Skeleton } from "@/components/ui/skeleton";
import { createEmployeeMobileConfig } from "@/config/employee-table-mobile";
import { createEmployeeColumns } from "@/config/employee-table-desktop";
import { InlineStatStrip } from "@/components/shared/InlineStatStrip";
import { useEmployeeStatsConfig } from "@/hooks/useEmployeeStatsConfig";
import { EmployeeFiltersBar } from "@/components/employees/EmployeeFiltersBar";
import { EmployeePageHeader } from "@/components/employees/EmployeePageHeader";
import { EmployeeListContent } from "@/components/employees/EmployeeListContent";
import { useEmployeeModals } from "@/hooks/useModalNavigation";
import { useEmployeeInfiniteScroll } from "@/hooks/employees/useEmployeeInfiniteScroll";
import { formatCurrency } from "@/utils/formatters";
import { Employee, EmployeeFilters } from "@/types/api/employee.types";
import { useEmployeeExport } from "@/hooks/employees/useEmployeeExport";
import { useAssignableProjects } from "@/hooks/api/useProjects";
import { ExportEmployeesModal } from "@/components/modals/ExportEmployeesModal";
import { MissingBankDetailsSection } from "@/components/employees/MissingBankDetailsSection";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { useTableSorting } from "@/utils/sorting";

import type { ColumnDef } from "@tanstack/react-table";

import { EMPLOYEE_SORT_FIELD_MAP } from "./constants";

const EmployeesPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const {
    employees,
    loading,
    searchEmployees,
    searchTerm,
    updateFilters,
    clearFilters,
    currentFilters,
    pagination,
    onPageChange,
    onPageSizeChange,
    onSortChange,
  } = useEmployeeData();

  const { sorting, onSortingChange } = useTableSorting(
    currentFilters.sortBy || "created_at",
    currentFilters.sortOrder || "desc",
    onSortChange,
    EMPLOYEE_SORT_FIELD_MAP,
  );

  const employeeStats = useEmployeeStatsConfig({
    formatCurrency,
    currentFilters,
    updateFilters,
    clearFilters,
  });
  const { exportEmployees, isExporting } = useEmployeeExport();
  const { data: projectsData } = useAssignableProjects();
  const [exportModalOpen, setExportModalOpen] = useState(false);

  const { openEmployeeDetails, openAddEmployee } = useEmployeeModals();

  useEffect(() => {
    const action = searchParams.get("action");
    if (action === "add") {
      const newParams = new URLSearchParams(searchParams);
      newParams.delete("action");
      newParams.set("modal", "add_employee");
      setSearchParams(newParams, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  useEffect(() => {
    const status = searchParams.get("status") as
      | "working"
      | "unassigned"
      | null;

    if (status) {
      const filtersToApply: Partial<EmployeeFilters> = {};
      if (status) filtersToApply.status = status;

      updateFilters(filtersToApply);

      const newParams = new URLSearchParams(searchParams);
      if (status) newParams.delete("status");

      if (newParams.toString() !== searchParams.toString()) {
        setSearchParams(newParams, { replace: true });
      }
    }
  }, [searchParams, updateFilters, setSearchParams]);

  const handleExportEmployees = () => {
    setExportModalOpen(true);
  };

  const handleExportConfirm = useCallback((projectIds?: number[]) => {
    exportEmployees(projectIds);
  }, [exportEmployees]);

  const projectsForExport = useMemo(() =>
    projectsData?.data?.map((p) => ({
      id: p.id,
      name: p.name,
      code: p.code,
    })) ?? [], [projectsData]);

  const handleSearchFocus = () => {
    const searchInput = document.querySelector(
      'input[placeholder*="tìm kiếm"], input[placeholder*="Tìm kiếm"]',
    ) as HTMLInputElement;
    if (searchInput) {
      searchInput.focus();
      searchInput.select();
    }
  };

  const handleOpenEmployeeSheet = useCallback(
    (employee: Employee) => {
      openEmployeeDetails(employee.id.toString());
    },
    [openEmployeeDetails],
  );

  const infiniteScroll = useEmployeeInfiniteScroll({
    filteredEmployees: employees,
  });

  const columns: ColumnDef<Employee>[] = useMemo(
    () =>
      createEmployeeColumns({
        onRowClick: handleOpenEmployeeSheet,
      }),
    [handleOpenEmployeeSheet],
  );

  const mobileConfig = useMemo(
    () =>
      createEmployeeMobileConfig({
        formatCurrency,
        onRowClick: handleOpenEmployeeSheet,
      }),
    [handleOpenEmployeeSheet],
  );

  if (loading) {
    return (
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5 animate-fade-in">
        <div className="space-y-2">
          <Skeleton className="h-7 w-28" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-11 w-full max-w-md rounded-xl" />
        <div className="rounded-lg border bg-background overflow-hidden">
          <div className="border-b px-4 py-3">
            <div className="flex gap-6">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-3 w-20" />
              ))}
            </div>
          </div>
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="border-b px-4 py-3 flex items-center gap-3">
              <div className="h-8 w-8 rounded-full bg-muted animate-pulse" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3.5 w-36" />
                <Skeleton className="h-2.5 w-24" />
              </div>
              <Skeleton className="h-3.5 w-28" />
              <Skeleton className="h-3.5 w-20" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full">
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-4">

        <div className="opacity-0 animate-fade-in-up [animation-delay:50ms] [animation-fill-mode:forwards]">
          <EmployeePageHeader
            totalEmployees={pagination.totalRecords}
            onAddEmployeeClick={() => openAddEmployee()}
            onExportClick={handleExportEmployees}
            onSearchFocus={handleSearchFocus}
            isExporting={isExporting}
          />
        </div>

        <div className="opacity-0 animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:forwards]">
          <InlineStatStrip
            items={employeeStats.statsConfig.map(s => ({
              label: s.title,
              value: s.value,
              highlight: s.isActive,
            }))}
            isLoading={employeeStats.isLoading}
          />
        </div>

        <div className="opacity-0 animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:forwards]">
          <EmployeeFiltersBar
            searchTerm={searchTerm}
            onSearchChange={searchEmployees}
            statusFilter={currentFilters.status}
            onStatusFilterChange={(status) => updateFilters({ status })}
            month={currentFilters.month}
            onMonthChange={(month) => updateFilters({ month })}
            fromDate={currentFilters.fromDate}
            toDate={currentFilters.toDate}
            onDateRangeChange={(from, to) =>
              updateFilters({ fromDate: from, toDate: to })
            }
            projectId={currentFilters.projectId || null}
            onProjectChange={(projectId) =>
              updateFilters({ projectId: projectId || undefined })
            }
            projects={projectsData?.data?.map((p) => ({
              id: p.id,
              name: p.name,
              code: p.code,
            }))}
          />
        </div>

        <MissingBankDetailsSection onEmployeeClick={handleOpenEmployeeSheet} />

        <div className="opacity-0 animate-fade-in-up [animation-delay:200ms] [animation-fill-mode:forwards]">
          <EmployeeListContent
            filteredEmployees={employees}
            dataToDisplay={employees}
            columns={columns}
            mobileFields={mobileConfig.mobileFields}
            rowTitle={mobileConfig.rowTitle}
            rowSubtitle={mobileConfig.rowSubtitle}
            rowActions={[]}
            searchTerm={searchTerm}
            isMobile={false}
            hasMore={false}
            isLoadingMore={false}
            loadMore={() => {}}
            onRowClick={handleOpenEmployeeSheet}
            onClearSearch={() => searchEmployees("")}
            onAddEmployee={() => openAddEmployee()}
            sorting={sorting}
            onSortingChange={onSortingChange}
          />
        </div>

        <PaginationControls
          pagination={pagination}
          onPageChange={onPageChange}
          onPageSizeChange={onPageSizeChange}
        />

      </div>

      <ExportEmployeesModal
        open={exportModalOpen}
        onClose={() => setExportModalOpen(false)}
        onExport={handleExportConfirm}
        projects={projectsForExport}
        isExporting={isExporting}
      />
    </div>
  );
};

export default EmployeesPage;
