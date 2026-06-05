import { useState, useMemo, useEffect, useCallback } from 'react';
import { DayType, PayrateStructure } from '@/types/api/payrate.types';
import { NewTimesheetEntry } from '@/types/api/timesheet.types';
import { useProjectPayRates } from '@/hooks/api/usePayRates';
import { getDayTypeOrNull, isSaturdayRequiringDayType } from '@/utils/dateHelpers';
import { timesheetService } from '@/services/api/timesheet.service';
import { useProjects } from '@/hooks/api/useProjects';
import { useProjectEmployees } from '@/hooks/api/useProjectEmployees';
import { useTimesheetsByProjectAndDate } from '@/hooks/api/useTimesheets';


import { Project } from '@/types/api/project.types';

interface TimesheetEntry {
  employee_id: number;
  position: string;
  hour_entries: Record<string, number>; // hour_type -> hours_worked
  existingEntries?: Array<{ // Store existing timesheet data for status display
    id: number;
    hour_type: string;
    status: 'draft' | 'pending_approval' | 'approved' | 'rejected';
    paymentStatus?: 'pending' | 'paid' | 'failed' | 'cancelled';
  }>;
}

interface ValidationError {
  employeeId: number;
  totalHours: number;
  message: string;
}

interface TimesheetSummary {
  totalEmployees: number;
  totalHours: number;
  entriesCompleted: number;
  totalCost: number;
  averageHours: number;
  validationErrors: ValidationError[];
}


