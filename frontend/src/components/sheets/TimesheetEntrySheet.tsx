import { memo, useCallback, useEffect, useState, useMemo, useRef } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Sheet, SheetContent, SheetClose } from '@/components/ui/sheet';
import { Button } from '@/components/ui/button';
import { X, AlertTriangle } from 'lucide-react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command';
import { ChevronsUpDown, Check } from 'lucide-react';
import { cn } from '@/lib/utils';
import { formatCurrency } from '@/utils/formatters';
import { NewTimesheetEntry } from '@/types/api/timesheet.types';
import { useCreateTimesheets } from '@/hooks/api/useTimesheets';
import { useTimesheetProjects } from '@/hooks/api/useProjects';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import type { ModalConfig } from '@/types/modal-config.types';
import { useMultiTimesheetFormState } from './timesheet-entry/hooks/useMultiTimesheetFormState';
import { MultiEntryTable } from './timesheet-entry/components/MultiEntryTable';
import { getDefaultDateRange } from '@/utils/date-range.utils';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import { SheetFooter } from '@/components/shared/SheetFooter';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

export const modalConfig: ModalConfig = {
  id: 'timesheet_entry',
  name: 'Timesheet Entry',
  description: 'Create or edit timesheet entries',
  category: 'timesheet',
  permissions: {
    action: 'create',
    subject: 'Timesheet',
    roles: ['admin', 'partner'],
  },
  deeplink: {
    enabled: true,
    params: ['employeeId', 'projectId'],
  },
  requiresAuth: true,
  encryptData: false,
};

interface TimesheetEntrySheetProps {
  isOpen: boolean;
  onClose: () => void;
  employeeId?: number;
  projectId?: number;
  onSuccess?: () => Promise<void>;
  // Note: URL params are only used for initial values, not for tracking changes
}


interface TimesheetEntrySheetComponentProps extends TimesheetEntrySheetProps {}

