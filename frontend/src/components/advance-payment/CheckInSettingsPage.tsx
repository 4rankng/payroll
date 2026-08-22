import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  Clock3,
  Loader2,
  Power,
  PowerOff,
  RefreshCw,
  ScanFace,
  SearchX,
  Users,
} from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { SearchBar } from "@/components/shared/SearchBar";
import {
  useCheckInConfigurableProjects,
  useCheckInConfiguration,
  useDisableInactiveCheckInEmployees,
  useToggleCheckInEnabled,
} from "@/hooks/api/useProjectEmployees";
import type {
  CheckInConfigurationEmployee,
  CheckInConfigurationStatus,
} from "@/types/api/project-employee.types";
import { cn } from "@/lib/utils";

const PAGE_SIZE = 50;

function formatMonth(month?: string) {
  if (!month) return "tháng hiện tại";
  const [year, monthNumber] = month.split("-");
  return `tháng ${monthNumber}/${year}`;
}

function formatCheckInTime(value?: string | null) {
  if (!value) return "Chưa có";
  const normalized = value.includes("T") ? value : `${value.replace(" ", "T")}+07:00`;
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: "Asia/Ho_Chi_Minh",
  }).format(parsed);
}

function getEmployeeStatus(employee: CheckInConfigurationEmployee) {
  if (employee.pending_check_in_enable) {
    return { label: "Chờ kích hoạt", icon: Clock3, badge: "outline" as const };
  }
  if (!employee.check_in_enabled) {
    return { label: "Đang tắt", icon: PowerOff, badge: "secondary" as const };
  }
  if (employee.attendance_count > 0) {
    return { label: "Active", icon: CheckCircle2, badge: "default" as const };
  }
  return { label: "Inactive", icon: Power, badge: "outline" as const };
}

interface EmployeeStatusBadgeProps {
  employee: CheckInConfigurationEmployee;
}

function EmployeeStatusBadge({ employee }: EmployeeStatusBadgeProps) {
  const status = getEmployeeStatus(employee);
  const Icon = status.icon;
  return (
    <Badge variant={status.badge} className="gap-1 whitespace-nowrap">
      <Icon className="h-3.5 w-3.5" aria-hidden />
      {status.label}
    </Badge>
  );
}

interface EmployeeActionProps {
  employee: CheckInConfigurationEmployee;
  disabled: boolean;
  pending: boolean;
  onToggle: (employee: CheckInConfigurationEmployee) => void;
}