export function useTimesheetEntry() {
  const [selectedDate, setSelectedDate] = useState<Date>(new Date());
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [entries, setEntries] = useState<TimesheetEntry[]>([]);
  const [validationErrors, setValidationErrors] = useState<ValidationError[]>([]);
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saturdayDayType, setSaturdayDayType] = useState<DayType | null>(null);
  const [showSaturdayModal, setShowSaturdayModal] = useState(false);

  // Fetch real data from APIs
  const { data: projectsData, isLoading: isLoadingProjects } = useProjects();
  const { data: employeesData, isLoading: isLoadingEmployees } = useProjectEmployees(selectedProject?.id || 0, {}, !!selectedProject);
  const { data: payRateData, isLoading: isLoadingPayRate } = useProjectPayRates(selectedProject?.id || 0, {}, !!selectedProject);

  // Format date to YYYY-MM-DD for API
  const formattedDate = selectedDate.toISOString().split('T')[0];

  // Fetch existing timesheets for the selected date and project
  const { data: existingTimesheets, isLoading: isLoadingExisting } = useTimesheetsByProjectAndDate(
    selectedProject?.id || 0,
    formattedDate,
    !!selectedProject
  );

  const projects = projectsData?.data || [];

  // Transform project-employee assignment data to Employee format expected by TimesheetEntryTable
  const employees = useMemo(() => {
    const assignmentsData = employeesData?.data || [];
    return assignmentsData.map(assignment => ({
      id: assignment.employee_id,
      fullname: assignment.employee_name,
      cccd: assignment.employee_cccd,
      position: assignment.position,
      employee_code: assignment.employee_code
    }));
  }, [employeesData]);

  const payrateConfig = useMemo(() => {
    if (payRateData?.data && Array.isArray(payRateData.data) && payRateData.data.length > 0) {
      // Get the most recent active payrate from the list
      const today = new Date().toISOString().split('T')[0];
      const activePayrate = payRateData.data.find(payrate => {
        const fromDate = payrate.fromDate;
        const toDate = payrate.toDate;
        return fromDate <= today && (!toDate || toDate >= today);
      });

      // If no active payrate, use the most recent one
      const payrate = activePayrate || payRateData.data[0];
      return payrate?.rates;
    }
    return undefined;
  }, [payRateData]);

  // Determine day type for selected date
  const dayType = useMemo(() => {
    const dateStr = selectedDate.toISOString().split('T')[0];

    // Check if it's Saturday requiring user input
    if (isSaturdayRequiringDayType(dateStr)) {
      return saturdayDayType;
    }

    // Auto-determine for other days
    return getDayTypeOrNull(dateStr);
  }, [selectedDate, saturdayDayType]);

  // Show Saturday modal when date changes to Saturday
  useEffect(() => {
    const dateStr = selectedDate.toISOString().split('T')[0];
    if (isSaturdayRequiringDayType(dateStr) && !saturdayDayType) {
      setShowSaturdayModal(true);
    }
  }, [selectedDate, saturdayDayType]);

  // Populate entries from existing timesheets when data is loaded
  useEffect(() => {
    if (existingTimesheets?.data && employees.length > 0 && !isLoadingExisting) {
      const existingTimesheetsData = existingTimesheets.data;

      // Create a map of employee entries with their existing timesheet data
      const entriesMap = new Map<number, TimesheetEntry>();

      // Initialize all employees with empty entries
      employees.forEach(employee => {
        entriesMap.set(employee.id, {
          employee_id: employee.id,
          position: employee.position,
          hour_entries: {},
          existingEntries: []
        });
      });

      // Populate existing data and status information
      existingTimesheetsData.forEach(timesheet => {
        const existingEntry = entriesMap.get(timesheet.employee_id);
        if (existingEntry) {
          existingEntry.hour_entries[timesheet.hour_type] = timesheet.hours_worked;

          // Add status information for this timesheet entry
          if (!existingEntry.existingEntries) {
            existingEntry.existingEntries = [];
          }
          existingEntry.existingEntries.push({
            id: timesheet.id,
            hour_type: timesheet.hour_type,
            status: timesheet.status,
            paymentStatus: undefined // Will be populated when we have payroll data
          });
        }
      });

      // Convert map to array and update state
      const newEntries = Array.from(entriesMap.values());
      setEntries(newEntries);
      setHasChanges(false); // Existing data shouldn't count as changes
    }
  }, [existingTimesheets, employees, isLoadingExisting]);

  // Calculate rate for an employee's position and hour type
  const getRate = useCallback((position: string, hourType: string, payrates?: PayrateStructure): number => {
    if (!payrates || !payrates[position] || !dayType || !payrates[position][dayType]) {
      return 0;
    }
    return payrates[position][dayType][hourType] || 0;
  }, [dayType]);

  // Calculate total cost for an entry
  const calculateEntryTotal = useCallback((entry: TimesheetEntry, payrates?: PayrateStructure): number => {
    if (!entry || !entry.hour_entries || !payrates) return 0;

    let total = 0;
    Object.entries(entry.hour_entries).forEach(([hourType, hours]) => {
      const rate = getRate(entry.position, hourType, payrates);
      total += rate * hours;
    });
    return total;
  }, [getRate]);

  const summary: TimesheetSummary = useMemo(() => {
    const totalEmployees = employees.length;
    const totalHours = entries.reduce((sum, entry) => {
      if (!entry.hour_entries) return sum;
      return sum + Object.values(entry.hour_entries).reduce((h, hours) => h + hours, 0);
    }, 0);
    const entriesCompleted = entries.filter(entry => {
      if (!entry.hour_entries) return false;
      return Object.values(entry.hour_entries).some(hours => hours > 0);
    }).length;

    const totalCost = entries.reduce((sum, entry) => {
      return sum + calculateEntryTotal(entry, payrateConfig);
    }, 0);

    const averageHours = totalEmployees > 0 ? totalHours / totalEmployees : 0;

    return {
      totalEmployees,
      totalHours,
      entriesCompleted,
      totalCost,
      averageHours,
      validationErrors
    };
  }, [employees.length, entries, validationErrors, payrateConfig, calculateEntryTotal]);

  const handleEntriesChange = (newEntries: TimesheetEntry[], newValidationErrors: ValidationError[]) => {
    setEntries(newEntries);
    setValidationErrors(newValidationErrors);
    setHasChanges(true);
  };

  const handleSave = async () => {
    if (!selectedProject) {
      throw new Error('Vui lòng chọn dự án');
    }

    if (!dayType) {
      throw new Error('Vui lòng xác định loại ngày cho Thứ 7');
    }

    if (validationErrors.length > 0) {
      throw new Error(`Không thể lưu: Có ${validationErrors.length} nhân viên vượt quá giới hạn 24 giờ/ngày. Vui lòng điều chỉnh trước khi lưu.`);
    }

    setIsSaving(true);
    try {
      // Convert entries to new API format
      const newTimesheetEntries: NewTimesheetEntry[] = [];
      const dateStr = selectedDate.toISOString().split('T')[0];

      entries.forEach(entry => {
        if (!entry.hour_entries) return;

        Object.entries(entry.hour_entries).forEach(([hourType, hours]) => {
          if (hours > 0) {
            const newEntry: NewTimesheetEntry = {
              projectId: selectedProject.id,
              employeeId: entry.employee_id,
              date: dateStr,
              hoursWorked: hours,
              hourType: hourType
            };

            // Add dayType only for Saturdays
            if (isSaturdayRequiringDayType(dateStr)) {
              newEntry.dayType = dayType === 'ngày lễ' ? 'Ngày lễ'
                : dayType === 'ngày thường' ? 'Ngày thường' : 'Ngày nghỉ';
            }

            newTimesheetEntries.push(newEntry);
          }
        });
      });

      if (newTimesheetEntries.length === 0) {
        throw new Error('Vui lòng nhập ít nhất một giờ làm việc');
      }

      // Make API call with new format
      const result = await timesheetService.createTimesheets(newTimesheetEntries);

      setHasChanges(false);
      return {
        success: result?.status === 'success',
        entriesCount: result?.data?.total_created || 0,
        failedCount: result?.data?.total_failed || 0,
        errors: result?.data?.failed || []
      };
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancel = () => {
    setEntries([]);
    setHasChanges(false);
  };

  const handleSaturdayDayTypeConfirm = (selectedDayType: DayType) => {
    setSaturdayDayType(selectedDayType);
    setShowSaturdayModal(false);
  };

  const handleDateChange = (date: Date) => {
    setSelectedDate(date);

    // Reset Saturday day type when date changes
    const dateStr = date.toISOString().split('T')[0];
    if (!isSaturdayRequiringDayType(dateStr)) {
      setSaturdayDayType(null);
    }
  };

  return {
    // State
    selectedDate,
    selectedProject,
    entries,
    validationErrors,
    hasChanges,
    isSaving,
    dayType,
    showSaturdayModal,
    isLoadingPayRate: isLoadingPayRate || isLoadingProjects || isLoadingEmployees || isLoadingExisting,

    // Data
    projects,
    employees,
    summary,
    payrateConfig,

    // Actions
    setSelectedDate: handleDateChange,
    setSelectedProject,
    handleEntriesChange,
    handleSave,
    handleCancel,
    handleSaturdayDayTypeConfirm,
    setShowSaturdayModal,
    setSaturdayDayType
  };
}
