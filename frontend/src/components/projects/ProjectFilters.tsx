import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { Project } from '@/types/api/project.types';
import { generateMonthOptions } from '@/utils/dateHelpers';
import { getVietnameseProjectStatus } from '@/utils/vietnamese';
import { X } from 'lucide-react';

interface ProjectFiltersProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  statusFilter: Project['status'][] | 'all';
  onStatusChange: (value: Project['status'][] | 'all') => void;
  monthFilter?: string;
  onMonthFilterChange: (value: string | undefined) => void;
  hasFilters: boolean;
  onClearFilters: () => void;
}

const STATUS_OPTIONS: Project['status'][] = ['active', 'paused', 'completed', 'cancelled'];
const monthOptions = generateMonthOptions(12).map(o => ({
  value: o.value,
  label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
}));

export const ProjectFilters = ({
  searchTerm,
  onSearchChange,
  statusFilter,
  onStatusChange,
  monthFilter,
  onMonthFilterChange,
  hasFilters,
  onClearFilters,
}: ProjectFiltersProps) => {
  const currentStatus = Array.isArray(statusFilter) && statusFilter.length === 1
    ? statusFilter[0]
    : 'all';

  return (
    <div className="flex items-center gap-2 flex-wrap rounded-xl border border-border/50 bg-card/50 backdrop-blur-sm px-3 py-2">
      <SearchBar
        searchTerm={searchTerm}
        onSearchChange={onSearchChange}
        placeholder="Tìm dự án, khách hàng..."
        className="w-52"
      />

      <div className="h-5 w-px bg-border/50 shrink-0 hidden sm:block" />

      <FilterPill
        value={currentStatus}
        onChange={(v) => onStatusChange(v === 'all' ? 'all' : [v as Project['status']])}
        placeholder="Trạng thái"
        options={STATUS_OPTIONS.map(s => ({ value: s, label: getVietnameseProjectStatus(s) }))}
      />

      <FilterPill
        value={monthFilter ?? 'all'}
        onChange={(v) => onMonthFilterChange(v === 'all' ? undefined : v)}
        placeholder="Tháng"
        options={monthOptions}
      />

      {hasFilters && (
        <button
          onClick={onClearFilters}
          className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors ml-1 px-2 py-1 rounded-md hover:bg-muted/50"
        >
          <X className="h-3 w-3" />
          Xóa lọc
        </button>
      )}
    </div>
  );
};