function EmployeeAction({ employee, disabled, pending, onToggle }: EmployeeActionProps) {
  const isEnabledOrPending = employee.check_in_enabled || employee.pending_check_in_enable;
  const label = employee.pending_check_in_enable
    ? "Hủy chờ"
    : employee.check_in_enabled
      ? "Tắt"
      : "Bật";

  return (
    <Button
      type="button"
      variant={isEnabledOrPending ? "outline" : "default"}
      size="sm"
      className="min-h-11 min-w-20"
      disabled={disabled}
      onClick={() => onToggle(employee)}
      aria-label={`${label} điểm danh cho ${employee.employee_name}`}
    >
      {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
      {label}
    </Button>
  );
}

export default function CheckInSettingsPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const returnPath = location.pathname.startsWith("/adv-partner/")
    ? "/adv-partner/advance-payments"
    : "/admin/advance-payments";

  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null);
  const [status, setStatus] = useState<CheckInConfigurationStatus>("enabled");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [pendingEmployeeId, setPendingEmployeeId] = useState<number | null>(null);
  const [confirmBulkDisable, setConfirmBulkDisable] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const projectsQuery = useCheckInConfigurableProjects();
  const flexibleProjects = useMemo(
    () => (projectsQuery.data ?? []).filter(
      (project) => project.is_flexible && project.status === "active",
    ),
    [projectsQuery.data],
  );

  useEffect(() => {
    if (flexibleProjects.length === 0) return;
    if (flexibleProjects.some((project) => project.id === selectedProjectId)) return;
    setSelectedProjectId(flexibleProjects[0].id);
  }, [flexibleProjects, selectedProjectId]);

  const queryParams = useMemo(() => {
    const trimmedSearch = search.trim();
    return {
      page,
      pageSize: PAGE_SIZE,
      status,
      ...(trimmedSearch ? { search: trimmedSearch } : {}),
    };
  }, [page, search, status]);

  const configurationQuery = useCheckInConfiguration(
    selectedProjectId ?? 0,
    queryParams,
    Boolean(selectedProjectId),
  );
  const toggleMutation = useToggleCheckInEnabled();
  const disableInactiveMutation = useDisableInactiveCheckInEmployees();

  const isSwitchingProject = configurationQuery.isPlaceholderData;
  const configuration = isSwitchingProject ? undefined : configurationQuery.data;
  const summary = configuration?.summary;
  const employees = configuration?.employees ?? [];
  const selectedProject = flexibleProjects.find(
    (project) => project.id === selectedProjectId,
  );
  const monthLabel = formatMonth(configuration?.month);
  const isMutating = toggleMutation.isPending || disableInactiveMutation.isPending;

  const tabs: Array<{
    value: CheckInConfigurationStatus;
    label: string;
    count?: number;
    description: string;
  }> = [
    { value: "all", label: "Tất cả", description: "Toàn bộ nhân viên hiện tại" },
    {
      value: "enabled",
      label: "Đang bật",
      count: summary?.enabled,
      description: "Có quyền sử dụng điểm danh",
    },
    {
      value: "active",
      label: "Active",
      count: summary?.active,
      description: "Đã điểm danh trong tháng",
    },
    {
      value: "inactive",
      label: "Inactive",
      count: summary?.inactive,
      description: "Đang bật nhưng chưa điểm danh",
    },
    {
      value: "pending",
      label: "Chờ bật",
      count: summary?.pending,
      description: "Kích hoạt vào đầu tháng tới",
    },
  ];

  const handleStatusChange = (value: string) => {
    setStatus(value as CheckInConfigurationStatus);
    setPage(1);
    setActionError(null);
  };

  const handleProjectChange = (value: string) => {
    setSelectedProjectId(Number(value));
    setStatus("enabled");
    setSearch("");
    setPage(1);
    setActionError(null);
  };

  const handleSearchChange = (value: string) => {
    setSearch(value);
    setPage(1);
  };

  const handleToggleEmployee = (employee: CheckInConfigurationEmployee) => {
    if (!selectedProjectId || isMutating) return;
    const enabled = employee.check_in_enabled || employee.pending_check_in_enable
      ? false
      : true;
    setActionError(null);
    setPendingEmployeeId(employee.employee_id);
    toggleMutation.mutate(
      { projectId: selectedProjectId, employeeId: employee.employee_id, enabled },
      {
        onError: (error: unknown) => {
          const apiError = error as {
            response?: { data?: { message?: string } };
            message?: string;
          };
          setActionError(
            apiError.response?.data?.message ||
              apiError.message ||
              "Không thể cập nhật điểm danh. Vui lòng thử lại.",
          );
        },
        onSettled: () => setPendingEmployeeId(null),
      },
    );
  };

  const handleDisableInactive = () => {
    if (!selectedProjectId) return;
    setActionError(null);
    disableInactiveMutation.mutate(
      { projectId: selectedProjectId },
      {
        onSuccess: () => setConfirmBulkDisable(false),
        onError: (error: unknown) => {
          const apiError = error as {
            response?: { data?: { message?: string } };
            message?: string;
          };
          setConfirmBulkDisable(false);
          setActionError(
            apiError.response?.data?.message ||
              apiError.message ||
              "Không thể tắt điểm danh cho nhóm Inactive.",
          );
        },
      },
    );
  };

  return (
    <main className="mx-auto w-full max-w-[1440px] space-y-5 p-3 pb-24 sm:p-5 lg:p-8">
      <header className="flex flex-col gap-4 border-b border-border pb-5 sm:flex-row sm:items-end sm:justify-between">
        <div className="min-w-0 space-y-3">
          <Button
            type="button"
            variant="ghost"
            className="min-h-11 w-fit gap-2 px-2"
            onClick={() => navigate(returnPath)}
            aria-label="Quay lại"
          >
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Quay lại
          </Button>
          <div className="flex items-start gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <ScanFace className="h-5 w-5" aria-hidden />
            </div>
            <div className="min-w-0">
              <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
                Cấu hình điểm danh
              </h1>
              <p className="mt-1 max-w-2xl text-sm leading-6 text-muted-foreground">
                Theo dõi mức sử dụng trong tháng và quản lý quyền điểm danh theo dự án.
              </p>
            </div>
          </div>
        </div>

        <div className="w-full min-w-0 space-y-1.5 sm:w-80">
          <label className="text-sm font-medium text-foreground" htmlFor="check-in-project">
            Dự án
          </label>
          <Select
            value={selectedProjectId ? String(selectedProjectId) : ""}
            onValueChange={handleProjectChange}
          >
            <SelectTrigger id="check-in-project" className="h-11 w-full">
              <SelectValue placeholder="Chọn dự án..." />
            </SelectTrigger>
            <SelectContent>
              {flexibleProjects.map((project) => (
                <SelectItem key={project.id} value={String(project.id)}>
                  {project.name}{project.code ? ` (${project.code})` : ""}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </header>

      {projectsQuery.isLoading ? (
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-20 rounded-xl" />
          ))}
        </div>
      ) : projectsQuery.isError ? (
        <div className="rounded-xl border border-destructive/40 bg-destructive/5 px-4 py-14 text-center">
          <h2 className="font-semibold text-foreground">Không thể tải danh sách dự án</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Kiểm tra kết nối hoặc quyền truy cập rồi thử lại.
          </p>
          <Button
            type="button"
            variant="outline"
            className="mt-4 min-h-11"
            onClick={() => projectsQuery.refetch()}
          >
            Thử lại
          </Button>
        </div>
      ) : flexibleProjects.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border px-4 py-14 text-center">
          <Users className="mx-auto h-8 w-8 text-muted-foreground" aria-hidden />
          <h2 className="mt-3 font-semibold text-foreground">Không có dự án linh động</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Chỉ dự án linh động mới sử dụng cấu hình điểm danh.
          </p>
        </div>
      ) : (
        <>
          <div
            role="tablist"
            aria-label="Lọc trạng thái điểm danh"
            className="grid h-auto w-full grid-cols-2 gap-2 bg-transparent p-0 sm:grid-cols-3 xl:grid-cols-5"
          >
              {tabs.map((tab) => (
                <button
                  key={tab.value}
                  type="button"
                  role="tab"
                  aria-selected={status === tab.value}
                  onClick={() => handleStatusChange(tab.value)}
                  className={cn(
                    "flex min-h-20 min-w-0 flex-col items-start rounded-xl border border-border bg-card px-3 py-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                    status === tab.value && "border-primary bg-primary/5 text-foreground",
                  )}
                >
                  <span className="flex w-full items-center justify-between gap-2">
                    <span className="truncate font-semibold">{tab.label}</span>
                    {tab.count !== undefined ? (
                      <span className="tabular-nums text-base font-semibold">{tab.count}</span>
                    ) : null}
                  </span>
                  <span className="mt-1 line-clamp-2 text-xs font-normal leading-4 text-muted-foreground">
                    {tab.description}
                  </span>
                </button>
              ))}
          </div>

          <section className="space-y-3" aria-label="Danh sách cấu hình điểm danh">
            <div className="flex flex-col gap-3 rounded-xl border border-border bg-card p-3 sm:flex-row sm:items-end sm:justify-between">
              <div className="min-w-0 flex-1 space-y-1.5 sm:max-w-md">
                <label htmlFor="check-in-employee-search" className="text-sm font-medium text-foreground">
                  Tìm kiếm nhân viên
                </label>
                <SearchBar
                  inputId="check-in-employee-search"
                  searchTerm={search}
                  onSearchChange={handleSearchChange}
                  placeholder="Tên, CCCD hoặc mã nhân viên..."
                  className="h-11 w-full rounded-md"
                />
              </div>

              <div className="flex flex-wrap items-center gap-2 sm:justify-end">
                <div className="mr-auto text-sm text-muted-foreground sm:mr-2 sm:text-right">
                  <p className="font-medium text-foreground">{monthLabel}</p>
                  <p className="tabular-nums">
                    {configuration?.pagination.totalRecords ?? 0} nhân viên
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="min-h-11 gap-2"
                  onClick={() => configurationQuery.refetch()}
                  disabled={configurationQuery.isFetching}
                >
                  <RefreshCw
                    className={cn("h-4 w-4", configurationQuery.isFetching && "animate-spin")}
                    aria-hidden
                  />
                  Làm mới
                </Button>
                {status === "inactive" && (summary?.inactive ?? 0) > 0 ? (
                  <Button
                    type="button"
                    variant="destructive"
                    className="min-h-11 gap-2"
                    disabled={isMutating}
                    onClick={() => setConfirmBulkDisable(true)}
                    aria-label={`Tắt ${summary?.inactive ?? 0} nhân viên Inactive`}
                  >
                    <PowerOff className="h-4 w-4" aria-hidden />
                    Tắt {summary?.inactive ?? 0} nhân viên Inactive
                  </Button>
                ) : null}
              </div>
            </div>

            {actionError ? (
              <Alert variant="destructive">
                <AlertDescription>{actionError}</AlertDescription>
              </Alert>
            ) : null}

            {configurationQuery.isError ? (
              <div className="rounded-xl border border-destructive/40 bg-destructive/5 px-4 py-10 text-center">
                <h2 className="font-semibold text-foreground">Không thể tải danh sách</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  Kiểm tra kết nối và thử tải lại dữ liệu.
                </p>
                <Button
                  type="button"
                  variant="outline"
                  className="mt-4 min-h-11"
                  onClick={() => configurationQuery.refetch()}
                >
                  Thử lại
                </Button>
              </div>
            ) : configurationQuery.isLoading || isSwitchingProject ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-20 rounded-xl" />
                ))}
              </div>
            ) : employees.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border px-4 py-14 text-center">
                <SearchX className="mx-auto h-8 w-8 text-muted-foreground" aria-hidden />
                <h2 className="mt-3 font-semibold text-foreground">
                  {search ? "Không tìm thấy nhân viên" : "Không có nhân viên trong nhóm này"}
                </h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  {search ? "Thử tên, CCCD hoặc mã nhân viên khác." : "Chọn nhóm khác để tiếp tục quản lý."}
                </p>
              </div>
            ) : (
              <>
                <div className="hidden overflow-hidden rounded-xl border border-border bg-card md:block">
                  <Table>
                    <TableHeader>
                      <TableRow className="bg-muted/40 hover:bg-muted/40">
                        <TableHead>Nhân viên</TableHead>
                        <TableHead>Trạng thái</TableHead>
                        <TableHead className="text-right">Lượt điểm danh</TableHead>
                        <TableHead>Lần gần nhất</TableHead>
                        <TableHead className="w-28 text-right">Hành động</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {employees.map((employee) => (
                        <TableRow key={employee.assignment_id}>
                          <TableCell>
                            <div className="max-w-md">
                              <p className="font-medium text-foreground">{employee.employee_name}</p>
                              <p className="mt-0.5 text-xs tabular-nums text-muted-foreground">
                                {employee.employee_cccd}
                                {employee.employee_code ? ` · ${employee.employee_code}` : ""}
                              </p>
                            </div>
                          </TableCell>
                          <TableCell><EmployeeStatusBadge employee={employee} /></TableCell>
                          <TableCell className="text-right font-medium tabular-nums">
                            {employee.attendance_count}
                          </TableCell>
                          <TableCell className="text-sm tabular-nums text-muted-foreground">
                            {formatCheckInTime(employee.last_check_in_at)}
                          </TableCell>
                          <TableCell className="text-right">
                            <EmployeeAction
                              employee={employee}
                              disabled={isMutating}
                              pending={pendingEmployeeId === employee.employee_id}
                              onToggle={handleToggleEmployee}
                            />
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>

                <div className="space-y-2 md:hidden">
                  {employees.map((employee) => (
                    <article
                      key={employee.assignment_id}
                      className="rounded-xl border border-border bg-card p-3"
                    >
                      <div className="flex min-w-0 items-start justify-between gap-3">
                        <div className="min-w-0">
                          <h2 className="break-words text-[15px] font-semibold leading-5 text-foreground">
                            {employee.employee_name}
                          </h2>
                          <p className="mt-1 break-all text-xs tabular-nums text-muted-foreground">
                            {employee.employee_cccd}
                            {employee.employee_code ? ` · ${employee.employee_code}` : ""}
                          </p>
                        </div>
                        <EmployeeStatusBadge employee={employee} />
                      </div>
                      <div className="mt-3 grid grid-cols-2 gap-2 border-y border-border py-3 text-sm">
                        <div>
                          <p className="text-xs text-muted-foreground">Lượt điểm danh</p>
                          <p className="mt-0.5 font-semibold tabular-nums text-foreground">
                            {employee.attendance_count}
                          </p>
                        </div>
                        <div className="min-w-0">
                          <p className="text-xs text-muted-foreground">Lần gần nhất</p>
                          <p className="mt-0.5 break-words text-xs tabular-nums text-foreground">
                            {formatCheckInTime(employee.last_check_in_at)}
                          </p>
                        </div>
                      </div>
                      <div className="mt-3 flex justify-end">
                        <EmployeeAction
                          employee={employee}
                          disabled={isMutating}
                          pending={pendingEmployeeId === employee.employee_id}
                          onToggle={handleToggleEmployee}
                        />
                      </div>
                    </article>
                  ))}
                </div>
              </>
            )}

            {(configuration?.pagination.totalPages ?? 0) > 1 ? (
              <nav
                aria-label="Phân trang nhân viên"
                className="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-3"
              >
                <p className="text-sm tabular-nums text-muted-foreground">
                  Trang {configuration?.pagination.page} / {configuration?.pagination.totalPages}
                </p>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="h-11 w-11"
                    aria-label="Trang trước"
                    disabled={page <= 1 || isMutating}
                    onClick={() => setPage((current) => Math.max(1, current - 1))}
                  >
                    <ChevronLeft className="h-4 w-4" aria-hidden />
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="icon"
                    className="h-11 w-11"
                    aria-label="Trang sau"
                    disabled={page >= (configuration?.pagination.totalPages ?? 1) || isMutating}
                    onClick={() => setPage((current) => current + 1)}
                  >
                    <ChevronRight className="h-4 w-4" aria-hidden />
                  </Button>
                </div>
              </nav>
            ) : null}
          </section>
        </>
      )}

      <AlertDialog open={confirmBulkDisable} onOpenChange={setConfirmBulkDisable}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle>
              Tắt điểm danh cho {summary?.inactive ?? 0} nhân viên?
            </AlertDialogTitle>
            <AlertDialogDescription className="leading-6">
              Hệ thống sẽ kiểm tra lại và tắt toàn bộ nhân viên Inactive của dự án{" "}
              <span className="font-medium text-foreground">{selectedProject?.name}</span> trong{" "}
              {monthLabel}. Nhân viên đã phát sinh điểm danh sẽ không bị ảnh hưởng.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="min-h-11">Giữ nguyên</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              className="min-h-11"
              disabled={disableInactiveMutation.isPending}
              onClick={handleDisableInactive}
            >
              {disableInactiveMutation.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
              ) : null}
              Xác nhận tắt
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}
