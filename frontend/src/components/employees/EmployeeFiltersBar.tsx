import { ReactNode } from 'react';
import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { EmployeeStatusFilter } from './EmployeeStatusFilter';
import { cn } from '@/lib/utils';
import { generateMonthOptions } from '@/utils/dateHelpers';

interface Project {
  id: number;
  name: string;
  code: string;
}

interface EmployeeFiltersBarProps {
  searchTerm: string;
  onSearchChange: (v: string) => void;
  statusFilter?: 'working' | 'unassigned';
  onStatusFilterChange: (status: 'working' | 'unassigned' | undefined) => void;
  month?: string;
  onMonthChange: (month: string | undefined) => void;
  fromDate?: string;
  toDate?: string;
  onDateRangeChange?: (from: string | undefined, to: string | undefined) => void;
  projectId?: number | null;
  onProjectChange?: (projectId: number | null) => void;
  projects?: Project[];
  extra?: ReactNode;
  className?: string;
}

const monthOptions = generateMonthOptions(12).map(o => ({
  value: o.value,
  label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
}));

export const EmployeeFiltersBar = ({
  searchTerm,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  month,
  onMonthChange,
  projectId,
  onProjectChange,
  projects,
  extra,
  className,
}: EmployeeFiltersBarProps) => {
  return (
    <div className={cn(
      'flex items-center gap-2 flex-wrap rounded-xl border border-border/50 bg-card/50 backdrop-blur-sm px-3 py-2',
      className,
    )}>
      <SearchBar
        searchTerm={searchTerm}
        onSearchChange={onSearchChange}
        placeholder="Tìm nhân viên..."
        className="w-44"
      />

      <div className="h-5 w-px bg-border/50 shrink-0 hidden sm:block" />

      {projects && onProjectChange && (
        <FilterPill
          value={projectId ? projectId.toString() : 'all'}
          onChange={(v) => onProjectChange(v === 'all' ? null : parseInt(v))}
          placeholder="Dự án"
          options={projects.map((p) => ({ value: p.id.toString(), label: `${p.name} (${p.code})` }))}
        />
      )}

      <EmployeeStatusFilter
        value={statusFilter}
        onValueChange={onStatusFilterChange}
        className="h-7"
      />

      <FilterPill
        value={month ?? 'all'}
        onChange={(v) => onMonthChange(v === 'all' ? undefined : v)}
        placeholder="Tháng"
        options={monthOptions}
      />

      {extra}
    </div>
  );
};
