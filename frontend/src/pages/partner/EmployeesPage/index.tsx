import { format } from "date-fns";
import { useState, useMemo, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { PageHeader } from "@/components/shared/PageHeader";
import { InlineStatStrip } from "@/components/shared/InlineStatStrip";
import { SearchBar } from "@/components/shared/SearchBar";
import { FilterPill } from "@/components/shared/FilterPill";
import { usePartnerEmployeesData } from "@/hooks/partner-employees/usePartnerEmployeesData";
import { useEmployeesSummary } from "@/hooks/api/useEmployees";
import { useEmployeeModals } from "@/hooks/useModalNavigation";
import { useEmployeeExport } from "@/hooks/employees/useEmployeeExport";
import { useAssignableProjects } from "@/hooks/api/useProjects";
import { ExportEmployeesModal } from "@/components/modals/ExportEmployeesModal";
import { MissingBankDetailsSection } from "@/components/employees/MissingBankDetailsSection";
import { useTableSorting } from "@/utils/sorting";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";
import {
  Users,
  Plus,
  Download,
  Landmark,
  CreditCard,
  Phone,
  Clock,
  X,
  UserPlus,
} from "lucide-react";
import {
  Employee,
  getEmployeeProjects,
  getEmployeeProjectCount,
  hasCompleteBankDetails,
} from "@/types/api/employee.types";
import { generateMonthOptions } from "@/utils/dateHelpers";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { ColumnDef } from "@tanstack/react-table";

const SCHEDULE_STYLES: Record<string, string> = {
  weekly: "bg-sky-50 text-sky-700 border-sky-200/80",
  monthly: "bg-violet-50 text-violet-700 border-violet-200/80",
  flexible: "bg-emerald-50 text-emerald-700 border-emerald-200/80",
};

const SCHEDULE_LABELS: Record<string, string> = {
  weekly: "lương tuần",
  monthly: "lương tháng",
  flexible: "linh động",
};

const monthOptions = generateMonthOptions(12).map(o => ({
  value: o.value,
  label: `T${o.value.split("-")[1]}/${o.value.split("-")[0]}`,
}));

const EmployeesPage = () => {
  const navigate = useNavigate();
  const {
    employees,
    pagination,
    isLoading,
    searchEmployees,
    searchTerm,
    projectId,
    filterByProject,
    statusFilter,
    month,
    updateStatusFilter,
    updateMonth,
    clearAllFilters,
    handlePageChange,
    handlePageSizeChange,
    sortBy,
    sortOrder,
    handleSortChange,
  } = usePartnerEmployeesData();

  const { sorting, onSortingChange } = useTableSorting(
    sortBy,
    sortOrder,
    handleSortChange,
  );

  const { data: employeesSummary, isLoading: summaryLoading } =
    useEmployeesSummary();
  const { data: projectsData } = useAssignableProjects();
  const { exportEmployees, isExporting } = useEmployeeExport();
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const { openEmployeeDetails, openAddEmployee, openTimesheetEntry } = useEmployeeModals();

  const handleEmployeeClick = useCallback(
    (employee: Employee) => openEmployeeDetails(employee.id.toString()),
    [openEmployeeDetails],
  );

  const activeEmployees = useMemo(
    () => employees.filter((emp) => emp.status === "active").length,
    [employees],
  );

  const projects = useMemo(
    () =>
      projectsData?.data?.map((p) => ({
        id: p.id,
        name: p.name,
        code: p.code,
      })) ?? [],
    [projectsData],
  );

  const columns: ColumnDef<Employee>[] = useMemo(
    () => [
      {
        accessorKey: "fullname",
        header: "Nhân viên",
        size: 0,
        minSize: 0,
        cell: ({ row }) => {
          const employee = row.original;
          const pendingCount = employee.timesheet_summary?.pending_timesheets ?? 0;
          const bankOk = hasCompleteBankDetails(employee);
          return (
            <div
              className="flex items-center gap-3 cursor-pointer -mx-2 px-2 py-1.5 rounded-lg transition-colors hover:bg-accent/30 min-w-0 group"
              onClick={(e) => {
                e.stopPropagation();
                handleEmployeeClick(employee);
              }}
            >
              <UserAvatar name={employee.fullname} email={employee.email} cccd={employee.cccd} size="md" />
              <div className="min-w-0 flex-1 space-y-0.5">
                <div className="flex items-center gap-2">
                  <span className="typography-body-medium font-semibold text-foreground truncate group-hover:text-primary transition-colors">
                    {employee.fullname}
                  </span>
                  {pendingCount > 0 && (
                    <TooltipProvider delayDuration={200}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              navigate(`/partner/timesheet?employee=${employee.id}`);
                            }}
                            className="inline-flex items-center gap-0.5 shrink-0 text-[10px] font-semibold text-amber-700 bg-amber-50 border border-amber-200/80 rounded px-1.5 py-0.5 hover:bg-amber-100 transition-colors cursor-pointer"
                          >
                            <Clock className="h-2.5 w-2.5" />
                            {pendingCount}
                          </button>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs">
                          {pendingCount} bảng công chờ duyệt
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )}
                  {!bankOk && (
                    <TooltipProvider delayDuration={200}>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="inline-flex items-center justify-center h-4 w-4 rounded bg-amber-50 border border-amber-200/60">
                            <X className="h-2.5 w-2.5 text-amber-500" />
                          </span>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="text-xs">
                          Thiếu thông tin ngân hàng
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  )}
                </div>
                <div className="flex items-center gap-2.5 text-muted-foreground/60">
                  <div className="flex items-center gap-1">
                    <CreditCard className="h-3 w-3 shrink-0" />
                    <span className="typography-label-medium font-mono truncate">{employee.cccd}</span>
                  </div>
                  {employee.mobile && (
                    <div className="flex items-center gap-1">
                      <Phone className="h-3 w-3 shrink-0" />
                      <span className="typography-label-medium tabular-nums">{employee.mobile}</span>
                    </div>
                  )}
                </div>
              </div>
            </div>
          );
        },
      },
      {
        id: "project_info",
        header: "Dự án",
        size: 0,
        minSize: 0,
        accessorFn: (row) => {
          const projects = getEmployeeProjects(row);
          return projects.length > 0 ? projects[0].name : "";
        },
        cell: ({ row }) => {
          const employee = row.original;
          const projects = getEmployeeProjects(employee);
          const projectCount = getEmployeeProjectCount(employee);

          if (projectCount === 0) {
            return (
              <span className="inline-flex items-center gap-1 text-xs text-muted-foreground/50 px-2 py-1 rounded-md bg-muted/30">
                Chưa phân công
              </span>
            );
          }

          const first = projects[0];
          const more = projectCount - 1;

          return (
            <div className="space-y-1 min-w-0">
              <div className="flex items-center gap-2 min-w-0">
                <span className="typography-body-medium text-foreground truncate">
                  {first.name}
                </span>
                {first.payment_schedule && !first.last_date && (
                  <span className={cn(
                    "inline-flex items-center px-1.5 py-px rounded text-[10px] font-semibold border shrink-0",
                    SCHEDULE_STYLES[first.payment_schedule],
                  )}>
                    {SCHEDULE_LABELS[first.payment_schedule]}
                  </span>
                )}
              </div>
              <div className="flex items-center gap-1.5">
                <span className="typography-label-medium text-muted-foreground/50 font-mono truncate">
                  {first.code}
                  {first.position && ` · ${first.position}`}
                </span>
                {more > 0 && (
                  <TooltipProvider delayDuration={200}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex items-center justify-center h-4 min-w-[1.25rem] px-1 rounded text-[10px] font-bold bg-primary/5 text-primary/60 border border-primary/10 shrink-0">
                          +{more}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs max-w-[200px]">
                        {projects.slice(1).map(p => p.name).join(', ')}
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                )}
              </div>
            </div>
          );
        },
      },
      {
        id: "bank",
        header: "Ngân hàng",
        size: 0,
        accessorFn: (row) => row.bank?.branch_name || "",
        cell: ({ row }) => {
          const employee = row.original;
          const branch = employee.bank?.branch_name;
          if (!branch) return <span className="text-muted-foreground/40">—</span>;
          return (
            <div className="space-y-0.5 min-w-0">
              <div className="flex items-center gap-2">
                <div className="flex h-5 w-5 items-center justify-center rounded bg-primary/5 border border-primary/10 shrink-0">
                  <Landmark className="h-2.5 w-2.5 text-primary/50" />
                </div>
                <span className="typography-body-medium text-foreground/80 truncate" title={branch}>
                  {branch}
                </span>
              </div>
              {employee.bank_account_number && (
                <span className="typography-label-medium text-muted-foreground/50 tabular-nums pl-7 truncate block">
                  {employee.bank_account_number}
                </span>
              )}
            </div>
          );
        },
      },
      {
        accessorKey: "created_at",
        header: "Ngày tạo",
        size: 100,
        cell: ({ row }) => (
          <span className="typography-body-medium text-muted-foreground tabular-nums">
            {row.original.created_at
              ? format(new Date(row.original.created_at), 'dd/MM/yyyy')
              : "—"}
          </span>
        ),
      },
    ],
    [navigate, handleEmployeeClick],
  );

  const mobileConfig = useMemo(
    () => ({
      mobileFields: [
        {
          key: "project_info",
          label: "Dự án",
          render: (row: Employee) => {
            const projects = getEmployeeProjects(row);
            const count = getEmployeeProjectCount(row);
            if (count === 0) return <span className="text-muted-foreground/50">Chưa phân công</span>;
            const p = projects[0];
            return (
              <div className="space-y-0.5">
                <span className="typography-body-medium text-foreground">{p.name}</span>
                {p.payment_schedule && (
                  <span className={cn(
                    "ml-1.5 inline-flex items-center px-1.5 py-px rounded text-[10px] font-semibold border",
                    SCHEDULE_STYLES[p.payment_schedule],
                  )}>
                    {SCHEDULE_LABELS[p.payment_schedule]}
                  </span>
                )}
                {count > 1 && (
                  <span className="ml-1 text-xs text-primary/60 font-bold">+{count - 1}</span>
                )}
              </div>
            );
          },
        },
        {
          key: "bank",
          label: "Ngân hàng",
          render: (row: Employee) => (
            <span className="typography-body-medium text-foreground/80">
              {row.bank?.branch_name || "—"}
            </span>
          ),
        },
      ],
      rowTitle: (row: Employee) => (
        <div className="flex items-center gap-2.5">
          <UserAvatar name={row.fullname} email={row.email} cccd={row.cccd} size="sm" />
          <div className="min-w-0">
            <p className="typography-body-medium font-semibold text-foreground truncate">{row.fullname}</p>
            <div className="flex items-center gap-2 text-muted-foreground/60">
              <div className="flex items-center gap-1">
                <CreditCard className="h-3 w-3" />
                <span className="typography-label-medium font-mono">{row.cccd || "—"}</span>
              </div>
            </div>
          </div>
        </div>
      ),
      rowSubtitle: () => null,
    }),
    [],
  );

  const hasFilters = !!(statusFilter || month || projectId || searchTerm);

  if (isLoading) {
    return (
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5 animate-fade-in">
        <div className="space-y-2">
          <Skeleton className="h-7 w-36" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-11 w-full max-w-md rounded-xl" />
        <div className="rounded-lg border bg-background overflow-hidden">
          <div className="border-b px-4 py-3">
            <div className="flex gap-6">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-3 w-20" />
              ))}
            </div>
          </div>
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="border-b px-4 py-3 flex items-center gap-3">
              <div className="h-8 w-8 rounded-full bg-muted animate-pulse" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3.5 w-36" />
                <Skeleton className="h-2.5 w-28" />
              </div>
              <Skeleton className="h-3.5 w-32" />
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
          <PageHeader
            title="Nhân viên dự án"
            description="Danh sách nhân viên trong các dự án được phân quyền"
            icon={Users}
            actions={[
              {
                label: isExporting ? 'Đang xuất...' : 'Xuất Excel',
                onClick: () => setExportModalOpen(true),
                icon: Download,
                variant: 'outline' as const,
                className: isExporting ? 'opacity-50 pointer-events-none' : '',
              },
              {
                label: 'Thêm',
                onClick: () => openAddEmployee(),
                icon: Plus,
                variant: 'default' as const,
              },
            ]}
          />
        </div>

        <div className="opacity-0 animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:forwards]">
          <InlineStatStrip
            isLoading={summaryLoading}
            items={
              employeesSummary
                ? [
                    {
                      label: "Tổng",
                      value: employeesSummary.total_employees,
                      onClick: () => { updateStatusFilter(undefined); updateMonth(undefined); },
                    },
                    {
                      label: "Chưa phân công",
                      value: employeesSummary.total_employees - employeesSummary.total_working_employees,
                      highlight: (employeesSummary.total_employees - employeesSummary.total_working_employees) > 0,
                      onClick: () => { updateStatusFilter(statusFilter === "unassigned" ? undefined : "unassigned"); updateMonth(undefined); },
                    },
                    {
                      label: "Đang làm việc",
                      value: employeesSummary.total_working_employees,
                      onClick: () => { updateStatusFilter(statusFilter === "working" ? undefined : "working"); updateMonth(undefined); },
                    },
                    {
                      label: "Thêm tháng này",
                      value: employeesSummary.employees_hired_this_month,
                      onClick: () => { updateStatusFilter(undefined); updateMonth(month ? undefined : new Date().toISOString().slice(0, 7)); },
                    },
                  ]
                : [
                    { label: "Tổng", value: employees.length, onClick: () => { updateStatusFilter(undefined); updateMonth(undefined); } },
                    {
                      label: "Chưa phân công",
                      value: employees.length - activeEmployees,
                      highlight: (employees.length - activeEmployees) > 0,
                      onClick: () => { updateStatusFilter(statusFilter === "unassigned" ? undefined : "unassigned"); updateMonth(undefined); },
                    },
                    {
                      label: "Đang làm việc",
                      value: activeEmployees,
                      onClick: () => { updateStatusFilter(statusFilter === "working" ? undefined : "working"); updateMonth(undefined); },
                    },
                    {
                      label: "Thêm tháng này",
                      value: 0,
                      onClick: () => { updateStatusFilter(undefined); updateMonth(month ? undefined : new Date().toISOString().slice(0, 7)); },
                    },
                  ]
            }
          />
        </div>

        <div className="opacity-0 animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:forwards]">
          <div className="flex items-center gap-2 flex-wrap rounded-xl border border-border/50 bg-card/50 backdrop-blur-sm px-3 py-2">
            <SearchBar
              searchTerm={searchTerm}
              onSearchChange={searchEmployees}
              placeholder="Tìm nhân viên..."
              className="w-44"
            />

            <div className="h-5 w-px bg-border/50 shrink-0 hidden sm:block" />

            {projects.length > 0 && (
              <FilterPill
                value={projectId ? projectId.toString() : "all"}
                onChange={(v) => filterByProject(v === "all" ? null : parseInt(v))}
                placeholder="Dự án"
                options={projects.map((p) => ({ value: p.id.toString(), label: p.name }))}
              />
            )}

            <FilterPill
              value={statusFilter ?? "all"}
              onChange={(v) => updateStatusFilter(v === "all" ? undefined : (v as "working" | "unassigned"))}
              placeholder="Trạng thái"
              options={[
                { value: "working", label: "Đang làm việc" },
                { value: "unassigned", label: "Chưa phân công" },
              ]}
            />

            <FilterPill
              value={month ?? "all"}
              onChange={(v) => updateMonth(v === "all" ? undefined : v)}
              placeholder="Tháng"
              options={monthOptions}
            />

            {hasFilters && (
              <button
                onClick={clearAllFilters}
                className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors ml-1 px-2 py-1 rounded-md hover:bg-muted/50"
              >
                <X className="h-3 w-3" />
                Xóa lọc
              </button>
            )}
          </div>
        </div>

        <MissingBankDetailsSection onEmployeeClick={handleEmployeeClick} />

        <div className="opacity-0 animate-fade-in-up [animation-delay:200ms] [animation-fill-mode:forwards]">
          <ResponsiveTable
            data={employees}
            columns={columns}
            mobileFields={mobileConfig.mobileFields}
            rowTitle={mobileConfig.rowTitle}
            rowSubtitle={mobileConfig.rowSubtitle}
            getRowId={(row) => row.id.toString()}
            onRowClick={handleEmployeeClick}
            pagination={pagination}
            onPageChange={handlePageChange}
            onPageSizeChange={handlePageSizeChange}
            sorting={sorting}
            onSortingChange={onSortingChange}
            emptyState={
              <div className="flex flex-col items-center justify-center py-16 px-4">
                <div className="relative mb-6">
                  <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-primary/5 border border-primary/10">
                    <Users className="h-9 w-9 text-primary/30" />
                  </div>
                  <div className="absolute -right-1 -bottom-1 flex h-8 w-8 items-center justify-center rounded-lg bg-background border shadow-sm">
                    <UserPlus className="h-4 w-4 text-muted-foreground" />
                  </div>
                </div>
                <h3 className="typography-headline-small text-foreground mb-1">
                  Không có nhân viên nào
                </h3>
                <p className="typography-body-medium text-muted-foreground max-w-xs text-center">
                  Chưa có nhân viên nào trong các dự án được phân quyền.
                </p>
              </div>
            }
            accordionType="single"
          />
        </div>

      </div>

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

export default EmployeesPage;
