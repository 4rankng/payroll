import { useState, useCallback, useMemo, useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
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
import { EmployeeMobileCard } from "@/components/employees/EmployeeMobileCard";
import { EmployeeEmptyStates } from "@/components/employees/EmployeeEmptyStates";
import { MissingBankDetailsSection } from "@/components/employees/MissingBankDetailsSection";
import { useEmployeeDataInfinite } from "@/hooks/employees/useEmployeeDataInfinite";
import { useEmployeesSummary } from "@/hooks/api/useEmployees";
import { useEmployeeModals } from "@/hooks/useModalNavigation";
import { useEmployeeExport } from "@/hooks/employees/useEmployeeExport";
import { useAssignableProjects } from "@/hooks/api/useProjects";
import {
  Users,
  UserCheck,
  UserX,
  UserPlus,
  SlidersHorizontal,
  Plus,
  Download,
  X,
} from "lucide-react";
import type { Employee } from "@/types/api/employee.types";

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
  const [searchParams, setSearchParams] = useSearchParams();
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);

  const {
    employees,
    loading,
    searchEmployees,
    searchTerm: employeeSearchTerm,
    updateFilters,
    clearFilters,
    currentFilters,
    pagination,
    hasMore,
    isFetchingNextPage,
    fetchNextPage,
  } = useEmployeeDataInfinite();

  const { data: employeesSummary, isLoading: summaryLoading } =
    useEmployeesSummary();
  const { data: projectsData } = useAssignableProjects();
  const { exportEmployees, isExporting } = useEmployeeExport();
  const { openEmployeeDetails, openAddEmployee } = useEmployeeModals();

  useEffect(() => {
    const status = searchParams.get("status") as
      | "working"
      | "unassigned"
      | null;
    if (status) {
      updateFilters({ status });
      const p = new URLSearchParams(searchParams);
      p.delete("status");
      setSearchParams(p, { replace: true });
    }
  }, [searchParams, updateFilters, setSearchParams]);

  const handleEmployeeClick = useCallback(
    (employee: Employee) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  const handleLoadMore = useCallback(() => {
    if (hasMore && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasMore, isFetchingNextPage, fetchNextPage]);

  const statusFilter = currentFilters.status as
    | "working"
    | "unassigned"
    | undefined;
  const month = currentFilters.month as string | undefined;
  const projectId = currentFilters.projectId as number | undefined;

  const activeFilterCount = useMemo(() => {
    let c = 0;
    if (statusFilter) c++;
    if (month) c++;
    if (projectId) c++;
    return c;
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
    if (!employeesSummary) return [];
    return [
      {
        label: "Tổng",
        value: employeesSummary.total_employees,
        icon: Users,
        color: "text-blue-600",
        bg: "bg-blue-50",
        filter: null as "working" | "unassigned" | null,
      },
      {
        label: "Đang làm",
        value: employeesSummary.total_working_employees,
        icon: UserCheck,
        color: "text-emerald-600",
        bg: "bg-emerald-50",
        filter: "working" as const,
      },
      {
        label: "Chưa phân",
        value:
          employeesSummary.total_employees -
          employeesSummary.total_working_employees,
        icon: UserX,
        color: "text-amber-600",
        bg: "bg-amber-50",
        filter: "unassigned" as const,
      },
      {
        label: "Tháng này",
        value: employeesSummary.employees_hired_this_month,
        icon: UserPlus,
        color: "text-violet-600",
        bg: "bg-violet-50",
        filter: null as "working" | "unassigned" | null,
      },
    ];
  }, [employeesSummary]);

  if (loading && employees.length === 0) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-7 w-28" />
          <div className="flex gap-2">
            <Skeleton className="h-9 w-20" />
            <Skeleton className="h-9 w-16" />
          </div>
        </div>
        <div className="flex gap-2">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-[72px] shrink-0 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-11 w-full rounded-xl" />
        <div className="space-y-2">
          {[...Array(6)].map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col min-h-full pb-20">
      {/* Header */}
      <div className="px-4 pt-4 pb-3 flex items-center justify-between gap-2">
        <h1 className="text-xl font-bold text-foreground">Nhân viên</h1>
        <div className="flex items-center gap-2 shrink-0">
          <Button
            variant="outline"
            size="sm"
            className="h-9 px-3"
            onClick={() => exportEmployees()}
            disabled={isExporting}
          >
            <Download className="h-3.5 w-3.5" />
          </Button>
          <Button
            size="sm"
            className="h-9 px-3 btn-admin-primary"
            onClick={() => openAddEmployee()}
          >
            <Plus className="h-4 w-4 mr-1" />
            Thêm
          </Button>
        </div>
      </div>

      {/* Stats strip */}
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
                    updateFilters({
                      status: isActive ? undefined : stat.filter,
                    });
                  }}
                  className={`flex flex-col items-center gap-1 px-3 py-2.5 rounded-xl border shrink-0 min-w-[72px] transition-all ${isActive ? "border-primary/40 bg-primary/5" : "border-border bg-card"} ${stat.filter !== null ? "active:scale-95" : "cursor-default"}`}
                >
                  <div className={`p-1 rounded-xl ${stat.bg}`}>
                    <Icon className={`h-3.5 w-3.5 ${stat.color}`} />
                  </div>
                  <span
                    className={`text-sm font-bold tabular-nums leading-none ${isActive ? "text-primary" : "text-foreground"}`}
                  >
                    {stat.value.toLocaleString("vi-VN")}
                  </span>
                  <span
                    className={`text-[10px] leading-none ${isActive ? "text-primary/70" : "text-muted-foreground"}`}
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

      <div className="px-4 pb-3 flex gap-2">
        <MobileSearchInput
          value={employeeSearchTerm}
          onSearch={searchEmployees}
          placeholder="Tìm kiếm nhân viên..."
        />
        <Button
          variant="outline"
          size="icon"
          className="h-11 w-11 rounded-xl border-border bg-card shrink-0 relative"
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

      {/* Active filter chips */}
      {activeFilterCount > 0 && (
        <div className="px-4 pb-3 flex gap-2 flex-wrap">
          {statusFilter && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer"
              onClick={() => updateFilters({ status: undefined })}
            >
              {statusFilter === "working" ? "Đang làm việc" : "Chưa phân công"}
              <X className="h-3 w-3" />
            </Badge>
          )}
          {month && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer"
              onClick={() => updateFilters({ month: undefined })}
            >
              {monthOptions.find((m) => m.value === month)?.label ?? month}
              <X className="h-3 w-3" />
            </Badge>
          )}
          {projectId && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer"
              onClick={() => updateFilters({ projectId: undefined })}
            >
              {projects.find((p) => p.id === projectId)?.name ?? "Dự án"}
              <X className="h-3 w-3" />
            </Badge>
          )}
          <button
            onClick={() => {
              clearFilters();
              searchEmployees("");
            }}
            className="text-xs text-muted-foreground underline underline-offset-2"
          >
            Xóa tất cả
          </button>
        </div>
      )}

      <MissingBankDetailsSection onEmployeeClick={handleEmployeeClick} />

      {/* List */}
      <div className="flex-1 px-4">
        {employees.length === 0 ? (
          <EmployeeEmptyStates
            searchTerm={employeeSearchTerm}
            onClearSearch={() => searchEmployees("")}
            onAddEmployee={() => openAddEmployee()}
          />
        ) : (
          <>
            <div className="space-y-2">
              {employees.map((employee) => (
                <EmployeeMobileCard
                  key={employee.id}
                  employee={employee}
                  onClick={handleEmployeeClick}
                />
              ))}
            </div>
            {hasMore && (
              <div className="flex justify-center py-4">
                <Button
                  variant="outline"
                  size="sm"
                  className="gap-2"
                  onClick={handleLoadMore}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? (
                    <>
                      <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                      Đang tải...
                    </>
                  ) : (
                    "Tải thêm"
                  )}
                </Button>
              </div>
            )}
            {isFetchingNextPage && (
              <div className="flex justify-center py-2">
                <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
              </div>
            )}
            {!hasMore && !isFetchingNextPage && employees.length > 0 && (
              <p className="text-center py-3 text-xs text-muted-foreground">
                {pagination?.totalRecords ?? employees.length} nhân viên
              </p>
            )}
          </>
        )}
      </div>

      {/* Filter sheet */}
      <Sheet open={filterSheetOpen} onOpenChange={setFilterSheetOpen}>
        <SheetContent side="bottom" className="rounded-t-2xl pb-safe">
          <SheetHeader className="pb-4">
            <SheetTitle>Bộ lọc</SheetTitle>
          </SheetHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Trạng thái</label>
              <Select
                value={statusFilter ?? "all"}
                onValueChange={(v) =>
                  updateFilters({
                    status:
                      v === "all" ? undefined : (v as "working" | "unassigned"),
                  })
                }
              >
                <SelectTrigger className="h-11">
                  <SelectValue placeholder="Tất cả trạng thái" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả trạng thái</SelectItem>
                  <SelectItem value="working">Đang làm việc</SelectItem>
                  <SelectItem value="unassigned">Chưa phân công</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Tháng</label>
              <Select
                value={month ?? "all"}
                onValueChange={(v) =>
                  updateFilters({ month: v === "all" ? undefined : v })
                }
              >
                <SelectTrigger className="h-11">
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
            {projects.length > 0 && (
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Dự án</label>
                <Select
                  value={projectId ? projectId.toString() : "all"}
                  onValueChange={(v) =>
                    updateFilters({
                      projectId: v === "all" ? undefined : parseInt(v),
                    })
                  }
                >
                  <SelectTrigger className="h-11">
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
            <div className="flex gap-3 pt-2">
              <Button
                variant="outline"
                className="flex-1 h-11"
                onClick={() => {
                  clearFilters();
                  searchEmployees("");
                  setFilterSheetOpen(false);
                }}
              >
                Xóa bộ lọc
              </Button>
              <Button
                className="flex-1 h-11 btn-admin-primary"
                onClick={() => setFilterSheetOpen(false)}
              >
                Áp dụng
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default EmployeesPageMobile;
