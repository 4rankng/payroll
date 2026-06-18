import { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { MobilePageShell, MobileSurface } from "@/components/shared/MobilePageShell";
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
import { ProjectMobileList } from "@/components/projects/ProjectMobileList";
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
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);
  const [displayCount, setDisplayCount] = useState(ITEMS_PER_PAGE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const loaderRef = useRef<HTMLDivElement>(null);

  const filterControls = useProjectFilters({
    initialFilters: { page: 1, pageSize: 100 },
  });

  const { data: response, isLoading } = useProjects(filterControls.apiFilters);
  const { data: summary, isLoading: summaryLoading } = usePartnerProjectSummary();
  const projects = response?.data || [];
  const monthOptions = useMemo(() => generateMonthOptions(), []);

  const { openCreateProject, openPartnerProjectDetails } = useProjectModals();

  const paginatedProjects = projects.slice(0, displayCount);
  const hasMore = displayCount < projects.length;

  // Reset display count when filters change
  useEffect(() => {
    setDisplayCount(ITEMS_PER_PAGE);
  }, [
    filterControls.searchTerm,
    filterControls.statusFilter,
    filterControls.monthFilter,
  ]);

  // Infinite scroll observer
  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isLoadingMore && !isLoading) {
          setIsLoadingMore(true);
          setTimeout(() => {
            setDisplayCount((prev) => prev + ITEMS_PER_PAGE);
            setIsLoadingMore(false);
          }, 300);
        }
      },
      { threshold: 0.1 },
    );
    const el = loaderRef.current;
    if (el) observer.observe(el);
    return () => { if (el) observer.unobserve(el); };
  }, [hasMore, isLoadingMore, isLoading]);

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
          <div className="flex gap-2 overflow-x-auto scrollbar-none">
            {[
              { label: "Tổng", value: summary.total_projects, icon: Briefcase, color: "text-primary", bg: "bg-[#E7EEF6]" },
              { label: "Đang dùng", value: summary.active_projects, icon: CheckCircle, color: "text-emerald-700", bg: "bg-[#DDF7EC]" },
              { label: "Hoàn thành", value: summary.completed_projects, icon: CheckCircle, color: "text-violet-700", bg: "bg-[#EEE7FF]" },
              { label: "Nhân viên", value: summary.total_employees, icon: Users, color: "text-amber-700", bg: "bg-[#FFF0C2]" },
            ].map((stat) => {
              const Icon = stat.icon;
              return (
                <div
                  key={stat.label}
                  className="flex min-h-16 min-w-[72px] shrink-0 flex-col items-center gap-1 rounded-2xl border border-[#D8E2EE] bg-white px-2.5 py-2 transition-all duration-200 active:scale-95"
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
        <div className="px-4 pb-3 flex gap-2">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 w-[72px] shrink-0 rounded-xl" />
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
          className="h-11 w-11 rounded-2xl border-[#D8E2EE] bg-white shrink-0 relative shadow-sm"
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
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer rounded-xl"
              onClick={() => filterControls.setStatusFilter("all")}
            >
              {currentStatusLabel}
              <X className="h-3 w-3" />
            </Badge>
          )}
          {currentMonthLabel && (
            <Badge
              variant="secondary"
              className="gap-1 cursor-pointer rounded-xl"
              onClick={() => filterControls.setMonthFilter(undefined)}
            >
              {currentMonthLabel}
              <X className="h-3 w-3" />
            </Badge>
          )}
          <button
            onClick={filterControls.clearFilters}
            className="text-xs text-muted-foreground underline underline-offset-2"
          >
            Xóa tất cả
          </button>
        </div>
      )}

      {/* ── Project list ── */}
      <div className="flex-1">
        <ProjectMobileList
          projects={paginatedProjects}
          onRowClick={handleProjectClick}
          emptyState={
            <div className="flex flex-col items-center justify-center py-12 gap-3">
              <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-[#E7EEF6]">
                <Briefcase className="h-7 w-7 text-muted-foreground/50" />
              </div>
              <div className="text-center">
                <p className="text-sm font-medium text-foreground">
                  Không có dự án nào
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  Bạn chưa được phân quyền truy cập vào dự án nào.
                </p>
              </div>
            </div>
          }
        />

        {hasMore && (
          <div ref={loaderRef} className="flex justify-center py-4">
            <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        )}
        {!hasMore && paginatedProjects.length > 0 && (
          <p className="text-center py-3 text-xs text-muted-foreground">
            {projects.length} dự án
          </p>
        )}
      </div>

      {/* ── Filter sheet ── */}
      <Sheet open={filterSheetOpen} onOpenChange={setFilterSheetOpen}>
        <SheetContent side="bottom" className="rounded-t-3xl border-[#D8E2EE] bg-white px-5 pt-2 pb-6">
          <div className="mx-auto mt-2 mb-1 h-1 w-10 rounded-full bg-slate-300" />
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
            <div className="flex gap-3 pt-4 pb-safe">
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
