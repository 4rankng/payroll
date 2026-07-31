import { useMemo, memo } from 'react';
import { GroupedEntryCard } from './GroupedEntryCard';
import type { TimesheetEntry, TimesheetValidation } from '../types/multi-timesheet.types';
import type { Employee } from '@/types/api/employee.types';

interface MultiEntryTableProps {
  entries: TimesheetEntry[];
  availableEmployees: Employee[];
  validation: TimesheetValidation;
  getDayTypesForPosition: (position: string) => string[];
  getHourTypesForEntry: (position: string, dayType: string) => string[];
  hasPayRateForEntry: (position: string, dayType: string, hourType: string) => boolean;
  getPayRateForEntry: (position: string, dayType: string, hourType: string) => number;
  isFlexibleProject: boolean;
  onEntryChange: (index: number, field: keyof TimesheetEntry, value: unknown) => void;
  onAddEntry: (employee?: Employee) => void;
  onRemoveEntry: (index: number) => void;
  onPreviewRequest?: () => void;
  isLoading?: boolean;
  filteredEmployeeId?: number | null;
  getErrorsForEmployee?: (employeeId: number) => string[] | null;
  getErrorsForRow?: (employeeId: number, date: string) => string[] | null;
}

export const MultiEntryTable = memo(function MultiEntryTable({
  entries,
  availableEmployees,
  validation,
  getDayTypesForPosition,
  getHourTypesForEntry,
  hasPayRateForEntry,
  getPayRateForEntry,
  isFlexibleProject,
  onEntryChange,
  onAddEntry,
  onRemoveEntry,
  onPreviewRequest,
  isLoading = false,
  filteredEmployeeId,
  getErrorsForEmployee,
  getErrorsForRow
}: MultiEntryTableProps) {
  // Group entries by employee
  const groupedEntries = useMemo(() => {
    const groups: Record<number, { employee: Employee | null; entries: { entry: TimesheetEntry; originalIndex: number }[] }> = {};

    // Process all entries (no filtering needed since auto-population handles date ranges)
    entries.forEach((entry, originalIndex) => {
      // Find the actual original index from the unfiltered entries array
      const actualIndex = entries.indexOf(entry);

      if (entry.employeeId && entry.employee) {
        // Filter by employee if filteredEmployeeId is set
        if (filteredEmployeeId && entry.employeeId !== filteredEmployeeId) {
          return; // Skip this entry if it doesn't match the filter
        }

        // Entries with assigned employees
        if (!groups[entry.employeeId]) {
          groups[entry.employeeId] = {
            employee: entry.employee,
            entries: []
          };
        }
        groups[entry.employeeId].entries.push({ entry, originalIndex: actualIndex });
      }
    });

    return Object.values(groups);
  }, [entries, filteredEmployeeId]);


  return (
    <div className="columns-1 lg:columns-2 gap-4">
        {groupedEntries.map((group) => {
          // Skip blank entries (no employee assigned yet)
          if (!group.employee) {
            return null;
          }

          // Handle entries with assigned employees - show GroupedEntryCard
          return (
            <div key={group.employee.id} className="break-inside-avoid mb-4">
              <GroupedEntryCard
                employee={group.employee}
                entries={group.entries.map(e => e.entry)}
                entryValidations={validation.entryValidations}
                getDayTypesForPosition={getDayTypesForPosition}
                getHourTypesForEntry={getHourTypesForEntry}
                hasPayRateForEntry={hasPayRateForEntry}
                getPayRateForEntry={getPayRateForEntry}
                isFlexibleProject={isFlexibleProject}
                onEntryChange={(entryIndex, field, value) => {
                  const groupEntry = group.entries[entryIndex];
                  if (groupEntry) onEntryChange(groupEntry.originalIndex, field, value);
                }}
                onRemoveEntry={(entryIndex) => {
                  const groupEntry = group.entries[entryIndex];
                  if (groupEntry) onRemoveEntry(groupEntry.originalIndex);
                }}
                onPreviewRequest={onPreviewRequest}
                isLoading={isLoading}
                previewErrors={getErrorsForEmployee ? getErrorsForEmployee(group.employee.id) : null}
                getErrorsForRow={getErrorsForRow ? (date) => getErrorsForRow(group.employee!.id, date) : undefined}
              />
            </div>
          );
        })}
    </div>
  );
});

MultiEntryTable.displayName = 'MultiEntryTable';
