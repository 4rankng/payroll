import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Loader2,
  PowerOff,
  ScanFace,
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
import { Button } from "@/components/ui/button";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { Skeleton } from "@/components/ui/skeleton";
import { SearchBar } from "@/components/shared/SearchBar";
import { EmptyState } from "@/components/shared/EmptyState";
import { CheckInEmployeeCard } from "@/components/advance-payment/CheckInEmployeeCard";
import { CheckInMonthSelector } from "@/components/advance-payment/CheckInMonthSelector";
import {
  useCheckInConfigurableProjects,
  useInfiniteCheckInConfiguration,
  useDisableInactiveCheckInEmployees,
  useDisablePendingCheckInEmployees,
  useToggleCheckInEnabled,
} from "@/hooks/api/useProjectEmployees";
import { useInfiniteScroll } from "@/hooks/use-infinite-scroll";
import { cn } from "@/lib/utils";
import type {
  CheckInConfigurationEmployee,
  CheckInConfigurationStatus,
} from "@/types/api/project-employee.types";
import {
  buildCheckInStatusFilters,
  DEFAULT_CHECK_IN_PAGE_SIZE,
  flattenCheckInConfigurationEmployees,
  formatCheckInMonth,
  getCurrentCheckInMonthValue,
} from "@/utils/checkInSettingsHelpers";

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
  const [selectedMonth, setSelectedMonth] = useState(getCurrentCheckInMonthValue);
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
      pageSize: DEFAULT_CHECK_IN_PAGE_SIZE,
      status,
      month: selectedMonth,
      ...(trimmedSearch ? { search: trimmedSearch } : {}),
    };
  }, [search, selectedMonth, status]);

  const configurationQuery = useInfiniteCheckInConfiguration(
    selectedProjectId ?? 0,
    queryParams,
    Boolean(selectedProjectId),
  );
  const toggleMutation = useToggleCheckInEnabled();
  const disableInactiveMutation = useDisableInactiveCheckInEmployees();
  const disablePendingMutation = useDisablePendingCheckInEmployees();

  const configuration = configurationQuery.data?.pages[0];
  const summary = configuration?.summary;
  const employees = useMemo(
    () => flattenCheckInConfigurationEmployees(configurationQuery.data?.pages),
    [configurationQuery.data?.pages],
  );
  const selectedProject = flexibleProjects.find(
    (project) => project.id === selectedProjectId,
  );
  const monthLabel = formatCheckInMonth(configuration?.month);
  const isMutating =
    toggleMutation.isPending ||
    disableInactiveMutation.isPending ||
    disablePendingMutation.isPending;
  const statusFilters = buildCheckInStatusFilters(summary);
  const isCurrentMonth = selectedMonth === getCurrentCheckInMonthValue();
  const { observerRef } = useInfiniteScroll({
    hasMore: Boolean(configurationQuery.hasNextPage),
    isLoading: configurationQuery.isFetchingNextPage,
    onLoadMore: () => {
      void configurationQuery.fetchNextPage();
    },
    enabled: employees.length > 0,
    rootMargin: "320px",
    threshold: 0.1,
  });

  const handleStatusChange = (value: CheckInConfigurationStatus) => {
    setStatus(value);
    setActionError(null);
  };

  const handleProjectChange = (value: string) => {
    setSelectedProjectId(Number(value));
    setStatus("enabled");
    setSearch("");
    setActionError(null);
  };

  const handleSearchChange = (value: string) => {
    setSearch(value);
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
            <SearchableSelect
              triggerId="check-in-project"
              value={selectedProjectId ? String(selectedProjectId) : ""}
              onChange={handleProjectChange}
              placeholder="Chọn dự án..."
              searchPlaceholder="Tìm dự án..."
              triggerClassName="w-full"
              options={flexibleProjects.map((project) => ({
                value: String(project.id),
                label: `${project.name}${project.code ? ` (${project.code})` : ""}`,
              }))}
            />
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
        <div className="rounded-xl border border-dashed border-border px-4">
          <EmptyState
            title="Không có dự án linh động"
            description="Chỉ dự án linh động mới sử dụng cấu hình điểm danh."
            size="sm"
          />
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
                <CheckInMonthSelector
                  value={selectedMonth}
                  onValueChange={(value) => {
                    setSelectedMonth(value);
                    setActionError(null);
                  }}
                />
                <p
                  className="mr-auto min-w-[5.5rem] text-xs tabular-nums text-muted-foreground lg:mr-1 lg:text-right"
                  aria-live="polite"
                >
                  <span className="block font-semibold text-foreground">
                    {configuration?.pagination.totalRecords ?? 0}
                  </span>
                  kết quả
                </p>
                {isCurrentMonth && status === "inactive" && (summary?.inactive ?? 0) > 0 ? (
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
                {isCurrentMonth && status === "pending" && (summary?.pending ?? 0) > 0 ? (
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

            {configurationQuery.isError && !configuration ? (
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
            ) : configurationQuery.isLoading ? (
              <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4" aria-label="Đang tải nhân viên">
                {Array.from({ length: 6 }).map((_, index) => (
                  <Skeleton key={index} className="h-[10.5rem] rounded-lg" />
                ))}
              </div>
            ) : employees.length === 0 ? (
              <div className="rounded-xl border border-dashed border-border px-4">
                <EmptyState
                  title={search ? "Không tìm thấy nhân viên" : "Không có nhân viên trong nhóm này"}
                  description={search ? "Thử tên, CCCD hoặc mã nhân viên khác." : "Chọn nhóm khác để tiếp tục quản lý."}
                  size="sm"
                />
              </div>
            ) : (
              <div
                role="list"
                aria-label="Nhân viên cấu hình điểm danh"
                className="grid gap-2 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
              >
                {employees.map((employee) => (
                  <CheckInEmployeeCard
                    key={employee.assignment_id}
                    employee={employee}
                    disabled={isMutating}
                    pending={pendingEmployeeId === employee.employee_id}
                    onToggle={handleToggleEmployee}
                  />
                ))}
              </div>
            )}

            {employees.length > 0 ? (
              <div
                ref={observerRef}
                className="flex min-h-11 items-center justify-center border-t border-border pt-3 text-xs text-muted-foreground"
                aria-live="polite"
              >
                {configurationQuery.isFetchNextPageError ? (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="h-11 min-h-11 sm:h-8 sm:min-h-0"
                    aria-label="Thử tải thêm nhân viên"
                    onClick={() => {
                      void configurationQuery.fetchNextPage();
                    }}
                  >
                    Thử tải thêm
                  </Button>
                ) : configurationQuery.isFetchingNextPage ? (
                  <span className="inline-flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
                    Đang tải thêm nhân viên...
                  </span>
                ) : configurationQuery.hasNextPage ? (
                  "Cuộn để tải thêm"
                ) : (
                  `Đã hiển thị tất cả ${employees.length} nhân viên`
                )}
              </div>
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
