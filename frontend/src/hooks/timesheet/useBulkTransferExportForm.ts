import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { getAvailableWeekPeriods, formatWeekPeriodDisplay, type WeekPeriod, type WeekPeriodsResult } from '@/utils/weekPeriodHelpers';
import { getAvailableMonthPeriods, type MonthPeriod, type MonthPeriodsResult } from '@/utils/monthPeriodHelpers';
import { getCustomDateRanges, type CustomDateRange } from '@/utils/weekPeriodHelpers';
import { formatDateForAPI } from '@/utils/formatters';
import type { Project } from '@/types/api/project.types';
import type { Employee } from '@/types/api/employee.types';

export interface BulkTransferExportParams {
  project_ids?: number[];
  employee_ids?: number[];
  fromDate?: string;
  toDate?: string;
  for_month?: string;
}

interface UseBulkTransferExportFormProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  projects: Project[];
  employees: Employee[];
  initialProjectIds?: number[];
}

function getDefaultWeeklyDates() {
  const currentWeekPeriods = getAvailableWeekPeriods();
  const currentCustomRanges = getCustomDateRanges();

  if (currentCustomRanges.length > 0) {
    const today = new Date();
    const monthNumber = format(today, 'MM', { locale: vi });
    let defaultRange;

    if (today.getDate() >= 28) {
      defaultRange = currentCustomRanges.find(r => r.label === `22 - 28 Tháng ${monthNumber}`);
    } else if (today.getDate() <= 10) {
      // 01-07 current month is closer to today than 22-28 previous month
      defaultRange = currentCustomRanges[1];
    }

    if (defaultRange) {
      return { from: new Date(defaultRange.from), to: new Date(defaultRange.to), selectedCustomRange: defaultRange.label };
    }
  }

  return {
    from: new Date(currentWeekPeriods.defaultPeriod.from),
    to: new Date(currentWeekPeriods.defaultPeriod.to),
    selectedCustomRange: ''
  };
}

