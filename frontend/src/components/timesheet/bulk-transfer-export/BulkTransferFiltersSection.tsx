import { memo, useMemo } from 'react';
import { Label } from '@/components/ui/label';
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible';
import { ProjectMultiSelector } from '@/components/ui/project-multi-selector';
import { EmployeeMultiSelector } from '@/components/ui/employee-multi-selector';
import { ChevronDown, SlidersHorizontal } from 'lucide-react';
import { cn } from '@/lib/utils';

interface BulkTransferFiltersSectionProps {
  selectedProjects: number[];
  selectedEmployees: number[];
  projects: Array<{ id: number; code: string; name: string }>;
  employees: Array<{
    id: number;
    employee_code?: string;
    fullname: string;
    cccd?: string | null;
    date_of_birth?: string | null;
  }>;
  onProjectChange: (ids: number[]) => void;
  onEmployeeChange: (ids: number[]) => void;
}

export const BulkTransferFiltersSection = memo(function BulkTransferFiltersSection({
  selectedProjects,
  selectedEmployees,
  projects,
  employees,
  onProjectChange,
  onEmployeeChange,
}: BulkTransferFiltersSectionProps) {
  const hasActiveFilters = useMemo(() => {
    const hasProjectFilter = selectedProjects.length > 0 && selectedProjects.length < projects.length;
    const hasEmployeeFilter = selectedEmployees.length > 0 && selectedEmployees.length < employees.length;
    return hasProjectFilter || hasEmployeeFilter;
  }, [selectedProjects.length, selectedEmployees.length, projects.length, employees.length]);

  const filterSummary = useMemo(() => {
    if (!hasActiveFilters) return null;
    const parts: string[] = [];
    if (selectedProjects.length > 0 && selectedProjects.length < projects.length) {
      parts.push(`${selectedProjects.length} dự án`);
    }
    if (selectedEmployees.length > 0 && selectedEmployees.length < employees.length) {
      parts.push(`${selectedEmployees.length} nhân viên`);
    }
    return parts.join(' · ');
  }, [hasActiveFilters, selectedProjects.length, selectedEmployees.length, projects.length, employees.length]);

  return (
    <Collapsible>
      <CollapsibleTrigger asChild>
        <button
          type="button"
          className={cn(
            "flex w-full items-center justify-between rounded-xl border border-dashed px-3 py-2 text-sm transition-colors group",
            hasActiveFilters
              ? "border-primary/30 bg-primary/5 text-primary hover:bg-primary/10"
              : "border-muted-foreground/25 text-muted-foreground hover:border-muted-foreground/40 hover:text-foreground"
          )}
        >
          <span className="flex items-center gap-2">
            <SlidersHorizontal className="w-3.5 h-3.5" />
            {hasActiveFilters ? (
              <span>Đã lọc: {filterSummary}</span>
            ) : (
              <span>Bộ lọc nâng cao</span>
            )}
          </span>
          <ChevronDown className="w-3.5 h-3.5 transition-transform duration-200 group-data-[state=open]:rotate-180" />
        </button>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-3">
          <div className="space-y-1.5">
            <Label className="text-xs">Dự án</Label>
            <ProjectMultiSelector
              value={selectedProjects}
              onChange={onProjectChange}
              projects={projects}
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs">Nhân viên</Label>
            <EmployeeMultiSelector
              value={selectedEmployees}
              onChange={onEmployeeChange}
              employees={employees}
            />
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
});
