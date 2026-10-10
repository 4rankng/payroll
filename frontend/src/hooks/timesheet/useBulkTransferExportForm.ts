import { useState, useCallback, useMemo, useEffect, useRef } from 'react';
import { format } from 'date-fns';
import { getAvailableWeekPeriods, formatWeekPeriodDisplay, type WeekPeriod, type WeekPeriodsResult } from '@/utils/weekPeriodHelpers';
import { getAvailableMonthPeriods, type MonthPeriod, type MonthPeriodsResult } from '@/utils/monthPeriodHelpers';
import { getCustomDateRanges, type CustomDateRange } from '@/utils/weekPeriodHelpers';
import { formatDateForAPI } from '@/utils/formatters';
import type { Project } from '@/types/api/project.types';

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
  initialProjectIds?: number[];
}

export function getDefaultWeeklyDates(now: Date = new Date()) {
  const currentWeekPeriods = getAvailableWeekPeriods(now);
  const currentCustomRanges = getCustomDateRanges(now);

  if (currentCustomRanges.length > 0) {
    let defaultRange;

    if (now.getDate() >= 28) {
      // Late month: prep the ky 4 transfer (day 22 to month end) that pays on
      // day 1 of the next month. Select by index — the label's end day varies
      // with the month length, and on day >= 28 the dialog only renders these
      // custom-range buttons.
      defaultRange = currentCustomRanges[1];
    } else if (now.getDate() <= 10) {
      // 01-07 current month is closer to today than 22-end-of-month previous month
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

  // Display text for selectors
  const projectDisplayText = useMemo(() => {
    if (selectedProjects.length === 0 || selectedProjects.length === projects.length) {
      return 'Tất cả dự án';
    }
    return `Đã chọn ${selectedProjects.length} dự án`;
  }, [selectedProjects.length, projects.length]);

  const employeeDisplayText = useMemo(() => {
    if (selectedEmployees.length === 0) {
      return 'Tất cả nhân viên';
    }
    return `Đã chọn ${selectedEmployees.length} nhân viên`;
  }, [selectedEmployees.length]);

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

    if (selectedEmployees.length > 0) {
      params.employee_ids = selectedEmployees;
    } else {
      params.employee_ids = [];
    }

    return params;
  }, [canExport, fromDate, toDate, paymentSchedule, selectedProjects, selectedEmployees, projects.length]);

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
    buildExportParams
  };
}
