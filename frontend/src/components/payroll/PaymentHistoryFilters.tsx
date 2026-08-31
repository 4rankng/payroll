import { memo, useMemo } from 'react';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { Input } from '@/components/ui/input';
import { useProjects } from '@/hooks/api/useProjects';
import { useEmployees } from '@/hooks/api/useEmployees';
import { useProjectEmployees } from '@/hooks/api/useProjectEmployees';
import { SearchBar } from '@/components/shared/SearchBar';
import { X } from 'lucide-react';
import type { PaymentHistoryFilters as FiltersType } from '@/types/api/payroll.types';

interface PaymentHistoryFiltersProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
}

export const PaymentHistoryFilters = memo(function PaymentHistoryFilters({
  filters,
  onFiltersChange,
  onClearFilters,
  hasFilters,
}: PaymentHistoryFiltersProps) {
  const { data: projectsData } = useProjects({ pageSize: 100, status: ['active', 'paused', 'completed'] });

  const selectedProjectId = useMemo(() => filters.projectId?.[0], [filters.projectId]);

  const { data: projectEmployeesData, isLoading: isLoadingProjectEmployees } = useProjectEmployees(
    selectedProjectId!,
    { pageSize: 100, status: 'current' },
    !!selectedProjectId
  );
  const { data: allEmployeesData, isLoading: isLoadingAllEmployees } = useEmployees(
    !selectedProjectId ? { pageSize: 100, sortBy: 'fullname', sortOrder: 'asc' } : undefined
  );

  const employeesData = useMemo(() => {
    if (selectedProjectId && projectEmployeesData?.data) {
      return {
        data: projectEmployeesData.data.map(a => ({ id: a.employee_id, name: a.employee_name })),
        pagination: projectEmployeesData.pagination,
      };
    }
    if (allEmployeesData?.data) {
      return {
        data: allEmployeesData.data.map(e => ({ id: e.id, name: e.fullname })),
        pagination: allEmployeesData.pagination,
      };
    }
    return undefined;
  }, [selectedProjectId, projectEmployeesData, allEmployeesData]);

  const isLoadingEmployees = selectedProjectId ? isLoadingProjectEmployees : isLoadingAllEmployees;
  const selectedEmployeeId = useMemo(() => filters.employeeId?.[0], [filters.employeeId]);

  const set = (key: keyof FiltersType, value: unknown) =>
    onFiltersChange({ ...filters, [key]: value, page: 1 });

  const handleProjectChange = (value: string) => {
    if (value === 'all') {
      onFiltersChange({ ...filters, projectId: undefined, employeeId: undefined, page: 1 });
    } else {
      onFiltersChange({ ...filters, projectId: [parseInt(value)], employeeId: undefined, page: 1 });
    }
  };

  const handleEmployeeChange = (value: string) => {
    set('employeeId', value === 'all' ? undefined : [parseInt(value)]);
  };

  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <SearchBar
        searchTerm={filters.search || ''}
        onSearchChange={(v) => set('search', v || undefined)}
        placeholder="Tìm tên, CCCD..."
        className="w-44"
      />

      <Input
        type="date"
        value={filters.fromDate || ''}
        onChange={(e) => set('fromDate', e.target.value)}
        className="h-8 w-[130px] text-xs"
        aria-label="Từ ngày"
      />
      <span className="text-muted-foreground text-xs">–</span>
      <Input
        type="date"
        value={filters.toDate || ''}
        onChange={(e) => set('toDate', e.target.value)}
        className="h-8 w-[130px] text-xs"
        aria-label="Đến ngày"
      />

      <SearchableSelect
        value={selectedProjectId?.toString() || 'all'}
        onChange={handleProjectChange}
        options={[
          { value: 'all', label: 'Tất cả dự án' },
          ...(projectsData?.data?.map((p) => ({ value: p.id.toString(), label: p.name })) ?? []),
        ]}
        placeholder="Tất cả dự án"
        searchPlaceholder="Tìm dự án..."
        triggerClassName="h-8 min-h-0 w-[148px] text-xs"
      />

      <SearchableSelect
        key={selectedProjectId || 'all'}
        value={selectedEmployeeId?.toString() || 'all'}
        onChange={handleEmployeeChange}
        options={[
          { value: 'all', label: 'Tất cả nhân viên' },
          ...(employeesData?.data?.map((e) => ({ value: e.id.toString(), label: e.name })) ?? []),
        ]}
        placeholder={isLoadingEmployees ? 'Đang tải...' : 'Tất cả nhân viên'}
        disabled={isLoadingEmployees}
        searchPlaceholder="Tìm nhân viên..."
        triggerClassName="h-8 min-h-0 w-[148px] text-xs"
      />

      {hasFilters && (
        <button
          onClick={onClearFilters}
          className="flex items-center gap-1 h-8 px-2 text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          <X className="h-3 w-3" />
          Xóa lọc
        </button>
      )}
    </div>
  );
});
