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
import { PaginationControls } from "@/components/ui/pagination-controls";
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
  useDisablePendingCheckInEmployees,
  useToggleCheckInEnabled,
} from "@/hooks/api/useProjectEmployees";
import { cn } from "@/lib/utils";
import type {
  CheckInConfigurationEmployee,
  CheckInConfigurationStatus,
} from "@/types/api/project-employee.types";
import {
  buildCheckInStatusFilters,
  DEFAULT_CHECK_IN_PAGE_SIZE,
  formatCheckInMonth,
  formatLastCheckIn,
  getCheckInEmployeeState,
} from "@/utils/checkInSettingsHelpers";

const employeeStatePresentation = {
  pending: { label: "Chờ kích hoạt", icon: Clock3, badge: "warning" as const },
  disabled: { label: "Đang tắt", icon: PowerOff, badge: "secondary" as const },
  used: { label: "Đã điểm danh", icon: CheckCircle2, badge: "success" as const },
  unused: { label: "Chưa điểm danh", icon: Power, badge: "outline" as const },
};

interface EmployeeStatusBadgeProps {
  employee: CheckInConfigurationEmployee;
}

function EmployeeStatusBadge({ employee }: EmployeeStatusBadgeProps) {
  const state = getCheckInEmployeeState(employee);
  const presentation = employeeStatePresentation[state];
  const Icon = presentation.icon;
  return (
    <Badge variant={presentation.badge} className="gap-1 whitespace-nowrap">
      <Icon className="h-3.5 w-3.5" aria-hidden />
      {presentation.label}
    </Badge>
  );
}

interface EmployeeActionProps {
  employee: CheckInConfigurationEmployee;
  disabled: boolean;
  pending: boolean;
  mobile?: boolean;
  onToggle: (employee: CheckInConfigurationEmployee) => void;
}

function EmployeeAction({
  employee,
  disabled,
  pending,
  mobile = false,
  onToggle,
}: EmployeeActionProps) {
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
      className={cn(
        "min-w-0",
        mobile ? "h-11 min-h-11 px-4" : "h-8 min-h-0 px-2.5 text-xs",
      )}
      disabled={disabled}
      onClick={() => onToggle(employee)}
      aria-label={`${label} điểm danh cho ${employee.employee_name}`}
    >
      {pending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
      {label}
    </Button>
  );
}

