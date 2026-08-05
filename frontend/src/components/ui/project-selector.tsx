import { useState, useMemo, useCallback } from 'react';
import { useInfiniteQuery } from '@tanstack/react-query';
import { AsyncSearchableDropdown } from '@/components/ui/async-searchable-dropdown';
import { useDebounce } from '@/hooks/useDebounce';
import { projectService } from '@/services/api/project.service';
import { cn } from '@/lib/utils';
import type { Project } from '@/types/api/project.types';

interface ProjectSelectorProps {
  value?: Project | null;
  onSelect: (project: Project | null) => void;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  activeOnly?: boolean;
  flexibleOnly?: boolean;
  excludeProjectIds?: number[];
}

export function ProjectSelector({
  value,
  onSelect,
  placeholder = "Chọn dự án...",
  disabled = false,
  className,
  activeOnly = false,
  flexibleOnly = false,
  excludeProjectIds = []
}: ProjectSelectorProps) {
  const [searchValue, setSearchValue] = useState('');
  const debouncedSearchValue = useDebounce(searchValue, 300);
  const isSearching = searchValue !== debouncedSearchValue;

  const {
    data: projectsData,
    isLoading,
    error,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage
  } = useInfiniteQuery({
    queryKey: ['projects', 'selector', { activeOnly, flexibleOnly, search: debouncedSearchValue }],
    queryFn: async ({ pageParam = 1 }) => {
      const filters = {
        page: pageParam,
        pageSize: 20,
        status: activeOnly ? ['active' as const] : ['active' as const, 'draft' as const, 'paused' as const],
        sortBy: 'created_at',
        sortOrder: 'desc' as const,
        ...(debouncedSearchValue.length >= 3 && { search: debouncedSearchValue })
      };

      return await projectService.getProjects(filters);
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      if (!lastPage.pagination) return undefined;
      const { page, totalPages } = lastPage.pagination;
      return page < totalPages ? page + 1 : undefined;
    },
    enabled: debouncedSearchValue.length === 0 || debouncedSearchValue.length >= 3,
  });

  const projects = useMemo(() => {
    const allProjects = projectsData?.pages.flatMap(page => page.data || []) || [];
    return allProjects.filter(project =>
      !excludeProjectIds.includes(project.id) && (!flexibleOnly || project.is_flexible)
    );
  }, [projectsData, excludeProjectIds, flexibleOnly]);

  const handleSelect = useCallback((project: Project) => {
    onSelect(project);
  }, [onSelect]);

  const renderTrigger = useCallback((selected: Project | null, placeholderText: string) => {
    if (selected) {
      return (
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <span className="font-medium truncate text-sm">{selected.name}</span>
          {selected.code && (
            <span className="text-xs text-muted-foreground shrink-0">
              {selected.code}
            </span>
          )}
        </div>
      );
    }
    return <span className="truncate text-left text-sm">{placeholderText}</span>;
  }, []);

  const renderOption = useCallback((project: Project) => (
    <div className="flex flex-col flex-1 min-w-0">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="font-medium">{project.name}</span>
          {project.code && (
            <span className="typography-body-medium text-muted-foreground">
              Mã: {project.code}
            </span>
          )}
        </div>
      </div>
    </div>
  ), []);

  const filterValue = useCallback((project: Project) => `${project.name} ${project.client_name} ${project.code || ''}`, []);

  const errorMessage = error ? 'Có lỗi xảy ra khi tìm kiếm dự án' : undefined;

  return (
    <AsyncSearchableDropdown
      options={projects}
      value={value ?? null}
      onSelect={handleSelect}
      placeholder={placeholder}
      searchValue={searchValue}
      onSearchValueChange={setSearchValue}
      isLoading={isLoading}
      isSearching={isSearching}
      hasNextPage={hasNextPage}
      isFetchingNextPage={isFetchingNextPage}
      loadMore={fetchNextPage}
      errorMessage={errorMessage}
      searchPlaceholder="Tên hoặc mã dự án"
      loadingMessage="Đang tìm kiếm..."
      loadingMoreMessage="Đang tải thêm..."
      emptyMessages={{
        searchTooShort: "Nhập tối thiểu 3 ký tự để tìm kiếm",
        noResults: "Không tìm thấy dự án nào",
        empty: "Không có dự án khả dụng"
      }}
      minSearchLength={3}
      renderTrigger={renderTrigger}
      renderOption={renderOption}
      getOptionKey={(project) => project.id}
      getOptionFilterValue={filterValue}
      disabled={disabled}
      triggerClassName={cn("w-full justify-between", className)}
      popoverContentClassName="w-[var(--radix-popover-trigger-width)] max-w-[400px] p-0"
    />
  );
}
