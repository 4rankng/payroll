import { memo, useMemo, useState, useCallback } from 'react';
import { cn } from '@/lib/utils';
import { UserAvatar } from '@/components/ui/user-avatar';
import { Table, TableBody, TableHead, TableHeader as ShadcnTableHeader, TableRow } from '@/components/ui/table';
import { AlertTriangle, ChevronDown } from 'lucide-react';
import { EntryRow } from './EntryRow';
import type { TimesheetEntry } from '../types/multi-timesheet.types';
import type { Employee } from '@/types/api/employee.types';
import { formatCurrency as formatVND } from '@/utils/formatters';
import { calculateTimesheetPreviewAmount } from '@/components/timesheet/components/timesheet-pay-unit';

interface GroupedEntryCardProps {
  employee: Employee;
  entries: TimesheetEntry[];
  entryValidations: Record<string, { valid: boolean; errors: string[] }>;
  getDayTypesForPosition: (position: string) => string[];
  getHourTypesForEntry: (position: string, dayType: string) => string[];
  hasPayRateForEntry: (position: string, dayType: string, hourType: string) => boolean;
  getPayRateForEntry: (position: string, dayType: string, hourType: string) => number;
  isFlexibleProject: boolean;
  onEntryChange: (entryIndex: number, field: keyof TimesheetEntry, value: unknown) => void;
  onRemoveEntry: (entryIndex: number) => void;
  onPreviewRequest?: () => void;
  isLoading?: boolean;
  previewErrors?: string[] | null;
  getErrorsForRow?: (date: string) => string[] | null;
}