export function useBulkTransferExportForm({
  isOpen,
  onOpenChange,
  projects,
  employees,
  initialProjectIds,
}: UseBulkTransferExportFormProps) {
  const [paymentSchedule, setPaymentScheduleState] = useState<'weekly' | 'monthly'>('weekly');
  const [fromDate, setFromDate] = useState<Date>();
  const [toDate, setToDate] = useState<Date>();
  const [selectedProjects, setSelectedProjects] = useState<number[]>([]);
  const [selectedEmployees, setSelectedEmployees] = useState<number[]>([]);
  const [selectedCustomRange, setSelectedCustomRange] = useState<string>('');

  const prevIsOpen = useRef(false);
  const initialProjectIdsRef = useRef(initialProjectIds);
  initialProjectIdsRef.current = initialProjectIds;

  // Period calculations
  const weekPeriods = useMemo(() => getAvailableWeekPeriods(), []);
  const monthPeriods = useMemo(() => getAvailableMonthPeriods(), []);
  const customDateRanges = useMemo(() => getCustomDateRanges(), []);

  // Initialize/reset dates when dialog opens/closes
  useEffect(() => {
    if (isOpen && !prevIsOpen.current) {
      if (paymentSchedule === 'weekly') {
        const defaults = getDefaultWeeklyDates();
        setFromDate(defaults.from);
        setToDate(defaults.to);
        setSelectedCustomRange(defaults.selectedCustomRange);
      } else {
        const currentMonthPeriods = getAvailableMonthPeriods();
        setFromDate(new Date(currentMonthPeriods.defaultPeriod.from));
        setToDate(new Date(currentMonthPeriods.defaultPeriod.to));
        setSelectedCustomRange('');
      }
      setSelectedProjects(initialProjectIdsRef.current ?? []);
    } else if (!isOpen && prevIsOpen.current) {
      setFromDate(undefined);
      setToDate(undefined);
      setSelectedProjects([]);
      setSelectedEmployees([]);
      setPaymentScheduleState('weekly');
      setSelectedCustomRange('');
    }
    prevIsOpen.current = isOpen;
  }, [isOpen, paymentSchedule]);

  // Handle dialog open/close from Radix (user interactions like overlay click, escape)
  const handleDialogOpen = useCallback((open: boolean) => {
    onOpenChange(open);
  }, [onOpenChange]);

  // Period handlers
  const applyWeekPeriod = useCallback((period: WeekPeriod) => {
    setFromDate(new Date(period.from));
    setToDate(new Date(period.to));
    setSelectedCustomRange('');
  }, []);

  const applyMonthPeriod = useCallback((period: MonthPeriod) => {
    setFromDate(new Date(period.from));
    setToDate(new Date(period.to));
  }, []);

  const applyCustomRange = useCallback((rangeLabel: string) => {
    const range = customDateRanges.find(r => r.label === rangeLabel);
    if (range) {
      setFromDate(new Date(range.from));
      setToDate(new Date(range.to));
      setSelectedCustomRange(rangeLabel);
    }
  }, [customDateRanges]);

  // Handle payment schedule change
  const handlePaymentScheduleChange = useCallback((value: string) => {
    setPaymentScheduleState(value as 'weekly' | 'monthly');

    if (value === 'weekly') {
      const defaults = getDefaultWeeklyDates();
      setFromDate(defaults.from);
      setToDate(defaults.to);
      setSelectedCustomRange(defaults.selectedCustomRange);
    } else {
      const currentMonthPeriods = getAvailableMonthPeriods();
      setFromDate(new Date(currentMonthPeriods.defaultPeriod.from));
      setToDate(new Date(currentMonthPeriods.defaultPeriod.to));
      setSelectedCustomRange('');
    }
  }, []);

  // Selection handlers
  const handleProjectSelect = useCallback((projectId: number) => {
    setSelectedProjects(prev =>
      prev.includes(projectId)
        ? prev.filter(id => id !== projectId)
        : [...prev, projectId]
    );
  }, []);

  const handleProjectSelectAll = useCallback(() => {
    setSelectedProjects(prev =>
      prev.length === projects.length ? [] : projects.map(p => p.id)
    );
  }, [projects]);

  const handleEmployeeSelect = useCallback((employeeId: number) => {
    setSelectedEmployees(prev =>
      prev.includes(employeeId)
        ? prev.filter(id => id !== employeeId)
        : [...prev, employeeId]
    );
  }, []);

  const handleEmployeeSelectAll = useCallback(() => {
    setSelectedEmployees(prev =>
      prev.length === employees.length ? [] : employees.map(e => e.id)
    );
  }, [employees]);

  // Display text for selectors
  const projectDisplayText = useMemo(() => {
    if (selectedProjects.length === 0 || selectedProjects.length === projects.length) {
      return 'Tất cả dự án';
    }
    return `Đã chọn ${selectedProjects.length} dự án`;
  }, [selectedProjects.length, projects.length]);

  const employeeDisplayText = useMemo(() => {
    if (selectedEmployees.length === 0 || selectedEmployees.length === employees.length) {
      return 'Tất cả nhân viên';
    }
    return `Đã chọn ${selectedEmployees.length} nhân viên`;
  }, [selectedEmployees.length, employees.length]);

  // Validation
  const canExport = useMemo(() => {
    return !!(fromDate && toDate && fromDate <= toDate);
  }, [fromDate, toDate]);

  const hasDateError = useMemo(() => {
    return !!(fromDate && toDate && fromDate > toDate);
  }, [fromDate, toDate]);

  // Handle close
  const handleClose = useCallback(() => {
    onOpenChange(false);
  }, [onOpenChange]);

  // Build export params
  const buildExportParams = useCallback((): BulkTransferExportParams => {
    if (!fromDate || !toDate) {
      console.error('Export failed: dates are not set', { fromDate, toDate });
      return {};
    }

    if (!canExport) {
      console.error('Export failed: validation failed', { fromDate, toDate, canExport });
      return {};
    }

    const params: BulkTransferExportParams = {};

    const validFromDate = fromDate instanceof Date ? fromDate : new Date(fromDate);
    const validToDate = toDate instanceof Date ? toDate : new Date(toDate);

    if (paymentSchedule === 'weekly') {
      params.fromDate = formatDateForAPI(validFromDate);
      params.toDate = formatDateForAPI(validToDate);
    } else {
      params.for_month = format(validToDate, 'yyyy-MM');
    }

    if (
      selectedProjects.length > 0 &&
      (projects.length === 0 || selectedProjects.length < projects.length)
    ) {
      params.project_ids = selectedProjects;
    } else if (selectedProjects.length === 0 || selectedProjects.length === projects.length) {
      params.project_ids = [];
    }

    if (selectedEmployees.length > 0 && selectedEmployees.length < employees.length) {
      params.employee_ids = selectedEmployees;
    } else if (selectedEmployees.length === 0 || selectedEmployees.length === employees.length) {
      params.employee_ids = [];
    }

    return params;
  }, [canExport, fromDate, toDate, paymentSchedule, selectedProjects, selectedEmployees, projects.length, employees.length]);

  return {
    paymentSchedule,
    fromDate,
    toDate,
    selectedProjects,
    selectedEmployees,
    weekPeriods,
    monthPeriods,
    customDateRanges,
    selectedCustomRange,
    canExport,
    hasDateError,
    projectDisplayText,
    employeeDisplayText,
    setPaymentSchedule: handlePaymentScheduleChange,
    setFromDate,
    setToDate,
    setSelectedProjects,
    setSelectedEmployees,
    handleDialogOpen,
    handleClose,
    applyWeekPeriod,
    applyMonthPeriod,
    applyCustomRange,
    handleProjectSelect,
    handleProjectSelectAll,
    handleEmployeeSelect,
    handleEmployeeSelectAll,
    buildExportParams
  };
}
