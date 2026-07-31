import { useMemo } from 'react';
import { FilterPill } from '@/components/shared/FilterPill';
import { SearchBar } from '@/components/shared/SearchBar';
import { SearchableDropdown } from '@/components/ui/searchable-dropdown';
import { EmployeeDropdownAdapter } from '@/components/timesheet/EmployeeDropdownAdapter';
import type { Employee } from '@/types/api/employee.types';
import { Badge } from '@/components/ui/badge';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { SlidersHorizontal } from 'lucide-react';
import { useIsMobile } from '@/hooks/useBreakpoint';
import { useTimesheetContext } from '@/components/timesheet/TimesheetContext';

const STATUS_OPTIONS = [
  { value: 'pending_approval', label: 'Chờ duyệt' },
  { value: 'pending_payment', label: 'Chờ TT' },
  { value: 'paid', label: 'Đã thanh toán' },
  { value: 'rejected', label: 'Bị loại' },
];

export const TimesheetFilters = () => {
  const { filters } = useTimesheetContext();
  const {
    selectedMonth,
    onMonthChange,
    selectedProject,
    onProjectChange,
    selectedEmployee,
    onEmployeeChange,
    statusFilter,
    onStatusChange,
    projects,
    projectEmployees,
    searchTerm = '',
    onSearchChange,
    userRole = 'admin',
  } = filters;

  const isMobile = useIsMobile();

  const monthOptions = useMemo(() => {
    const now = new Date();
    return Array.from({ length: 6 }, (_, i) => {
      const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
      return {
        value: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`,
        label: `T${d.getMonth() + 1}/${d.getFullYear()}`,
      };
    });
  }, []);

  const projectOptions = useMemo(() =>
    projects.map(p => {
      const parts = p.name.split(' - ');
      return { value: p.id.toString(), label: `${parts[0]} (${p.code})` };
    }), [projects]);

  const shouldUseProjectEmployees = selectedProject !== 'all' && projectEmployees && projectEmployees.length > 0;

  const activeFilterCount = [
    selectedMonth !== 'all',
    statusFilter !== 'all',
    selectedProject !== 'all',
    selectedEmployee !== 'all',
  ].filter(Boolean).length;

  // ── Mobile ─────────────────────────────────────────────────────────────────
  if (isMobile) {
    return (
      <div className="flex flex-col gap-1.5">
        {onSearchChange && (
          <SearchBar
            searchTerm={searchTerm}
            onSearchChange={onSearchChange}
            placeholder="Tìm kiếm..."
            className="w-full"
          />
        )}
        <div className="flex items-center gap-1.5">
        <Select value={selectedMonth} onValueChange={onMonthChange}>
          <SelectTrigger className="h-11 w-[112px] shrink-0 text-xs border-border/60">
            <SelectValue placeholder="Tháng" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Tất cả</SelectItem>
            {monthOptions.map(m => <SelectItem key={m.value} value={m.value}>{m.label}</SelectItem>)}
          </SelectContent>
        </Select>
        <Sheet>
          <SheetTrigger asChild>
            <button
              className="relative inline-flex h-11 w-11 items-center justify-center rounded-xl border border-border/60 text-muted-foreground transition-colors hover:border-border hover:text-foreground"
              aria-label="Bộ lọc"
            >
              <SlidersHorizontal className="h-3.5 w-3.5" />
              {activeFilterCount > 1 && (
                <Badge className="absolute -top-1.5 -right-1.5 h-4 min-w-4 p-0 text-[10px] flex items-center justify-center rounded-full">
                  {activeFilterCount - 1}
                </Badge>
              )}
            </button>
          </SheetTrigger>
          <SheetContent side="bottom" className="h-auto">
            <SheetHeader className="mb-4">
              <SheetTitle className="text-base">Bộ lọc</SheetTitle>
            </SheetHeader>
            <div className="space-y-3 pb-4" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-muted-foreground">Trạng thái</Label>
                <Select value={statusFilter} onValueChange={onStatusChange}>
                  <SelectTrigger className="h-11"><SelectValue placeholder="Tất cả trạng thái" /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Tất cả trạng thái</SelectItem>
                    {STATUS_OPTIONS.map(o => <SelectItem key={o.value} value={o.value}>{o.label}</SelectItem>)}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-muted-foreground">Dự án</Label>
                <SearchableDropdown
                  value={selectedProject} onValueChange={onProjectChange}
                  options={projectOptions}
                  placeholder="Tất cả dự án" searchPlaceholder="Tìm dự án..."
                  emptyMessage="Không tìm thấy dự án nào." className="w-full"
                  allOption={{ value: 'all', label: 'Tất cả dự án' }}
                  mobileTitle="Chọn dự án"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-muted-foreground">Nhân viên</Label>
                <EmployeeDropdownAdapter
                  value={selectedEmployee} onChange={onEmployeeChange}
                  selectedProject={selectedProject} placeholder="Tất cả nhân viên"
                  className="w-full"
                  availableEmployees={(shouldUseProjectEmployees ? projectEmployees : []) as unknown as Employee[]}
                  includeAvailableOnly={shouldUseProjectEmployees}
                />
              </div>
            </div>
          </SheetContent>
        </Sheet>
        </div>
      </div>
    );
  }

  // ── Desktop ────────────────────────────────────────────────────────────────
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      {onSearchChange && (
        <SearchBar
          searchTerm={searchTerm}
          onSearchChange={onSearchChange}
          placeholder="Tìm nhân viên, dự án..."
          className="w-64"
        />
      )}

      <div className="h-5 w-px bg-border/50 shrink-0 hidden sm:block" />

      {userRole !== 'partner' && (
        <FilterPill
          value={selectedMonth}
          onChange={onMonthChange}
          placeholder="Tháng"
          options={monthOptions}
          className="bg-card"
        />
      )}

      <FilterPill
        value={statusFilter}
        onChange={onStatusChange}
        placeholder="Trạng thái"
        options={STATUS_OPTIONS}
        className="bg-card"
      />

      <SearchableDropdown
        value={selectedProject}
        onValueChange={onProjectChange}
        options={projectOptions}
        placeholder="Dự án"
        searchPlaceholder="Tìm dự án..."
        emptyMessage="Không tìm thấy dự án nào."
        className="h-11 min-h-11 rounded-xl bg-card px-3 text-sm"
        allOption={{ value: 'all', label: 'Dự án' }}
        pillStyle
      />

      <EmployeeDropdownAdapter
        value={selectedEmployee}
        onChange={onEmployeeChange}
        selectedProject={selectedProject}
        placeholder="Nhân viên"
        className="inline-flex h-11 min-h-11 items-center gap-1 rounded-xl border border-border/60 bg-card px-3 py-0 text-sm font-medium whitespace-nowrap text-muted-foreground transition-colors duration-100 hover:border-border hover:bg-accent/40 hover:text-foreground [&>svg]:h-3.5 [&>svg]:w-3.5"
        availableEmployees={(shouldUseProjectEmployees ? projectEmployees : []) as unknown as Employee[]}
        includeAvailableOnly={shouldUseProjectEmployees}
        pillStyle
      />
    </div>
  );
};
