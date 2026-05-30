import { ReactNode } from 'react';
import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { cn } from '@/lib/utils';

interface UserFiltersBarProps {
  search: string;
  onSearchChange: (value: string) => void;
  role: 'admin' | 'partner' | 'employee' | undefined;
  onRoleChange: (role: 'admin' | 'partner' | 'employee' | undefined) => void;
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
  hideRoleFilter = false,
  extra,
  className,
}: UserFiltersBarProps) => {
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
          onChange={(v) => onRoleChange(v === 'all' ? undefined : v as 'admin' | 'partner' | 'employee')}
          placeholder="Vai trò"
          options={[
            { value: 'admin', label: 'Quản trị viên' },
            { value: 'partner', label: 'Quản lý' },
            { value: 'employee', label: 'Nhân viên' },
          ]}
        />
      )}
      {extra}
    </div>
  );
};
