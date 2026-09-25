import { useMemo } from 'react';
import { getAllHourTypes } from '@/components/payrates/types';
import { PayrateStructure, DayType } from '@/types/api/payrate.types';
import { MobileEmployeeCard } from './mobile/components/MobileEmployeeCard';

interface Employee {
  id: number;
  fullname: string;
  cccd: string;
  position: string;
  employee_code: string;
}

interface TimesheetEntry {
  employee_id: number;
  position: string;
  hour_entries: Record<string, number>;
  existingEntries?: Array<{
    id: number;
    hour_type: string;
    status: 'draft' | 'pending_approval' | 'approved' | 'rejected';
    paymentStatus?: 'pending' | 'paid' | 'failed' | 'cancelled';
  }>;
}

interface TimesheetEntryMobileProps {
  employees: Employee[];
  entries: TimesheetEntry[];
  onChange: (entries: TimesheetEntry[]) => void;
  payrateConfig?: PayrateStructure;
  dayType: DayType;
  isLoading?: boolean;
}

export function TimesheetEntryMobile({
  employees,
  entries,
  onChange,
  payrateConfig,
  dayType,
  isLoading = false
}: TimesheetEntryMobileProps) {
  // Available hour types from payrate config
  const availableHourTypes = useMemo(() => {
    if (!payrateConfig) return [];
    return getAllHourTypes(payrateConfig);
  }, [payrateConfig]);

  // Convert entries array to map for easier access
  const entriesMap = useMemo(() => {
    const map = new Map<number, TimesheetEntry>();

    // Initialize all employees with empty entries
    employees.forEach(emp => {
      const existingEntry = entries.find(e => e.employee_id === emp.id);
      map.set(emp.id, existingEntry ? {
        ...existingEntry,
        hour_entries: existingEntry.hour_entries || {}
      } : {
        employee_id: emp.id,
        position: emp.position,
        hour_entries: {}
      });
    });

    return map;
  }, [employees, entries]);


  // Get rate for position and hour type
  const getRate = (position: string, hourType: string): number => {
    if (!payrateConfig || !payrateConfig[position] || !payrateConfig[position][dayType]) {
      return 0;
    }
    return payrateConfig[position][dayType][hourType] || 0;
  };

  // Calculate total hours for an employee
  const calculateTotalHours = (entry: TimesheetEntry | undefined): number => {
    if (!entry || !entry.hour_entries) return 0;
    return Object.values(entry.hour_entries).reduce((sum, hours) => sum + (hours || 0), 0);
  };

  // Calculate total amount for an employee
  const calculateEmployeeTotal = (entry: TimesheetEntry | undefined): number => {
    if (!entry || !entry.hour_entries) return 0;

    let total = 0;
    Object.entries(entry.hour_entries).forEach(([hourType, hours]) => {
      const rate = getRate(entry.position, hourType);
      total += rate * hours;
    });
    return total;
  };


  // Update hour entry for an employee
  const updateHourEntry = (employeeId: number, hourType: string, hours: number) => {
    const entry = entriesMap.get(employeeId);
    if (!entry) return;

    const newHourEntries = {
      ...(entry.hour_entries || {}),
      [hourType]: hours
    };

    // Remove zero hours
    if (hours === 0) {
      delete newHourEntries[hourType];
    }

    const updatedEntry = {
      ...entry,
      hour_entries: newHourEntries
    };

    entriesMap.set(employeeId, updatedEntry);
    const newEntries = Array.from(entriesMap.values());

    onChange(newEntries);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <div className="animate-spin rounded-full h-8 w-8 border-2 border-blue-500 border-t-transparent" />
      </div>
    );
  }

  if (!payrateConfig) {
    return (
      <div className="mx-2 sm:mx-4 p-2 sm:p-3 border border-red-200 bg-red-50 rounded-xl">
        <div className="flex items-center gap-2">
          <div className="h-4 w-4 text-red-600 flex-shrink-0">⚠️</div>
          <div className="typography-body-medium text-red-700">
            Dự án chưa có cấu hình lương. Vui lòng thiết lập cấu hình lương trước khi nhập bảng công.
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-2 sm:space-y-3 px-2 sm:px-0">

      {employees.map((employee) => {
        const entry = entriesMap.get(employee.id);
        const totalHours = calculateTotalHours(entry);
        const hourStatus = (() => {
          if (totalHours > 16) return 'excessive'; // > 16 hours: red
          if (totalHours > 12) return 'exceeded'; // > 12 hours: orange
          return 'normal';
        })();

        return (
          <MobileEmployeeCard
            key={employee.id}
            employee={employee}
            entry={entry}
            hourStatus={hourStatus}
            availableHourTypes={availableHourTypes}
            onHourEntryUpdate={updateHourEntry}
            getRate={getRate}
            calculateTotalHours={calculateTotalHours}
            calculateEmployeeTotal={calculateEmployeeTotal}
          />
        );
      })}

      {employees.length === 0 && (
        <div className="text-center py-12 mx-2 sm:mx-4">
          <div className="text-gray-500 typography-display-medium mb-4">👥</div>
          <h3 className="typography-title-large text-foreground">Không có nhân viên</h3>
          <p className="typography-body-medium text-muted-foreground mt-2">
            Chưa có nhân viên nào trong dự án này.
          </p>
        </div>
      )}
    </div>
  );
}