export const GroupedEntryCard = memo(({
  employee,
  entries,
  entryValidations,
  getDayTypesForPosition,
  getHourTypesForEntry,
  hasPayRateForEntry,
  getPayRateForEntry,
  isFlexibleProject,
  onEntryChange,
  onRemoveEntry,
  onPreviewRequest,
  isLoading = false,
  previewErrors,
  getErrorsForRow
}: GroupedEntryCardProps) => {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const handleToggleCollapse = useCallback(() => {
    setIsCollapsed(prev => !prev);
  }, []);

  const totalHours = useMemo(() => {
    return entries.reduce((sum, entry) => {
      if (!entry.hours) return sum;
      // Sum all hours from the hours object
      const entryTotalHours = Object.values(entry.hours).reduce((hourSum, hours) => hourSum + hours, 0);
      return sum + entryTotalHours;
    }, 0);
  }, [entries]);

  // Calculate total earnings for the employee
  const totalEarnings = useMemo(() => {
    return entries.reduce((sum, entry) => {
      if (!entry.hours || !entry.position || !entry.dayType) return sum;

      let entryEarnings = 0;
      const hours = entry.hours;

      // Calculate earnings for each hour type
      for (const [hourType, hoursWorked] of Object.entries(hours)) {
        if (hoursWorked > 0) {
          const rate = getPayRateForEntry(entry.position!, entry.dayType!, hourType);
          entryEarnings += calculateTimesheetPreviewAmount(
            rate,
            hoursWorked,
            isFlexibleProject,
          );
        }
      }

      return sum + entryEarnings;
    }, 0);
  }, [entries, getPayRateForEntry, isFlexibleProject]);

  // formatVND is imported from shared utils

  const workDays = useMemo(() => {
    const workDates = new Set<string>();
    entries.forEach(entry => {
      if (entry.hours && Object.values(entry.hours).some(hours => hours > 0)) {
        workDates.add(entry.date);
      }
    });
    return workDates.size;
  }, [entries]);

  const hasErrors = useMemo(() => {
    return (previewErrors && previewErrors.length > 0) ||
      entries.some(entry => {
        const validation = entryValidations[entry.id];
        return validation && !validation.valid && validation.errors.length > 0;
      });
  }, [entries, entryValidations, previewErrors]);

  return (
    <div className={cn(
      "border rounded-xl bg-card overflow-hidden shadow-[0_2px_8px_rgba(0,0,0,0.06)]",
      hasErrors ? 'border-destructive/40' : 'border-border/70'
    )}>
      {/* Employee Header — clickable to toggle collapse */}
      <button
        type="button"
        onClick={handleToggleCollapse}
        className="flex items-center justify-between gap-3 px-4 py-3 border-b border-border/60 bg-muted/50 w-full text-left cursor-pointer hover:bg-muted/70 transition-colors"
        aria-expanded={!isCollapsed}
        aria-label={isCollapsed ? `Mở rộng ${employee.fullname}` : `Thu gọn ${employee.fullname}`}
      >
        <div className="flex items-center gap-2.5 min-w-0 flex-1">
          <UserAvatar name={employee.fullname} size="sm" className="shrink-0" />
          <div>
            <span className="text-sm font-semibold text-foreground block">{employee.fullname}</span>
            <span className="text-xs text-muted-foreground">{employee.cccd}</span>
          </div>
          {totalEarnings > 0 && (
            <div className={cn(
              "flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium shrink-0 ml-1",
              totalEarnings > 3500000
                ? "bg-red-100 text-red-600"
                : "bg-emerald-100 text-emerald-700"
            )}>
              {totalEarnings > 3500000 && <AlertTriangle className="h-3 w-3" />}
              <span className="tabular-nums">{formatVND(totalEarnings)}</span>
            </div>
          )}
          {isCollapsed && (
            <span className="text-xs text-muted-foreground shrink-0">{entries.length} mục</span>
          )}
        </div>
        <div className="flex items-center gap-2.5 shrink-0">
          <div className="flex items-center gap-2">
            <div className="text-right">
              <div className="text-xs font-semibold text-foreground tabular-nums leading-none">{workDays}</div>
              <div className="text-xs text-muted-foreground leading-none mt-0.5">ngày</div>
            </div>
            <div className="w-px h-5 bg-border shrink-0" />
            <div className="text-right">
              <div className="text-xs font-semibold text-foreground tabular-nums leading-none">{totalHours}h</div>
              <div className="text-xs text-muted-foreground leading-none mt-0.5">giờ</div>
            </div>
          </div>
          <ChevronDown className={cn(
            "h-4 w-4 text-muted-foreground transition-transform duration-200",
            isCollapsed && "-rotate-90"
          )} />
        </div>
      </button>

      {/* Per-employee preview error banner — only show errors without a specific date */}
      {previewErrors && previewErrors.length > 0 && (() => {
        // Only show errors that don't have a row-level match (date-less errors)
        const hasRowErrors = getErrorsForRow !== undefined;
        const globalErrors = hasRowErrors
          ? previewErrors.filter(msg => {
              // If any entry date has this error, it's a row-level error — skip here
              return !entries.some(e => e.date && getErrorsForRow!(e.date)?.includes(msg));
            })
          : previewErrors;
        if (globalErrors.length === 0) return null;
        return (
          <div className="px-3 py-2 bg-red-50 border-b border-red-100 flex items-start gap-2">
            <AlertTriangle className="h-3.5 w-3.5 text-red-600 shrink-0 mt-0.5" />
            <div className="space-y-0.5">
              {globalErrors.map((err, i) => (
                <p key={i} className="text-xs text-red-600">{err}</p>
              ))}
            </div>
          </div>
        );
      })()}

      {/* Table — hidden when collapsed.
          Note: only force the horizontal cell padding here. Vertical padding
          is owned by EntryRow / the header cells below so each date row gets
          breathing room (was previously `[&_td]:py-0` which crushed every row
          against its top/bottom border). */}
      {!isCollapsed && (
        <Table className="w-full [&_td]:px-2 [&_th]:px-2">
          <ShadcnTableHeader>
            <TableRow className="border-b border-border/40 bg-muted/30 hover:bg-muted/30">
              <TableHead className="w-[76px] text-xs font-semibold text-muted-foreground uppercase tracking-widest py-2">
                Ngày
              </TableHead>
              <TableHead className="w-px whitespace-nowrap text-xs font-semibold text-muted-foreground uppercase tracking-widest py-2">
                Loại ngày
              </TableHead>
              <TableHead className="text-xs font-semibold text-muted-foreground uppercase tracking-widest py-2">
                Ca làm việc
              </TableHead>
            </TableRow>
          </ShadcnTableHeader>
          <TableBody>
            {entries.map((entry, entryIndex) => (
              <EntryRow
                key={entry.id}
                entry={entry}
                entryIndex={entryIndex}
                entryValidation={entryValidations[entry.id]}
                getDayTypesForPosition={getDayTypesForPosition}
                getHourTypesForEntry={getHourTypesForEntry}
                hasPayRateForEntry={hasPayRateForEntry}
                getPayRateForEntry={getPayRateForEntry}
                onEntryChange={onEntryChange}
                onRemoveEntry={onRemoveEntry}
                onPreviewRequest={onPreviewRequest}
                isLoading={isLoading}
                hideEmployeeSelector={true}
                rowErrors={getErrorsForRow && entry.date ? getErrorsForRow(entry.date) : null}
              />
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
});

GroupedEntryCard.displayName = 'GroupedEntryCard';
