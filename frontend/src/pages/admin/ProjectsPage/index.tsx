import { useMemo, useEffect, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { ProjectPageHeader } from "@/components/projects/ProjectPageHeader";
import { ProjectFilters } from "@/components/projects/ProjectFilters";
import { AddProjectSheet } from "@/components/sheets/AddProjectSheet";
import { useProjects } from "@/hooks/api/useProjects";
import { useProjectFilters } from "@/hooks/projects/useProjectFilters";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { createProjectColumns } from "@/config/project-table-columns";
import { createProjectMobileConfig } from "@/config/project-table-mobile";
import { Skeleton } from "@/components/ui/skeleton";
import { Briefcase, FolderPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { CreateProjectData } from "@/types/api/project.types";
import { useTableSorting } from "@/utils/sorting";

const ProjectsPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const filterControls = useProjectFilters({
    initialFilters: {
      page: 1,
      pageSize: 10,
    }
  });

  useEffect(() => {
    const action = searchParams.get('action');
    if (action === 'add') {
      const newParams = new URLSearchParams(searchParams);
      newParams.delete('action');
      newParams.set('modal', 'project_create');
      setSearchParams(newParams, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  useEffect(() => {
    const status = searchParams.get('status');
    if (status === 'active') {
      filterControls.setStatusFilter(['active']);
    }
  }, [searchParams, filterControls]);

  const { data: response, isLoading } = useProjects(filterControls.apiFilters);
  const projects = response?.data || [];

  const handleSortChange = useCallback((newSortBy: string, newSortOrder: 'asc' | 'desc') => {
    filterControls.setSortBy(newSortBy);
    filterControls.setSortOrder(newSortOrder);
  }, [filterControls]);

  const { sorting, onSortingChange } = useTableSorting(
    filterControls.sortBy,
    filterControls.sortOrder,
    handleSortChange,
  );

  const { openCreateProject, openProjectDetails } = useProjectModals();

  const columns = useMemo(() => createProjectColumns({
    pagination: response?.pagination
  }), [response?.pagination]);

  const mobileConfig = createProjectMobileConfig({});

  if (isLoading && projects.length === 0) {
    return (
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5 animate-fade-in">
        <div className="space-y-2">
          <Skeleton className="h-7 w-32" />
          <Skeleton className="h-4 w-64" />
        </div>
        <Skeleton className="h-11 w-full max-w-md rounded-xl" />
        <div className="rounded-lg border bg-background overflow-hidden">
          <div className="border-b px-4 py-3">
            <div className="flex gap-6">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-3 w-20" />
              ))}
            </div>
          </div>
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="border-b px-4 py-4 flex items-center gap-4">
              <Skeleton className="h-9 w-9 rounded-lg" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3.5 w-40" />
                <Skeleton className="h-2.5 w-20" />
              </div>
              <Skeleton className="h-6 w-20 rounded-md" />
              <Skeleton className="h-3.5 w-24" />
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
          <ProjectPageHeader onCreateProject={() => openCreateProject()} />
        </div>

        <div className="opacity-0 animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:forwards]">
          <ProjectFilters
            searchTerm={filterControls.searchTerm}
            onSearchChange={filterControls.setSearchTerm}
            statusFilter={filterControls.statusFilter}
            onStatusChange={filterControls.setStatusFilter}
            monthFilter={filterControls.monthFilter}
            onMonthFilterChange={filterControls.setMonthFilter}
            hasFilters={filterControls.hasFilters}
            onClearFilters={filterControls.clearFilters}
          />
        </div>

        <div className="opacity-0 animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:forwards]">
          <ResponsiveTable
            data={projects}
            columns={columns}
            mobileFields={mobileConfig.mobileFields}
            rowTitle={mobileConfig.rowTitle}
            rowSubtitle={mobileConfig.rowSubtitle}
            getRowId={(row) => row.id.toString()}
            onRowClick={(project) => openProjectDetails(project.id.toString())}
            pagination={response?.pagination}
            onPageChange={filterControls.setPage}
            onPageSizeChange={filterControls.setPageSize}
            sorting={sorting}
            onSortingChange={onSortingChange}
            emptyState={
              <div className="flex flex-col items-center justify-center py-16 px-4">
                <div className="relative mb-6">
                  <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-primary/5 border border-primary/10">
                    <Briefcase className="h-9 w-9 text-primary/30" />
                  </div>
                  <div className="absolute -right-1 -bottom-1 flex h-8 w-8 items-center justify-center rounded-lg bg-background border shadow-sm">
                    <FolderPlus className="h-4 w-4 text-muted-foreground" />
                  </div>
                </div>
                <h3 className="typography-headline-small text-foreground mb-1">
                  Không tìm thấy dự án nào
                </h3>
                <p className="typography-body-medium text-muted-foreground max-w-xs text-center mb-6">
                  Hãy tạo dự án đầu tiên để bắt đầu quản lý nhân viên và bảng lương.
                </p>
                <Button
                  onClick={() => openCreateProject()}
                  variant="outline"
                  size="sm"
                  className="gap-1.5"
                >
                  <FolderPlus className="h-3.5 w-3.5" />
                  Tạo dự án
                </Button>
              </div>
            }
            accordionType="single"
          />
        </div>

      </div>
    </div>
  );
};

export default ProjectsPage;