function TimesheetEntrySheetComponent({
  isOpen,
  onClose,
  employeeId,
  projectId,
  onSuccess
}: TimesheetEntrySheetComponentProps) {
  const [isProcessingCache, setIsProcessingCache] = useState(false);
  const [isClosing, setIsClosing] = useState(false);
  const [showCloseConfirm, setShowCloseConfirm] = useState(false);
  const [selectedEmployeeFilter, setSelectedEmployeeFilter] = useState<number | null>(null);
  const [dateRange, setDateRange] = useState(() => getDefaultDateRange());
  const [isEmployeeDropdownOpen, setIsEmployeeDropdownOpen] = useState(false);
  const [isProjectDropdownOpen, setIsProjectDropdownOpen] = useState(false);
  const autoPopulatedRef = useRef<string>('');
  const createTimesheetsMutation = useCreateTimesheets();

  // Fetch project data first so off_days is available for entry factory
  const { data: projectsData } = useTimesheetProjects({ enabled: isOpen });
  const projects = projectsData?.data || [];

  // Track selected project id separately so we can derive off_days before the hook
  const [selectedProjectId, setSelectedProjectId] = useState<number>(projectId || 0);
  const selectedProject = projects.find(p => p.id === selectedProjectId) || null;
  const projectOffDays = selectedProject?.off_days ?? 0;

  const {
    formData,
    availableEmployees,
    validation,
    getDayTypesForPosition,
    getHourTypesForEntry,
    hasPayRateForEntry,
    getPayRateForEntry,
    getAvailableProjectsForEmployee,
    transformEntriesToAPI,
    handleProjectChange,
    handleAddEntry,
    handleRemoveEntry,
    handleDuplicateEntry,
    handleEntryChange,
    handleAddAllEmployees,
    handleDateRangeChange,
    resetForm,
    triggerPreview,
    isLoadingPayRate,
    isPayRateReady,
    isPreviewLoading,
    allEntriesValid,
    getValidationErrors,
    getErrorsForEmployee,
    getErrorsForRow,
    fetchAndMergeExistingEntries
  } = useMultiTimesheetFormState({
    employeeId,
    projectId,
    isOpen,
    isClosing,
    projectOffDays
  });

  const handleSave = useCallback(async () => {
    if (!validation.valid || isClosing) {
      return;
    }

    // Check if entries are already validated and valid
    const validationErrors = getValidationErrors();

    if (!allEntriesValid || validationErrors) {
      // Show cached validation errors without additional API calls
      return;
    }

    try {
      // Transform UI entries to API format (split hours object into multiple entries)
      // This wrapper automatically filters out unchanged entries
      const entries: NewTimesheetEntry[] = transformEntriesToAPI(formData.entries);

      // Send to backend and wait for response
      await createTimesheetsMutation.mutateAsync(entries);

      // Suppress auto-populate so it doesn't race with the fetch below
      const populationKey = `${formData.projectId}-${dateRange.startDate}-${dateRange.endDate}`;
      autoPopulatedRef.current = populationKey;

      // Reset form but keep project
      resetForm(formData.projectId);

      // Re-fetch fresh data from server — fetchingRef was cleared by resetForm
      if (formData.projectId && dateRange.startDate && dateRange.endDate) {
        await fetchAndMergeExistingEntries(formData.projectId, dateRange.startDate, dateRange.endDate);
      }

      if (onSuccess) {
        await onSuccess();
      }
    } catch (error) {
      // Error handling is done in the mutation hook
      console.error('Timesheet creation failed:', error);
    }
  }, [validation, formData.entries, formData.projectId, dateRange, transformEntriesToAPI, createTimesheetsMutation, resetForm, onSuccess, allEntriesValid, getValidationErrors, isClosing, fetchAndMergeExistingEntries]);

  const handleCloseClick = useCallback(() => {
    // Check for unsaved changes: any entry that has hours entered or differs from original
    const hasUnsavedChanges = formData.entries.some((entry) => {
      if (!entry.originalValues) {
        // New entry — has changes if any hours > 0
        return entry.hours && Object.values(entry.hours).some((h) => h > 0);
      }
      // Existing entry — has changes if hours differ from original
      const origHours = entry.originalValues.hours;
      const currHours = entry.hours || {};
      const allKeys = new Set([...Object.keys(origHours), ...Object.keys(currHours)]);
      return Array.from(allKeys).some((k) => (origHours[k] ?? 0) !== (currHours[k] ?? 0));
    });

    if (hasUnsavedChanges) {
      setShowCloseConfirm(true);
    } else {
      setIsClosing(true);
      onClose();
    }
  }, [formData.entries, onClose]);

  const handleConfirmClose = useCallback(() => {
    setShowCloseConfirm(false);
    setIsClosing(true);
    onClose();
  }, [onClose]);

  const handleProjectSelect = useCallback((project: { id?: number } | null) => {
    if (isClosing) return;
    const newProjectId = project?.id || 0;
    setSelectedProjectId(newProjectId);
    handleProjectChange(newProjectId);
    // Reset employee filter when project changes
    setSelectedEmployeeFilter(null);
  }, [handleProjectChange, isClosing]);

  const handleEmployeeFilterChange = useCallback((value: string) => {
    if (isClosing) return;
    // "all" means show all employees, otherwise it's an employee ID
    setSelectedEmployeeFilter(value === 'all' ? null : Number(value));
  }, [isClosing]);

  const handleStartDateChange = useCallback((newStartDate: string) => {
    if (isClosing) return;

    setDateRange((current) => {
      const endDate = current.endDate < newStartDate ? newStartDate : current.endDate;
      const newDateRange = {
        ...current,
        startDate: newStartDate,
        endDate
      };

      // Update entries to match the new date range
      handleDateRangeChange(newStartDate, endDate);

      return newDateRange;
    });
  }, [isClosing, handleDateRangeChange]);

  const handleEndDateChange = useCallback((newEndDate: string) => {
    if (isClosing) return;

    setDateRange((current) => {
      const startDate = current.startDate > newEndDate ? newEndDate : current.startDate;
      const newDateRange = {
        ...current,
        startDate,
        endDate: newEndDate
      };

      // Update entries to match the new date range
      handleDateRangeChange(startDate, newEndDate);

      return newDateRange;
    });
  }, [isClosing, handleDateRangeChange]);

  // Fetch existing entries when sheet is open, has project, date range, and employees are loaded
  // This handles both initial load and subsequent changes to date range or project
  useEffect(() => {
    if (isOpen &&
        !isClosing &&
        dateRange.startDate &&
        dateRange.endDate &&
        formData.projectId &&
        availableEmployees.length > 0) {
      // Fetch existing entries from API and merge/fill gaps
      fetchAndMergeExistingEntries(formData.projectId, dateRange.startDate, dateRange.endDate);
    }
  }, [isOpen, isClosing, dateRange.startDate, dateRange.endDate, formData.projectId, availableEmployees.length, fetchAndMergeExistingEntries]);

  const selectedEmployee = useMemo(
    () => availableEmployees.find((employee) => employee.id === selectedEmployeeFilter) || null,
    [availableEmployees, selectedEmployeeFilter]
  );

  const employeeDisplayValue = useMemo(() => {
    if (!selectedEmployeeFilter || !selectedEmployee) {
      return 'Tất cả nhân viên';
    }

    return `${selectedEmployee.fullname} • ${selectedEmployee.cccd}`;
  }, [selectedEmployeeFilter, selectedEmployee]);


  // Auto-populate entries when project is selected and has employees
  useEffect(() => {
    if (!isOpen || !formData.projectId || availableEmployees.length === 0 || !isPayRateReady || isClosing) {
      return;
    }

    // Create a unique key for this project/date range combination
    const populationKey = `${formData.projectId}-${dateRange.startDate}-${dateRange.endDate}`;

    // Skip if we've already auto-populated for this combination
    if (autoPopulatedRef.current === populationKey) {
      return;
    }

    // Ensure employees actually belong to the currently selected project
    const employeesForCurrentProject = availableEmployees.filter((employee) =>
      employee.current_projects?.some((project) => project.project_id === formData.projectId)
    );

    if (employeesForCurrentProject.length === 0) {
      return;
    }

    // Only auto-generate when there are no entries assigned to any employee
    // for the currently selected project yet.
    const hasAssignedEntriesForCurrentProject = formData.entries.some(
      (entry) => entry.employeeId && entry.projectId === formData.projectId
    );

    if (hasAssignedEntriesForCurrentProject) {
      // Mark as populated even if entries exist, to prevent re-runs
      autoPopulatedRef.current = populationKey;
      return;
    }

    // Create one entry per day in date range for each employee of this project (limited to current month)
    const start = new Date(dateRange.startDate);
    const end = new Date(dateRange.endDate);

    const days = Math.ceil((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)) + 1;

    employeesForCurrentProject.forEach((employee) => {
      for (let i = 0; i < days; i++) {
        const currentDate = new Date(start);
        currentDate.setDate(start.getDate() + i);
        const dateStr = currentDate.toISOString().split('T')[0];

        // Use the hook helper to create the entry with the correct date
        handleAddEntry(employee, dateStr);
      }
    });

    // Mark this combination as populated
    autoPopulatedRef.current = populationKey;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen, formData.projectId, availableEmployees, dateRange, handleAddEntry, isPayRateReady, isClosing]);
  // Note: formData.entries intentionally excluded to prevent infinite loops - guards above are sufficient

  // Keep selectedProjectId in sync with formData.projectId (e.g. when set via URL params)
  useEffect(() => {
    if (formData.projectId && formData.projectId !== selectedProjectId) {
      setSelectedProjectId(formData.projectId);
    }
  }, [formData.projectId, selectedProjectId]);

  // Reset form and filters when sheet closes
  useEffect(() => {
    if (!isOpen && !isClosing) {
      resetForm();
      setSelectedEmployeeFilter(null);
      setSelectedProjectId(0);
      const defaultRange = getDefaultDateRange();
      setDateRange(defaultRange);
      autoPopulatedRef.current = '';
      setIsClosing(false);
    }
  }, [isOpen, isClosing, resetForm]);

  const isLoading = createTimesheetsMutation.isPending || isProcessingCache;
  const isInitialDataLoading = isLoadingPayRate && !isPayRateReady;

  const footerSummary = useMemo(() => {
    if (!formData.projectId || !isPayRateReady || formData.entries.length === 0) return null;

    const employeesWithHours = new Set<number>();
    let totalHours = 0;
    let totalPayout = 0;
    let newFieldCount = 0;
    let editedFieldCount = 0;
    let deletedFieldCount = 0;

    formData.entries.forEach((entry) => {
      const hours = entry.hours || {};
      const originalHours = entry.originalValues?.hours || {};
      const entryHours = Object.values(hours).reduce((s, h) => s + h, 0);

      if (entryHours > 0 && entry.employeeId) employeesWithHours.add(entry.employeeId);
      totalHours += entryHours;

      if (entry.position && entry.dayType) {
        Object.entries(hours).forEach(([ht, h]) => {
          if (h > 0) totalPayout += h * getPayRateForEntry(entry.position!, entry.dayType!, ht);
        });
      }

      if (!entry.originalValues) {
        // New entry — count each field with a value
        Object.values(hours).forEach(h => { if (h > 0) newFieldCount++; });
      } else {
        // Existing entry — compare field by field
        const allKeys = new Set([...Object.keys(originalHours), ...Object.keys(hours)]);
        allKeys.forEach(ht => {
          const orig = originalHours[ht] ?? 0;
          const curr = hours[ht] ?? 0;
          if (curr === orig) return;
          if (orig > 0 && curr === 0) deletedFieldCount++;
          else editedFieldCount++;
        });
      }
    });

    return { employees: employeesWithHours.size, totalHours, totalPayout, newFieldCount, editedFieldCount, deletedFieldCount };
  }, [formData.entries, formData.projectId, isPayRateReady, getPayRateForEntry]);

  const saveDisabledReason = useMemo(() => {
    if (!formData.projectId) return 'Chưa chọn dự án';
    if (!isPayRateReady) return 'Chưa có cấu hình lương cho dự án này';
    if (isLoading) return null;
    if (!allEntriesValid) {
      const errorCount = Object.values(validation.entryValidations).filter(
        (v) => !v.valid && v.errors.length > 0
      ).length;
      return errorCount > 0 ? `Còn lỗi ở ${errorCount} dòng` : 'Dữ liệu chưa hợp lệ';
    }
    return null;
  }, [formData.projectId, isPayRateReady, isLoading, allEntriesValid, validation.entryValidations]);

  const canSave = !isLoading && validation.valid && allEntriesValid && !!formData.projectId;

  return (
    <Sheet open={isOpen} onOpenChange={(open) => { if (!open) handleCloseClick(); }}>
      <SheetContent
        side="right"
        title="Thêm chấm công"
        className="p-0 flex flex-col h-full !w-screen max-w-[1200px] mx-auto"
      >
        {/* Header — title, filters, close */}
        <div className="flex min-h-14 flex-wrap items-center gap-2 border-b bg-background px-4 py-2 flex-shrink-0">
          <h1 className="text-sm font-semibold text-foreground shrink-0">Thêm chấm công</h1>
          <div className="w-px h-4 bg-border shrink-0" />
          {/* Project selector */}
          <Popover open={isProjectDropdownOpen} onOpenChange={setIsProjectDropdownOpen} modal={false}>
            <PopoverTrigger asChild>
              <Button
                variant="ghost"
                role="combobox"
                aria-expanded={isProjectDropdownOpen}
                className="min-h-11 max-w-full justify-between gap-1 border border-transparent px-3 text-xs font-normal hover:bg-accent hover:border-input"
              >
                <span className="min-w-0 truncate">
                  {selectedProject
                    ? <><span className="font-medium">{selectedProject.name}</span>{selectedProject.code && <span className="text-muted-foreground ml-1 text-xs">{selectedProject.code}</span>}</>
                    : <span className="text-muted-foreground">Chọn dự án</span>
                  }
                </span>
                <ChevronsUpDown className="h-3 w-3 shrink-0 opacity-40" />
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-72 p-0 z-[100]" align="start">
              <Command
                shouldFilter={true}
                filter={(value, search) => {
                  if (!search) return 1;
                  return vietnameseIncludes(value, search) ? 1 : 0;
                }}
              >
                <CommandInput placeholder="Tên hoặc mã dự án" className="border-0 focus:ring-0" />
                <CommandList className="max-h-[440px]" onWheel={(e) => e.stopPropagation()}>
                  <CommandEmpty>Không tìm thấy dự án nào</CommandEmpty>
                  <CommandGroup>
                    {projects.map((project) => (
                      <CommandItem
                        key={project.id}
                        value={`${project.name} ${project.code || ''}`}
                        onSelect={() => { handleProjectSelect(project); setIsProjectDropdownOpen(false); }}
                      >
                        <Check className={cn('mr-2 h-4 w-4', selectedProject?.id === project.id ? 'opacity-100' : 'opacity-0')} />
                        <span className="font-medium">{project.name}</span>
                        {project.code && <span className="ml-1.5 text-xs text-muted-foreground shrink-0">{project.code}</span>}
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>

          {/* Employee filter */}
          <Popover open={isEmployeeDropdownOpen} onOpenChange={setIsEmployeeDropdownOpen} modal={false}>
            <PopoverTrigger asChild>
              <Button
                variant="ghost"
                role="combobox"
                aria-expanded={isEmployeeDropdownOpen}
                className="min-h-11 max-w-full justify-between gap-1 border border-transparent px-3 text-xs font-normal hover:bg-accent hover:border-input"
                disabled={availableEmployees.length === 0 || !isPayRateReady}
              >
                <span className="min-w-0 truncate">
                  {selectedEmployeeFilter && selectedEmployee
                    ? <span className="font-medium">{selectedEmployee.fullname}</span>
                    : <span className="text-muted-foreground">Tất cả nhân viên</span>
                  }
                </span>
                <ChevronsUpDown className="h-3 w-3 shrink-0 opacity-40" />
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-72 p-0 z-[100]" align="start">
              <Command
                shouldFilter={true}
                filter={(value, search) => {
                  if (!search) return 1;
                  return vietnameseIncludes(value, search) ? 1 : 0;
                }}
              >
                <CommandInput placeholder="Tên hoặc CCCD" className="border-0 focus:ring-0" />
                <CommandList className="max-h-[440px]" onWheel={(e) => e.stopPropagation()}>
                  <CommandEmpty>Không tìm thấy nhân viên nào</CommandEmpty>
                  <CommandGroup>
                    <CommandItem
                      value="Tất cả nhân viên"
                      onSelect={() => { handleEmployeeFilterChange('all'); setIsEmployeeDropdownOpen(false); }}
                    >
                      <Check className={cn('mr-2 h-4 w-4', selectedEmployeeFilter === null ? 'opacity-100' : 'opacity-0')} />
                      <span className="font-medium">Tất cả nhân viên</span>
                    </CommandItem>
                    {availableEmployees.map((employee) => (
                      <CommandItem
                        key={employee.id}
                        value={`${employee.fullname} ${employee.cccd}`}
                        onSelect={() => { handleEmployeeFilterChange(String(employee.id)); setIsEmployeeDropdownOpen(false); }}
                      >
                        <Check className={cn('mr-2 h-4 w-4', selectedEmployeeFilter === employee.id ? 'opacity-100' : 'opacity-0')} />
                        <div className="flex flex-col flex-1">
                          <span className="font-medium">{employee.fullname}</span>
                          <span className="typography-body-small text-muted-foreground">CCCD: {employee.cccd}</span>
                        </div>
                      </CommandItem>
                    ))}
                  </CommandGroup>
                </CommandList>
              </Command>
            </PopoverContent>
          </Popover>

          {/* Date range */}
          <DateRangePicker
            variant="compact"
            startDate={dateRange.startDate}
            endDate={dateRange.endDate}
            onStartDateChange={handleStartDateChange}
            onEndDateChange={handleEndDateChange}
            minDate={useMemo(() => new Date(Date.now() - 14 * 24 * 60 * 60 * 1000), [])}
            maxDate={useMemo(() => new Date(), [])}
          />

          <div className="min-w-6 flex-1" />
          <Button
            variant="ghost"
            size="icon"
            onClick={handleCloseClick}
            className="h-11 w-11 rounded-xl text-muted-foreground hover:text-foreground hover:bg-muted shrink-0"
            aria-label="Đóng"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Global error banner — only for errors not tied to a specific employee */}
        {getErrorsForEmployee(0) && (
          <div className="px-4 py-2 bg-red-50 border-b border-red-100 flex items-start gap-2 flex-shrink-0">
            <AlertTriangle className="h-3.5 w-3.5 text-red-500 shrink-0 mt-0.5" />
            <div className="space-y-0.5">
              {getErrorsForEmployee(0)!.map((err, i) => (
                <p key={i} className="text-xs text-red-600">{err}</p>
              ))}
            </div>
          </div>
        )}

        {/* Scrollable body */}
        <div className="flex-1 overflow-y-auto bg-muted/40">
        {/* Content */}
        <div className="px-6 pt-4 pb-6">
          {!isPayRateReady ? (
            <div className="border rounded-xl p-4 bg-muted/40 text-sm text-muted-foreground">
              {isInitialDataLoading
                ? 'Đang tải cấu hình mức lương cho dự án này...'
                : 'Dự án này chưa có cấu hình mức lương. Vui lòng cấu hình mức lương trước khi tạo chấm công.'}
            </div>
          ) : (
            <MultiEntryTable
              entries={formData.entries}
              availableEmployees={availableEmployees}
              validation={validation}
              getDayTypesForPosition={getDayTypesForPosition}
              getHourTypesForEntry={getHourTypesForEntry}
              hasPayRateForEntry={hasPayRateForEntry}
              getPayRateForEntry={getPayRateForEntry}
              onEntryChange={handleEntryChange}
              onAddEntry={handleAddEntry}
              onRemoveEntry={handleRemoveEntry}
              onPreviewRequest={triggerPreview}
              isLoading={createTimesheetsMutation.isPending}
              filteredEmployeeId={selectedEmployeeFilter}
              getErrorsForEmployee={getErrorsForEmployee}
              getErrorsForRow={getErrorsForRow}
            />
          )}
        </div>
        </div>

        {/* Footer */}
        <SheetFooter
          legend={
            footerSummary ? (
              <div className="flex items-center gap-4 text-xs text-muted-foreground">
                <span>
                  <span className="font-semibold text-foreground">{footerSummary.employees}</span> nhân viên
                </span>
                <span>
                  <span className="font-semibold text-foreground">{footerSummary.totalHours}h</span> tổng giờ
                </span>
                <span className="font-semibold text-emerald-600">{formatCurrency(footerSummary.totalPayout)}</span>
                {(footerSummary.newFieldCount > 0 || footerSummary.editedFieldCount > 0 || footerSummary.deletedFieldCount > 0) && (
                  <span className="flex items-center gap-2">
                    {footerSummary.newFieldCount > 0 && (
                      <span className="flex items-center gap-0.5">
                        <span className="inline-block w-2 h-2 rounded-sm border border-emerald-300 bg-emerald-100" />
                        <span className="text-emerald-600 font-medium">{footerSummary.newFieldCount} mới</span>
                      </span>
                    )}
                    {footerSummary.editedFieldCount > 0 && (
                      <span className="flex items-center gap-0.5">
                        <span className="inline-block w-2 h-2 rounded-sm border border-amber-300 bg-amber-100" />
                        <span className="text-amber-600 font-medium">{footerSummary.editedFieldCount} sửa</span>
                      </span>
                    )}
                    {footerSummary.deletedFieldCount > 0 && (
                      <span className="flex items-center gap-0.5">
                        <span className="inline-block w-2 h-2 rounded-sm border border-red-300 bg-red-100" />
                        <span className="text-red-500 font-medium">{footerSummary.deletedFieldCount} xóa</span>
                      </span>
                    )}
                  </span>
                )}
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <span className="inline-block w-4 h-3.5 rounded border border-emerald-300 bg-emerald-50/70 shrink-0" />
                  Mới tạo
                </span>
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <span className="inline-block w-4 h-3.5 rounded border border-amber-300 bg-amber-50/70 shrink-0" />
                  Đã sửa
                </span>
                <span className="flex items-center gap-1 text-[10px] text-muted-foreground">
                  <span className="inline-block w-4 h-3.5 rounded border border-red-300 bg-red-50/70 shrink-0" />
                  Sẽ xóa
                </span>
              </div>
            )
          }
          actions={
            <>
              <Button
                variant="outline"
                onClick={handleCloseClick}
                className="min-h-11 px-4"
              >
                Hủy
              </Button>
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    {/* span needed so tooltip works on disabled button */}
                    <span tabIndex={canSave ? -1 : 0}>
                      <Button
                        onClick={handleSave}
                        disabled={!canSave}
                        className="min-h-11 px-5"
                      >
                        {isLoading ? 'Đang lưu...' : 'Lưu chấm công'}
                      </Button>
                    </span>
                  </TooltipTrigger>
                  {saveDisabledReason && (
                    <TooltipContent side="top">
                      {saveDisabledReason}
                    </TooltipContent>
                  )}
                </Tooltip>
              </TooltipProvider>
            </>
          }
        />

        <ConfirmDialog
          open={showCloseConfirm}
          onOpenChange={setShowCloseConfirm}
          title="Thoát mà không lưu?"
          description="Bạn có thay đổi chưa được lưu. Nếu thoát bây giờ, các thay đổi sẽ bị mất."
          confirmText="Thoát"
          cancelText="Ở lại"
          onConfirm={handleConfirmClose}
          confirmVariant="destructive"
        />
      </SheetContent>
    </Sheet>
  );
}

/**
 * Container component that provides URL synchronization for TimesheetEntrySheet
 * Updated to work with modal navigation system
 */
export function TimesheetEntrySheet({
  isOpen: propIsOpen,
  onClose: propOnClose,
  ...props
}: TimesheetEntrySheetProps) {
  const [searchParams] = useSearchParams();
  const { closeModal } = useModalNavigation();

  // Check if modal is open via URL
  const modalId = searchParams.get('modal');
  const isOpenViaUrl = modalId === 'timesheet_entry';

  // Always use URL as source of truth for parameters
  const urlProjectId = searchParams.get('projectId');
  const urlEmployeeId = searchParams.get('employeeId');
  const projectId = urlProjectId ? parseInt(urlProjectId, 10) : props.projectId;
  const employeeId = urlEmployeeId ? parseInt(urlEmployeeId, 10) : props.employeeId;

  // Use URL state if available, otherwise use props
  const isOpen = isOpenViaUrl || propIsOpen;

  // Close modal using centralized navigation or provided onClose
  const handleClose = useCallback(() => {
    if (isOpenViaUrl) {
      closeModal();
    } else {
      propOnClose();
    }
  }, [isOpenViaUrl, closeModal, propOnClose]);

  // Note: URL parameters are only used for initial values, not for tracking changes
  // This prevents unnecessary re-renders while still supporting deep-linking

  return (
    <TimesheetEntrySheetComponent
      isOpen={isOpen}
      onClose={handleClose}
      projectId={projectId}
      employeeId={employeeId}
      {...props}
    />
  );
}

export default TimesheetEntrySheet;
