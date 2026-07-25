import { useState, useEffect, useRef, useMemo, useCallback } from "react";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { useSearchParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import { ProjectMobileList } from "@/components/projects/ProjectMobileList";
import { useProjects } from "@/hooks/api/useProjects";
import { useProjectFilters } from "@/hooks/projects/useProjectFilters";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { getVietnameseProjectStatus } from "@/utils/vietnamese";
import { generateMonthOptions } from "@/utils/dateHelpers";
import { Briefcase, Plus, Search, SlidersHorizontal, X } from "lucide-react";
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
  const [searchParams, setSearchParams] = useSearchParams();
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);
  const [displayCount, setDisplayCount] = useState(ITEMS_PER_PAGE);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const loaderRef = useRef<HTMLDivElement>(null);

  const filterControls = useProjectFilters({ initialFilters: { page: 1, pageSize: 100 } });
  const { data: response, isLoading } = useProjects(filterControls.apiFilters);
  const projects = response?.data || [];
  const monthOptions = useMemo(() => generateMonthOptions(), []);
  const { openCreateProject, openProjectDetails } = useProjectModals();

  // Handle URL params from dashboard
  useEffect(() => {
    const action = searchParams.get("action");
    if (action === "add") {
      const p = new URLSearchParams(searchParams);
      p.delete("action");
      p.set("modal", "project_create");
      setSearchParams(p, { replace: true });
    }
    const status = searchParams.get("status");
    if (status === "active") filterControls.setStatusFilter(["active"]);
  }, [searchParams, setSearchParams, filterControls]);

  const paginatedProjects = projects.slice(0, displayCount);
  const hasMore = displayCount < projects.length;

  useEffect(() => {
    setDisplayCount(ITEMS_PER_PAGE);
  }, [filterControls.searchTerm, filterControls.statusFilter, filterControls.monthFilter]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isLoadingMore && !isLoading) {
          setIsLoadingMore(true);
          setTimeout(() => { setDisplayCount((p) => p + ITEMS_PER_PAGE); setIsLoadingMore(false); }, 300);
        }
      },
      { threshold: 0.1 },
    );
    const el = loaderRef.current;
    if (el) observer.observe(el);
    return () => { if (el) observer.unobserve(el); };
  }, [hasMore, isLoadingMore, isLoading]);

  const activeFilterCount = useMemo(() => {
    let c = 0;
    if (filterControls.statusFilter !== "all") c++;
    if (filterControls.monthFilter) c++;
    return c;
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
    (project: Project) => openProjectDetails(project.id.toString()),
    [openProjectDetails],
  );

  if (isLoading && projects.length === 0) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-7 w-24" />
          <Skeleton className="h-9 w-24" />
        </div>
        <Skeleton className="h-11 w-full rounded-xl" />
        <div className="space-y-2">{[...Array(6)].map((_, i) => <Skeleton key={i} className="h-14 w-full rounded-xl" />)}</div>
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-col pb-[calc(6rem+env(safe-area-inset-bottom))]">
      {/* Header */}
      <MobilePageHeader
        title="Dự án"
        icon={Briefcase}
        actionsLayout="inline"
        actions={
          <Button
            size="sm"
            className="ct-btn ct-btn-primary ct-btn-sm h-11 min-h-11 shrink-0 rounded-xl px-3 normal-case shadow-sm"
            onClick={() => openCreateProject()}
          >
            <Plus className="h-4 w-4" />
            Tạo dự án
          </Button>
        }
      />

      {/* Search + filter */}
      <div className="flex gap-2 px-4 py-3">
        <MobileSearchInput
          value={filterControls.searchTerm}
          onSearch={filterControls.setSearchTerm}
          placeholder="Tìm kiếm dự án..."
        />
        <Button variant="outline" size="icon" className="h-11 w-11 rounded-xl border-border bg-card shrink-0 relative" onClick={() => setFilterSheetOpen(true)} aria-label="Bộ lọc">
          <SlidersHorizontal className="h-4 w-4" />
          {activeFilterCount > 0 && (
            <span className="absolute -top-1 -right-1 h-4 w-4 rounded-full bg-primary text-[11px] text-white flex items-center justify-center font-bold">{activeFilterCount}</span>
          )}
        </Button>
      </div>

      {/* Active filter chips */}
      {activeFilterCount > 0 && (
        <div className="px-4 pb-3 flex gap-2 flex-wrap">
          {currentStatusLabel && (
            <Badge variant="secondary" className="gap-1 cursor-pointer" onClick={() => filterControls.setStatusFilter("all")}>
              {currentStatusLabel}<X className="h-3 w-3" />
            </Badge>
          )}
          {currentMonthLabel && (
            <Badge variant="secondary" className="gap-1 cursor-pointer" onClick={() => filterControls.setMonthFilter(undefined)}>
              {currentMonthLabel}<X className="h-3 w-3" />
            </Badge>
          )}
          <button onClick={filterControls.clearFilters} className="min-h-11 px-1 text-xs text-muted-foreground underline underline-offset-2">Xóa tất cả</button>
        </div>
      )}

      {/* List */}
      <div className="flex-1 px-4">
        <ProjectMobileList
          projects={paginatedProjects}
          onRowClick={handleProjectClick}
          emptyState={
            <div className="text-center py-12">
              <Briefcase className="mx-auto h-12 w-12 text-muted-foreground/50" />
              <h3 className="mt-4 typography-title-large">Không tìm thấy dự án nào</h3>
              <p className="mt-2 typography-body-medium text-muted-foreground">Hãy tạo dự án đầu tiên để bắt đầu quản lý.</p>
            </div>
          }
        />
        {hasMore && (
          <div ref={loaderRef} className="flex justify-center py-4">
            <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        )}
        {!hasMore && paginatedProjects.length > 0 && (
          <p className="text-center py-3 text-xs text-muted-foreground">{projects.length} dự án</p>
        )}
      </div>

      {/* Filter sheet */}
      <Sheet open={filterSheetOpen} onOpenChange={setFilterSheetOpen}>
        <SheetContent
          side="bottom"
          className="max-h-[85dvh] overflow-y-auto rounded-t-2xl pb-[calc(1.25rem+env(safe-area-inset-bottom))]"
        >
          <SheetHeader className="pb-4"><SheetTitle>Bộ lọc</SheetTitle></SheetHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Trạng thái</label>
              <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
                {STATUS_OPTIONS.map((option) => {
                  const selected =
                    Array.isArray(filterControls.statusFilter) &&
                    filterControls.statusFilter.includes(option.value);
                  return (
                    <Button
                      key={option.value}
                      type="button"
                      variant={selected ? "default" : "outline"}
                      className="min-h-11 justify-start"
                      onClick={() => {
                        const current = Array.isArray(filterControls.statusFilter)
                          ? filterControls.statusFilter
                          : [];
                        const next = selected
                          ? current.filter((status) => status !== option.value)
                          : [...current, option.value];
                        filterControls.setStatusFilter(
                          next.length === 0 ? "all" : next,
                        );
                      }}
                    >
                      {option.label}
                    </Button>
                  );
                })}
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Tháng</label>
              <Select value={filterControls.monthFilter ?? "all"} onValueChange={(v) => filterControls.setMonthFilter(v === "all" ? undefined : v)}>
                <SelectTrigger className="h-11"><SelectValue placeholder="Tất cả tháng" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả tháng</SelectItem>
                  {monthOptions.map((opt) => <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>)}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Sắp xếp</label>
              <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                <Select
                  value={filterControls.sortBy}
                  onValueChange={filterControls.setSortBy}
                >
                  <SelectTrigger className="h-11 min-w-0">
                    <SelectValue placeholder="Sắp xếp theo" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="name">Tên dự án</SelectItem>
                    <SelectItem value="client_name">Khách hàng</SelectItem>
                    <SelectItem value="created_at">Ngày tạo</SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  type="button"
                  variant="outline"
                  className="h-11 min-w-20"
                  onClick={() =>
                    filterControls.setSortOrder(
                      filterControls.sortOrder === "asc" ? "desc" : "asc",
                    )
                  }
                >
                  {filterControls.sortOrder === "asc" ? "Tăng" : "Giảm"}
                </Button>
              </div>
            </div>
            <div className="grid grid-cols-1 gap-3 pt-2 min-[380px]:grid-cols-2">
              <Button variant="outline" className="flex-1 h-11" onClick={() => { filterControls.clearFilters(); setFilterSheetOpen(false); }}>Xóa bộ lọc</Button>
              <Button className="flex-1 h-11 btn-admin-primary" onClick={() => setFilterSheetOpen(false)}>Áp dụng</Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default ProjectsPageMobile;
