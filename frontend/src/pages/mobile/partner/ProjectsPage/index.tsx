import { useState, useMemo, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { MobilePageShell, MobileSurface } from "@/components/shared/MobilePageShell";
import { Button } from "@/components/ui/button";
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
import { ProjectMobileList } from "@/components/projects/ProjectMobileList";
import { EmptyState } from "@/components/shared/EmptyState";
import { useProjects } from "@/hooks/api/useProjects";
import { useProjectFilters } from "@/hooks/projects/useProjectFilters";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { usePartnerProjectSummary } from "@/hooks/partner/usePartnerProjectSummary";
import { getVietnameseProjectStatus } from "@/utils/vietnamese";
import { generateMonthOptions } from "@/utils/dateHelpers";
import {
  Briefcase,
  CheckCircle,
  Users,
  Plus,
  SlidersHorizontal,
  X,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import type { Project } from "@/types/api/project.types";

const ITEMS_PER_PAGE = 20;

const STATUS_OPTIONS: { value: Project["status"]; label: string }[] = [
  { value: "active", label: getVietnameseProjectStatus("active") },
  { value: "paused", label: getVietnameseProjectStatus("paused") },
  { value: "completed", label: getVietnameseProjectStatus("completed") },
  { value: "draft", label: getVietnameseProjectStatus("draft") },
  { value: "cancelled", label: getVietnameseProjectStatus("cancelled") },
];

const ProjectsPageMobile = () => {
  const navigate = useNavigate();
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);

  const filterControls = useProjectFilters({
    initialFilters: { page: 1, pageSize: ITEMS_PER_PAGE },
  });

  const { data: response, isLoading } = useProjects(filterControls.apiFilters);
  const { data: summary, isLoading: summaryLoading } = usePartnerProjectSummary();
  const projects = response?.data || [];
  const pagination = response?.pagination;
  const currentPage = pagination?.page ?? filterControls.page;
  const totalPages = pagination?.totalPages ?? 1;
  const monthOptions = useMemo(() => generateMonthOptions(), []);

  const { openCreateProject, openPartnerProjectDetails } = useProjectModals();

  const activeFilterCount = useMemo(() => {
    let count = 0;
    if (filterControls.statusFilter !== "all") count++;
    if (filterControls.monthFilter) count++;
    return count;
  }, [filterControls.statusFilter, filterControls.monthFilter]);

  const currentStatusLabel = useMemo(() => {
    if (filterControls.statusFilter === "all" || !Array.isArray(filterControls.statusFilter)) return null;
    return filterControls.statusFilter.map((s) => getVietnameseProjectStatus(s)).join(", ");
  }, [filterControls.statusFilter]);

  const currentMonthLabel = useMemo(
    () => monthOptions.find((m) => m.value === filterControls.monthFilter)?.label ?? null,
    [monthOptions, filterControls.monthFilter],
  );

  const handleProjectClick = useCallback(
    (project: Project) => openPartnerProjectDetails(project.id.toString()),
    [openPartnerProjectDetails],
  );

  if (isLoading && projects.length === 0) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-7 w-24" />
          <Skeleton className="h-9 w-24" />
        </div>
        <div className="flex gap-2">
          {[...Array(3)].map((_, i) => (
            <Skeleton key={i} className="h-14 w-20 shrink-0 rounded-xl" />
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
    <MobilePageShell className="space-y-3">
      {/* ── Header ── */}
      <MobilePageHeader
        title="Dự án"
        icon={Briefcase}
        sticky={false}
        bordered={false}
        actions={
          <Button
            size="sm"
            className="h-11 px-4 btn-partner-primary shrink-0 rounded-xl"
            onClick={() => openCreateProject()}
          >
            <Plus className="h-4 w-4 mr-1" />
            Thêm
          </Button>
        }
      />

      {/* ── Stats strip ── */}
      {!summaryLoading && summary && (
        <MobileSurface className="p-3">
          <div className="grid grid-cols-2 gap-2">
            {[
              { label: "Tổng", value: summary.total_projects, icon: Briefcase, color: "text-primary", bg: "bg-primary/10" },
              { label: "Đang dùng", value: summary.active_projects, icon: CheckCircle, color: "text-success", bg: "bg-success/10" },
              { label: "Hoàn thành", value: summary.completed_projects, icon: CheckCircle, color: "text-info", bg: "bg-info/10" },
              { label: "Nhân viên", value: summary.total_employees, icon: Users, color: "text-warning", bg: "bg-warning/10" },
            ].map((stat) => {
              const Icon = stat.icon;
              return (
                <div
                  key={stat.label}
                  className="flex min-h-16 min-w-0 flex-col items-center gap-1 rounded-2xl border border-border bg-card px-2.5 py-2 transition-all duration-200 active:scale-95"
                >
                  <div className={`p-1.5 rounded-xl ${stat.bg}`}>
                    <Icon className={`h-3 w-3 ${stat.color}`} />
                  </div>
                  <span className="text-xs font-bold tabular-nums leading-none text-foreground">
                    {stat.value.toLocaleString("vi-VN")}
                  </span>
                  <span className="text-[9px] leading-none text-muted-foreground">{stat.label}</span>
                </div>
              );
            })}
          </div>
        </MobileSurface>
      )}
      {summaryLoading && (
        <div className="grid grid-cols-2 gap-2 px-4 pb-3">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
      )}

      {/* ── Search + Filter ── */}
      <MobileSurface className="flex gap-2 p-3">
        <MobileSearchInput
          value={filterControls.searchTerm}
          onSearch={filterControls.setSearchTerm}
          placeholder="Tìm kiếm dự án..."
          className="w-full"
        />
        <Button
          variant="outline"
          size="icon"
          className="h-11 w-11 rounded-2xl border-border bg-card shrink-0 relative shadow-sm"
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
      </MobileSurface>

      {/* ── Active filter chips ── */}
      {activeFilterCount > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          {currentStatusLabel && (
            <button
              type="button"
              className="inline-flex min-h-11 items-center gap-1 rounded-xl bg-secondary px-3 text-xs font-semibold text-secondary-foreground touch-manipulation"
              onClick={() => filterControls.setStatusFilter("all")}
              aria-label={`Xóa lọc trạng thái ${currentStatusLabel}`}
            >
              {currentStatusLabel}
              <X className="h-3 w-3" />
            </button>
          )}
          {currentMonthLabel && (
            <button
              type="button"
              className="inline-flex min-h-11 items-center gap-1 rounded-xl bg-secondary px-3 text-xs font-semibold text-secondary-foreground touch-manipulation"
              onClick={() => filterControls.setMonthFilter(undefined)}
              aria-label={`Xóa lọc tháng ${currentMonthLabel}`}
            >
              {currentMonthLabel}
              <X className="h-3 w-3" />
            </button>
          )}
          <button
            onClick={filterControls.clearFilters}
            className="min-h-11 px-1 text-xs text-muted-foreground underline underline-offset-2"
          >
            Xóa tất cả
          </button>
        </div>
      )}

      {/* ── Project list ── */}
      <div className="flex-1">
        <ProjectMobileList
          projects={projects}
          onRowClick={handleProjectClick}
          onTimesheet={(project) =>
            navigate(`/partner/timesheet?project=${project.id}`)
          }
          emptyState={
            <EmptyState
              title="Không có dự án nào"
              description="Bạn chưa được phân quyền truy cập vào dự án nào."
              size="sm"
            />
          }
        />

        {pagination && pagination.totalRecords > pagination.pageSize && (
          <div className="mt-3 flex items-center justify-between gap-3">
            <Button
              variant="outline"
              size="sm"
              className="h-11 min-w-11 rounded-xl px-3"
              onClick={() => filterControls.setPage(Math.max(1, currentPage - 1))}
              disabled={currentPage <= 1 || isLoading}
              aria-label="Trang dự án trước"
            >
              <ChevronLeft className="h-4 w-4" />
              <span className="sr-only min-[360px]:not-sr-only">Trước</span>
            </Button>
            <p className="text-center text-xs text-muted-foreground tabular-nums">
              Trang {currentPage} / {totalPages}
              <span className="block">{pagination.totalRecords} dự án</span>
            </p>
            <Button
              variant="outline"
              size="sm"
              className="h-11 min-w-11 rounded-xl px-3"
              onClick={() => filterControls.setPage(Math.min(totalPages, currentPage + 1))}
              disabled={currentPage >= totalPages || isLoading}
              aria-label="Trang dự án sau"
            >
              <span className="sr-only min-[360px]:not-sr-only">Sau</span>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        )}
        {pagination && pagination.totalRecords <= pagination.pageSize && projects.length > 0 && (
          <p className="py-3 text-center text-xs text-muted-foreground">
            {pagination.totalRecords} dự án
          </p>
        )}
      </div>

      {/* ── Filter sheet ── */}
      <Sheet open={filterSheetOpen} onOpenChange={setFilterSheetOpen}>
        <SheetContent
          side="bottom"
          className="max-h-[85dvh] overflow-y-auto rounded-t-3xl border-border bg-card px-5 pb-[calc(1.5rem+env(safe-area-inset-bottom))] pt-2"
        >
          <div className="mx-auto mt-2 mb-1 h-1 w-10 rounded-full bg-muted-foreground/30" />
          <SheetHeader className="pb-3">
            <SheetTitle className="text-base font-semibold text-center">Bộ lọc</SheetTitle>
          </SheetHeader>

          <div className="space-y-4">
            {/* Status */}
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Trạng thái</label>
              <Select
                value={
                  filterControls.statusFilter === "all" || !Array.isArray(filterControls.statusFilter) || filterControls.statusFilter.length === 0
                    ? "all"
                    : filterControls.statusFilter[0]
                }
                onValueChange={(v) =>
                  filterControls.setStatusFilter(v === "all" ? "all" : [v as Project["status"]])
                }
              >
                <SelectTrigger className="h-11 rounded-xl">
                  <SelectValue placeholder="Tất cả trạng thái" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả trạng thái</SelectItem>
                  {STATUS_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Month */}
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Tháng</label>
              <Select
                value={filterControls.monthFilter ?? "all"}
                onValueChange={(v) =>
                  filterControls.setMonthFilter(v === "all" ? undefined : v)
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

            {/* Actions */}
            <div className="grid grid-cols-1 gap-3 pt-4 min-[380px]:grid-cols-2">
              <Button
                variant="outline"
                className="flex-1 h-11 rounded-xl"
                onClick={() => {
                  filterControls.clearFilters();
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
    </MobilePageShell>
  );
};

export default ProjectsPageMobile;
