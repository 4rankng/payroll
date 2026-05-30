import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { SearchBar } from '@/components/shared/SearchBar';
import type { ProjectFilters } from '@/types/api/project.types';

type ProjectStatus = 'draft' | 'active' | 'paused' | 'completed' | 'cancelled';

interface PartnerProjectsFiltersProps {
  filters: ProjectFilters;
  onFiltersChange: (filters: Partial<ProjectFilters>) => void;
}

export const PartnerProjectsFilters = ({ filters, onFiltersChange }: PartnerProjectsFiltersProps) => {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <SearchBar
        searchTerm={filters.search || ''}
        onSearchChange={(v) => onFiltersChange({ search: v })}
        placeholder="Tìm dự án..."
        className="w-48"
      />
      <Select
        value={Array.isArray(filters.status) ? filters.status.join(',') : (filters.status || 'all')}
        onValueChange={(v) => onFiltersChange({ status: v === 'all' ? undefined : (v as ProjectStatus) })}
      >
        <SelectTrigger className="h-8 w-[148px] text-xs">
          <SelectValue placeholder="Tất cả trạng thái" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Tất cả trạng thái</SelectItem>
          <SelectItem value="active">Đang dùng</SelectItem>
          <SelectItem value="completed">Kết thúc</SelectItem>
          <SelectItem value="draft">Bản nháp</SelectItem>
          <SelectItem value="cancelled">Đã hủy</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
};
