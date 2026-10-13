import { ReactNode } from 'react';
import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { useAllProjects } from '@/hooks/api/useProjects';
import { cn } from '@/lib/utils';

type UserRoleFilter = 'admin' | 'partner' | 'employee' | 'accountant';

interface UserFiltersBarProps {
  search: string;
  onSearchChange: (value: string) => void;
  role: UserRoleFilter | undefined;
  onRoleChange: (role: UserRoleFilter | undefined) => void;
  projectId: number | undefined;
  onProjectChange: (projectId: number | undefined) => void;
  onClearFilters?: () => void;
  hasActiveFilters?: boolean;
  hideRoleFilter?: boolean;
  extra?: ReactNode;
  className?: string;
}

export const UserFiltersBar = ({
  search,
  onSearchChange,
  role,
  onRoleChange,
  projectId,
  onProjectChange,
  hideRoleFilter = false,
  extra,
  className,
}: UserFiltersBarProps) => {
  const { data: projects } = useAllProjects();

  return (
    <div className={cn('flex items-center gap-1.5 flex-wrap', className)}>
      <SearchBar
        searchTerm={search}
        onSearchChange={onSearchChange}
        placeholder="Tìm theo tên, email..."
        className={hideRoleFilter ? 'flex-1 min-w-[200px] max-w-md' : 'w-48'}
      />
      {!hideRoleFilter && (
        <FilterPill
          value={role ?? 'all'}
          onChange={(v) => onRoleChange(v === 'all' ? undefined : v as UserRoleFilter)}
          placeholder="Vai trò"
          options={[
            { value: 'admin', label: 'Quản trị viên' },
            { value: 'partner', label: 'Quản lý' },
            { value: 'employee', label: 'Nhân viên' },
            { value: 'accountant', label: 'Kế toán' },
          ]}
        />
      )}
      <FilterPill
        value={projectId !== undefined ? String(projectId) : 'all'}
        onChange={(v) => onProjectChange(v === 'all' ? undefined : Number(v))}
        placeholder="Dự án"
        options={(projects ?? []).map((project) => ({
          value: String(project.id),
          label: project.name,
        }))}
      />
      {extra}
    </div>
  );
};
