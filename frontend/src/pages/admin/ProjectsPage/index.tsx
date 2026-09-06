import { useMemo, useEffect, useCallback } from "react";
import { useSearchParams } from "react-router-dom";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
  AdminFilterRow,
} from "@/components/shared/AdminPageFrame";
import { ProjectPageHeader } from "@/components/projects/ProjectPageHeader";
import { ProjectFilters } from "@/components/projects/ProjectFilters";
import { AddProjectSheet } from "@/components/sheets/AddProjectSheet";
import { useProjects } from "@/hooks/api/useProjects";
import { useProjectFilters } from "@/hooks/projects/useProjectFilters";
import { useProjectModals } from "@/hooks/useModalNavigation";
import { createProjectColumns } from "@/config/project-table-columns";
import { createProjectMobileConfig } from "@/config/project-table-mobile";
import { Skeleton } from "@/components/ui/skeleton";
import { EmptyState } from "@/components/shared/EmptyState";
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
      <AdminPageCanvas>
        <AdminPageHeaderCard>
          <Skeleton className="h-8 w-40" />
          <Skeleton className="mt-2 h-4 w-72" />
        </AdminPageHeaderCard>
        <AdminSectionCard>
          <Skeleton className="m-3 h-11 rounded-xl sm:m-4" />
          <div className="border-t border-slate-200/70">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="flex items-center gap-4 border-b border-slate-200/60 px-4 py-4 last:border-b-0">
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
        </AdminSectionCard>
      </AdminPageCanvas>
    );
  }

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <ProjectPageHeader onCreateProject={() => openCreateProject()} />
      </AdminPageHeaderCard>

      <AdminSectionCard aria-label="Danh sách dự án">
        <AdminFilterRow>
          <p className="text-sm font-semibold text-slate-800">Danh sách</p>
          <div className="flex flex-1 items-center justify-end">
            <ProjectFilters
              searchTerm={filterControls.searchTerm}
              onSearchChange={filterControls.setSearchTerm}
              statusFilter={filterControls.statusFilter}
              onStatusChange={filterControls.setStatusFilter}
              monthFilter={filterControls.monthFilter}
              onMonthFilterChange={filterControls.setMonthFilter}
              hasFilters={filterControls.hasFilters}
              onClearFilters={filterControls.clearFilters}
              className="rounded-none border-0 bg-transparent px-0 py-0 backdrop-blur-none"
            />
          </div>
        </AdminFilterRow>

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
            <EmptyState
              title="Không tìm thấy dự án nào"
              description="Hãy tạo dự án đầu tiên để bắt đầu quản lý nhân viên và bảng lương."
              action={{ label: "Tạo dự án", onClick: () => openCreateProject() }}
              className="py-8"
            />
          }
          accordionType="single"
          embedded
        />
      </AdminSectionCard>
    </AdminPageCanvas>
  );
};

export default ProjectsPage;
