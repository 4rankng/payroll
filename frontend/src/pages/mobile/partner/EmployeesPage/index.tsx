import { useState, useCallback, useMemo } from "react";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { InfiniteScrollContainer } from "@/components/ui/infinite-scroll-container";
import { EmployeeMobileCard } from "@/components/employees/EmployeeMobileCard";
import { EmployeeEmptyStates } from "@/components/employees/EmployeeEmptyStates";
import { usePartnerEmployeesData } from "@/hooks/partner-employees/usePartnerEmployeesData";
import { useEmployeesSummary } from "@/hooks/api/useEmployees";
import { useEmployeeModals } from "@/hooks/useModalNavigation";
import { useAssignableProjects } from "@/hooks/api/useProjects";
import { useEmployeeInfiniteScroll } from "@/hooks/employees/useEmployeeInfiniteScroll";
import { useEmployeeExport } from "@/hooks/employees/useEmployeeExport";
import { ExportEmployeesModal } from "@/components/modals/ExportEmployeesModal";
import { MissingBankDetailsSection } from "@/components/employees/MissingBankDetailsSection";
import {
  Users,
  UserCheck,
  UserPlus,
  UserX,
  SlidersHorizontal,
  Plus,
  Download,
  X,
} from "lucide-react";
import type { Employee } from "@/types/api/employee.types";

// Generate last 12 months
const getMonthOptions = () => {
  const options: { value: string; label: string }[] = [];
  const now = new Date();
  for (let i = 0; i < 12; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
    const label = `T${date.getMonth() + 1}/${date.getFullYear()}`;
    options.push({ value, label });
  }
  return options;
};

const monthOptions = getMonthOptions();