type BulkAction = "inactive" | "pending" | null;

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
  const [pageSize, setPageSize] = useState(DEFAULT_CHECK_IN_PAGE_SIZE);
  const [pendingEmployeeId, setPendingEmployeeId] = useState<number | null>(null);
  const [bulkAction, setBulkAction] = useState<BulkAction>(null);
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
      pageSize,
      status,
      ...(trimmedSearch ? { search: trimmedSearch } : {}),
    };
  }, [page, pageSize, search, status]);

  const configurationQuery = useCheckInConfiguration(
    selectedProjectId ?? 0,
    queryParams,
    Boolean(selectedProjectId),
  );
  const toggleMutation = useToggleCheckInEnabled();
  const disableInactiveMutation = useDisableInactiveCheckInEmployees();
  const disablePendingMutation = useDisablePendingCheckInEmployees();

  const configuration = configurationQuery.isPlaceholderData
    ? undefined
    : configurationQuery.data;
  const summary = configuration?.summary;
  const employees = configuration?.employees ?? [];
  const selectedProject = flexibleProjects.find(
    (project) => project.id === selectedProjectId,
  );
  const monthLabel = formatCheckInMonth(configuration?.month);
  const isMutating =
    toggleMutation.isPending ||
    disableInactiveMutation.isPending ||
    disablePendingMutation.isPending;
  const statusFilters = buildCheckInStatusFilters(summary);

  useEffect(() => {
    if (!configuration || configurationQuery.isFetching) return;
    const lastPage = Math.max(1, configuration.pagination.totalPages);
    if (page > lastPage) setPage(lastPage);
  }, [configuration, configurationQuery.isFetching, page]);

  const handleStatusChange = (value: CheckInConfigurationStatus) => {
    setStatus(value);
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

  const handlePageSizeChange = (value: number) => {
    setPageSize(value);
    setPage(1);
  };

  const handleToggleEmployee = (employee: CheckInConfigurationEmployee) => {
    if (!selectedProjectId || isMutating) return;
    const enabled = !(employee.check_in_enabled || employee.pending_check_in_enable);
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

  const handleBulkAction = () => {
    if (!selectedProjectId || !bulkAction) return;
    setActionError(null);
    const mutation = bulkAction === "inactive"
      ? disableInactiveMutation
      : disablePendingMutation;
    mutation.mutate(
      { projectId: selectedProjectId },
      {
        onSuccess: () => {
          setBulkAction(null);
          setPage(1);
        },
        onError: (error: unknown) => {
          const apiError = error as {
            response?: { data?: { message?: string } };
            message?: string;
          };
          setBulkAction(null);
          setActionError(
            apiError.response?.data?.message ||
              apiError.message ||
              "Không thể cập nhật nhóm nhân viên. Vui lòng thử lại.",
          );
        },
      },
    );
  };

  const bulkCount = bulkAction === "pending" ? summary?.pending ?? 0 : summary?.inactive ?? 0;
  const bulkPending = bulkAction === "pending"
    ? disablePendingMutation.isPending
    : disableInactiveMutation.isPending;

  return (
    <main className="mx-auto w-full max-w-[1440px] space-y-4 p-3 pb-24 sm:p-4 lg:p-6">
      <header className="border-b border-border pb-4">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div className="min-w-0">
            <Button
              type="button"
              variant="ghost"
              className="mb-2 h-11 min-h-11 w-fit gap-2 px-1.5 text-sm sm:h-9 sm:min-h-0"
              onClick={() => navigate(returnPath)}
              aria-label="Quay lại"
            >
              <ArrowLeft className="h-4 w-4" aria-hidden />
              Quay lại
            </Button>
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                <ScanFace className="h-5 w-5" aria-hidden />
              </div>
              <div className="min-w-0">
                <h1 className="text-2xl font-semibold tracking-tight text-foreground">
                  Cấu hình điểm danh
                </h1>
                <p className="mt-0.5 text-sm text-muted-foreground">
                  Quản lý quyền và theo dõi mức sử dụng theo dự án.
                </p>
              </div>
            </div>
          </div>

          <div className="w-full min-w-0 space-y-1 lg:w-80">
            <label className="text-xs font-medium text-muted-foreground" htmlFor="check-in-project">
              Dự án
            </label>
            <Select
              value={selectedProjectId ? String(selectedProjectId) : ""}
              onValueChange={handleProjectChange}
            >
              <SelectTrigger
                id="check-in-project"
                className="h-11 min-h-11 w-full sm:h-9 sm:min-h-0"
              >
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
        </div>
      </header>

      {projectsQuery.isLoading ? (
        <div className="flex gap-2 overflow-hidden">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-11 w-36 shrink-0 rounded-lg" />
          ))}
        </div>
      ) : projectsQuery.isError ? (
        <div className="rounded-xl border border-destructive/40 bg-destructive/5 px-4 py-12 text-center">
          <h2 className="font-semibold text-foreground">Không thể tải danh sách dự án</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Kiểm tra kết nối hoặc quyền truy cập rồi thử lại.
          </p>
          <Button
            type="button"
            variant="outline"
            className="mt-4 h-11 min-h-11 sm:h-9 sm:min-h-0"
            onClick={() => projectsQuery.refetch()}
          >
            Thử lại
          </Button>
        </div>
      ) : flexibleProjects.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border px-4 py-12 text-center">
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
            className="flex gap-1.5 overflow-x-auto border-b border-border pb-3"
          >
            {statusFilters.map((filter) => (
              <button
                key={filter.value}
                type="button"
                role="tab"
                aria-selected={status === filter.value}
                onClick={() => handleStatusChange(filter.value)}
                className={cn(
                  "flex h-11 shrink-0 items-center gap-2 rounded-lg border px-3 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:h-9",
                  status === filter.value
                    ? "border-primary bg-primary/5 text-foreground"
                    : "border-transparent text-muted-foreground hover:border-border hover:bg-muted/50 hover:text-foreground",
                )}
              >
                <span>{filter.label}</span>
                {filter.count !== undefined ? (
                  <span className="rounded-md bg-muted px-1.5 py-0.5 text-xs tabular-nums text-foreground">
                    {filter.count}
                  </span>
                ) : null}
              </button>
            ))}
          </div>

          <section className="space-y-3" aria-label="Danh sách cấu hình điểm danh">
            <div className="flex flex-col gap-3 border-b border-border pb-3 lg:flex-row lg:items-end lg:justify-between">
              <div className="min-w-0 flex-1 space-y-1 lg:max-w-md">
                <label htmlFor="check-in-employee-search" className="text-xs font-medium text-muted-foreground">
                  Tìm kiếm nhân viên
                </label>
                <SearchBar
                  inputId="check-in-employee-search"
                  searchTerm={search}
                  onSearchChange={handleSearchChange}
                  placeholder="Tên, CCCD hoặc mã nhân viên..."
                  className="h-11 w-full rounded-md sm:h-9"
                />
              </div>

              <div className="flex flex-wrap items-center gap-2 lg:justify-end">
                <div className="mr-auto min-w-[7rem] text-sm text-muted-foreground lg:mr-1 lg:text-right">
                  <p className="font-medium text-foreground">{monthLabel}</p>
                  <p className="text-xs tabular-nums">
                    {configuration?.pagination.totalRecords ?? 0} kết quả
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="h-11 min-h-11 gap-1.5 px-3 text-sm sm:h-9 sm:min-h-0"
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
                    size="sm"
                    className="h-11 min-h-11 gap-1.5 px-3 text-sm sm:h-9 sm:min-h-0"
                    disabled={isMutating}
                    onClick={() => setBulkAction("inactive")}
                    aria-label={`Tắt tất cả ${summary?.inactive ?? 0} nhân viên chưa điểm danh`}
                  >
                    <PowerOff className="h-4 w-4" aria-hidden />
                    Tắt tất cả ({summary?.inactive ?? 0})
                  </Button>
                ) : null}
                {status === "pending" && (summary?.pending ?? 0) > 0 ? (
                  <Button
                    type="button"
                    variant="destructive"
                    size="sm"
                    className="h-11 min-h-11 gap-1.5 px-3 text-sm sm:h-9 sm:min-h-0"
                    disabled={isMutating}
                    onClick={() => setBulkAction("pending")}
                    aria-label={`Hủy chờ tất cả ${summary?.pending ?? 0} nhân viên`}
                  >
                    <PowerOff className="h-4 w-4" aria-hidden />
                    Hủy chờ tất cả ({summary?.pending ?? 0})
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
                  className="mt-4 h-11 min-h-11 sm:h-9 sm:min-h-0"
                  onClick={() => configurationQuery.refetch()}
                >
                  Thử lại
                </Button>
              </div>
            ) : configurationQuery.isLoading || configurationQuery.isPlaceholderData ? (
              <div className="space-y-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-16 rounded-lg" />
                ))}
              </div>
            ) : employees.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border px-4 py-12 text-center">
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
                <div className="hidden overflow-hidden rounded-lg border border-border bg-card xl:block">
                  <Table>
                    <TableHeader>
                      <TableRow className="bg-muted/35 hover:bg-muted/35">
                        <TableHead>Nhân viên</TableHead>
                        <TableHead>Trạng thái</TableHead>
                        <TableHead className="text-right">Lượt điểm danh</TableHead>
                        <TableHead>Lần gần nhất</TableHead>
                        <TableHead className="w-24 text-right">Hành động</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {employees.map((employee) => (
                        <TableRow key={employee.assignment_id} className="h-14">
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
                            {formatLastCheckIn(employee.last_check_in_at)}
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

                <div className="divide-y divide-border rounded-lg border border-border bg-card xl:hidden">
                  {employees.map((employee) => (
                    <article key={employee.assignment_id} className="p-3">
                      <div className="flex min-w-0 items-start justify-between gap-3">
                        <div className="min-w-0">
                          <h2 className="break-words text-sm font-semibold leading-5 text-foreground">
                            {employee.employee_name}
                          </h2>
                          <p className="mt-0.5 break-all text-xs tabular-nums text-muted-foreground">
                            {employee.employee_cccd}
                            {employee.employee_code ? ` · ${employee.employee_code}` : ""}
                          </p>
                        </div>
                        <EmployeeStatusBadge employee={employee} />
                      </div>
                      <div className="mt-3 grid grid-cols-[minmax(0,1fr)_auto] items-end gap-3">
                        <dl className="grid min-w-0 grid-cols-2 gap-3 text-sm">
                          <div>
                            <dt className="text-xs text-muted-foreground">Lượt điểm danh</dt>
                            <dd className="mt-0.5 font-semibold tabular-nums text-foreground">
                              {employee.attendance_count}
                            </dd>
                          </div>
                          <div className="min-w-0">
                            <dt className="text-xs text-muted-foreground">Lần gần nhất</dt>
                            <dd className="mt-0.5 break-words text-xs tabular-nums text-foreground">
                              {formatLastCheckIn(employee.last_check_in_at)}
                            </dd>
                          </div>
                        </dl>
                        <EmployeeAction
                          employee={employee}
                          disabled={isMutating}
                          pending={pendingEmployeeId === employee.employee_id}
                          mobile
                          onToggle={handleToggleEmployee}
                        />
                      </div>
                    </article>
                  ))}
                </div>
              </>
            )}

            {configuration ? (
              <>
                <div className="hidden border-t border-border xl:block">
                  <PaginationControls
                    pagination={configuration.pagination}
                    onPageChange={setPage}
                    onPageSizeChange={handlePageSizeChange}
                  />
                </div>
                {configuration.pagination.totalPages > 1 ? (
                  <nav
                    aria-label="Phân trang nhân viên"
                    className="flex items-center justify-between border-t border-border pt-3 xl:hidden"
                  >
                    <p className="text-xs tabular-nums text-muted-foreground">
                      {(configuration.pagination.page - 1) * configuration.pagination.pageSize + 1}–
                      {Math.min(
                        configuration.pagination.page * configuration.pagination.pageSize,
                        configuration.pagination.totalRecords,
                      )} / {configuration.pagination.totalRecords}
                    </p>
                    <div className="flex items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        className="h-11 w-11"
                        aria-label="Trang trước trên thiết bị nhỏ"
                        disabled={page <= 1 || isMutating}
                        onClick={() => setPage((current) => Math.max(1, current - 1))}
                      >
                        <ChevronLeft className="h-4 w-4" aria-hidden />
                      </Button>
                      <span className="min-w-12 text-center text-xs tabular-nums text-muted-foreground">
                        {configuration.pagination.page}/{configuration.pagination.totalPages}
                      </span>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        className="h-11 w-11"
                        aria-label="Trang sau"
                        disabled={page >= configuration.pagination.totalPages || isMutating}
                        onClick={() => setPage((current) => current + 1)}
                      >
                        <ChevronRight className="h-4 w-4" aria-hidden />
                      </Button>
                    </div>
                  </nav>
                ) : null}
              </>
            ) : null}
          </section>
        </>
      )}

      <AlertDialog open={bulkAction !== null} onOpenChange={(open) => !open && setBulkAction(null)}>
        <AlertDialogContent className="max-w-md">
          <AlertDialogHeader>
            <AlertDialogTitle>
              {bulkAction === "pending"
                ? `Hủy kích hoạt đang chờ của ${bulkCount} nhân viên?`
                : `Tắt điểm danh cho ${bulkCount} nhân viên?`}
            </AlertDialogTitle>
            <AlertDialogDescription className="leading-6">
              {bulkAction === "pending" ? (
                <>
                  Hệ thống sẽ kiểm tra lại và hủy toàn bộ yêu cầu kích hoạt đang chờ của dự án{" "}
                  <span className="font-medium text-foreground">{selectedProject?.name}</span>. Thao tác
                  áp dụng cho toàn bộ kết quả, không chỉ trang đang xem.
                </>
              ) : (
                <>
                  Hệ thống sẽ kiểm tra lại và tắt toàn bộ nhân viên chưa điểm danh của dự án{" "}
                  <span className="font-medium text-foreground">{selectedProject?.name}</span> trong{" "}
                  {monthLabel}. Nhân viên đã phát sinh điểm danh sẽ không bị ảnh hưởng.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="min-h-11">Giữ nguyên</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              className="min-h-11"
              disabled={bulkPending}
              onClick={handleBulkAction}
            >
              {bulkPending ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
              {bulkAction === "pending" ? "Xác nhận hủy chờ" : "Xác nhận tắt"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}