const EmployeesPageMobile = () => {
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);

  const {
    employees,
    isLoading,
    searchEmployees,
    clearSearch,
    searchTerm,
    projectId,
    filterByProject,
    statusFilter,
    month,
    updateStatusFilter,
    updateMonth,
    clearAllFilters,
  } = usePartnerEmployeesData();

  const { data: employeesSummary, isLoading: summaryLoading } =
    useEmployeesSummary();
  const { data: projectsData } = useAssignableProjects();
  const { exportEmployees, isExporting } = useEmployeeExport();
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const { openEmployeeDetails, openAddEmployee } = useEmployeeModals();

  const infiniteScroll = useEmployeeInfiniteScroll({
    filteredEmployees: employees,
  });

  const handleEmployeeClick = useCallback(
    (employee: Employee) => {
      openEmployeeDetails(employee.id.toString());
    },
    [openEmployeeDetails],
  );

  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (statusFilter) count++;
    if (month) count++;
    if (projectId) count++;
    return count;
  }, [statusFilter, month, projectId]);

  const projects = useMemo(
    () =>
      projectsData?.data?.map((p) => ({
        id: p.id,
        name: p.name,
        code: p.code,
      })) ?? [],
    [projectsData],
  );

  const stats = useMemo(() => {
    if (employeesSummary) {
      return [
        {
          label: "Tổng",
          value: employeesSummary.total_employees,
          icon: Users,
          color: "text-primary",
          bg: "bg-primary/10",
          filter: null as "working" | "unassigned" | null,
        },
        {
          label: "Đang làm",
          value: employeesSummary.total_working_employees,
          icon: UserCheck,
          color: "text-emerald-600",
          bg: "bg-emerald-500/10",
          filter: "working" as const,
        },
        {
          label: "Chưa phân",
          value:
            employeesSummary.total_employees -
            employeesSummary.total_working_employees,
          icon: UserX,
          color: "text-amber-600",
          bg: "bg-amber-500/10",
          filter: "unassigned" as const,
        },
        {
          label: "Tháng này",
          value: employeesSummary.employees_hired_this_month,
          icon: UserPlus,
          color: "text-violet-600",
          bg: "bg-violet-500/10",
          filter: null as "working" | "unassigned" | null,
        },
      ];
    }
    return [];
  }, [employeesSummary]);

  if (isLoading) {
    return (
      <div className="flex flex-col h-full">
        {/* Header skeleton */}
        <div className="px-4 pt-4 pb-3 flex items-center justify-between">
          <Skeleton className="h-7 w-28" />
          <Skeleton className="h-9 w-16" />
        </div>
        {/* Stats skeleton */}
        <div className="px-4 flex gap-2 overflow-x-auto pb-3">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-20 shrink-0 rounded-xl" />
          ))}
        </div>
        {/* Search skeleton */}
        <div className="px-4 pb-3">
          <Skeleton className="h-11 w-full rounded-xl" />
        </div>
        {/* Cards skeleton */}
        <div className="px-4 space-y-3">
          {[...Array(5)].map((_, i) => (
            <Skeleton key={i} className="h-28 w-full rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col min-h-full pb-24">
      {/* ── Header ── */}
      <MobilePageHeader
        title="Nhân viên"
        icon={Users}
        sticky={false}
        bordered={false}
        actions={
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className="h-9 px-3 shrink-0"
              onClick={() => setExportModalOpen(true)}
              disabled={isExporting}
            >
              <Download className="h-4 w-4 mr-1" />
              {isExporting ? "..." : "Xuất"}
            </Button>
            <Button
              size="sm"
              className="h-9 px-4 btn-partner-primary shrink-0"
              onClick={() => openAddEmployee()}
            >
              <Plus className="h-4 w-4 mr-1" />
              Thêm
            </Button>
          </div>
        }
      />

      {/* ── Stats strip ── */}
      {!summaryLoading && stats.length > 0 && (
        <div className="px-4 pb-3">
          <div className="flex gap-2 overflow-x-auto scrollbar-none">
            {stats.map((stat) => {
              const Icon = stat.icon;
              const isActive =
                stat.filter !== null && statusFilter === stat.filter;
              return (
                <button
                  key={stat.label}
                  onClick={() => {
                    if (stat.filter === null) return;
                    if (isActive) {
                      updateStatusFilter(undefined);
                    } else {
                      updateStatusFilter(stat.filter);
                    }
                  }}
                  className={`flex flex-col items-center gap-1 px-2.5 py-2 rounded-2xl border shrink-0 min-w-[68px] transition-all duration-200 card-lift ${
                    isActive
                      ? "border-primary/40 bg-primary/5"
                      : "border-border bg-card"
                  } ${stat.filter !== null ? "active:scale-95" : "cursor-default"}`}
                >
                  <div className={`p-1.5 rounded-xl ${stat.bg}`}>
                    <Icon className={`h-3 w-3 ${stat.color}`} />
                  </div>
                  <span
                    className={`text-xs font-bold tabular-nums leading-none ${isActive ? "text-primary" : "text-foreground"}`}
                  >
                    {stat.value.toLocaleString("vi-VN")}
                  </span>
                  <span
                    className={`text-[9px] leading-none ${isActive ? "text-primary/70" : "text-muted-foreground"}`}
                  >
                    {stat.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      )}
      {summaryLoading && (
        <div className="px-4 pb-3 flex gap-2">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-[72px] shrink-0 rounded-xl" />
          ))}
        </div>
      )}

      {/* ── Search + Filter ── */}
      <div className="px-4 pb-3 flex gap-2">
        <MobileSearchInput
          value={searchTerm}
          onSearch={searchEmployees}
          placeholder="Tìm kiếm nhân viên..."
          className="w-full"
        />
        <Button
          variant="outline"
          size="icon"
          className="h-11 w-11 rounded-2xl border-border bg-card shrink-0 relative card-lift"
          onClick={() => setFilterSheetOpen(true)}
          aria-label="Bộ lọc"
        >
          <SlidersHorizontal className="h-4 w-4" />
          {activeFilterCount > 0 && (
            <span className="absolute -top-1 -right-1 h-4 w-4 rounded-full bg-primary text-[10px] text-white flex items-center justify-center font-bold">
              {activeFilterCount}
            </span>
          )}
        </Button>
      </div>

      {/* ── Active filter chips ── */}
      {activeFilterCount > 0 && (
        <div className="px-4 pb-3 flex gap-2 flex-wrap items-center">
          {statusFilter && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer rounded-xl"
              onClick={() => updateStatusFilter(undefined)}
            >
              {statusFilter === "working" ? "Đang làm việc" : "Chưa phân công"}
              <X className="h-3 w-3" />
            </Badge>
          )}
          {month && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer rounded-xl"
              onClick={() => updateMonth(undefined)}
            >
              {monthOptions.find((m) => m.value === month)?.label ?? month}
              <X className="h-3 w-3" />
            </Badge>
          )}
          {projectId && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer rounded-xl"
              onClick={() => filterByProject(null)}
            >
              {projects.find((p) => p.id === projectId)?.name ?? "Dự án"}
              <X className="h-3 w-3" />
            </Badge>
          )}
          <button
            onClick={clearAllFilters}
            className="text-xs text-muted-foreground underline underline-offset-2"
          >
            Xóa tất cả
          </button>
        </div>
      )}

      {/* ── Employee list ── */}
      <div className="flex-1 px-4">
        <MissingBankDetailsSection onEmployeeClick={handleEmployeeClick} />
        {employees.length === 0 ? (
          <EmployeeEmptyStates
            searchTerm={searchTerm}
            onClearSearch={() => searchEmployees("")}
            onAddEmployee={() => openAddEmployee()}
          />
        ) : (
          <InfiniteScrollContainer
            onLoadMore={infiniteScroll.loadMore}
            hasMore={infiniteScroll.hasMore}
            isLoading={infiniteScroll.isLoadingMore}
          >
            <div className="space-y-2">
              {infiniteScroll.dataToDisplay.map((employee) => (
                <EmployeeMobileCard
                  key={employee.id}
                  employee={employee}
                  onClick={handleEmployeeClick}
                />
              ))}
            </div>
          </InfiniteScrollContainer>
        )}
      </div>

      {/* ── Filter sheet ── */}
      <Sheet open={filterSheetOpen} onOpenChange={setFilterSheetOpen}>
        <SheetContent side="bottom" className="rounded-t-2xl px-5 pt-2 pb-6">
          <div className="mx-auto mt-2 mb-1 h-1 w-10 rounded-full bg-muted-foreground/20" />
          <SheetHeader className="pb-3">
            <SheetTitle className="text-base font-semibold text-center">Bộ lọc</SheetTitle>
          </SheetHeader>

          <div className="space-y-4">
            {/* Status */}
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Trạng thái
              </label>
              <Select
                value={statusFilter ?? "all"}
                onValueChange={(v) =>
                  updateStatusFilter(
                    v === "all" ? undefined : (v as "working" | "unassigned"),
                  )
                }
              >
                <SelectTrigger className="h-11 rounded-xl">
                  <SelectValue placeholder="Tất cả trạng thái" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả trạng thái</SelectItem>
                  <SelectItem value="working">Đang làm việc</SelectItem>
                  <SelectItem value="unassigned">Chưa phân công</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Month */}
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Tháng
              </label>
              <Select
                value={month ?? "all"}
                onValueChange={(v) =>
                  updateMonth(v === "all" ? undefined : v)
                }
              >
                <SelectTrigger className="h-11 rounded-xl">
                  <SelectValue placeholder="Tất cả tháng" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả tháng</SelectItem>
                  {monthOptions.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Project */}
            {projects.length > 0 && (
              <div className="space-y-1.5">
                <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  Dự án
                </label>
                <Select
                  value={projectId ? projectId.toString() : "all"}
                  onValueChange={(v) =>
                    filterByProject(v === "all" ? null : parseInt(v))
                  }
                >
                  <SelectTrigger className="h-11 rounded-xl">
                    <SelectValue placeholder="Tất cả dự án" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Tất cả dự án</SelectItem>
                    {projects.map((p) => (
                      <SelectItem key={p.id} value={p.id.toString()}>
                        {p.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            {/* Actions */}
            <div className="flex gap-3 pt-4 pb-safe">
              <Button
                variant="outline"
                className="flex-1 h-11 rounded-xl"
                onClick={() => {
                  clearAllFilters();
                  setFilterSheetOpen(false);
                }}
              >
                Xóa bộ lọc
              </Button>
              <Button
                className="flex-1 h-11 rounded-xl btn-partner-primary"
                onClick={() => setFilterSheetOpen(false)}
              >
                Áp dụng
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>

      <ExportEmployeesModal
        open={exportModalOpen}
        onClose={() => setExportModalOpen(false)}
        onExport={(projectIds) => exportEmployees(projectIds)}
        projects={projects}
        isExporting={isExporting}
      />
    </div>
  );
};

export default EmployeesPageMobile;
