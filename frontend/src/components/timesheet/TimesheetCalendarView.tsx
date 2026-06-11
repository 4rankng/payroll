import { useState, useMemo, useCallback } from 'react';
import { useEmployeeTimesheet } from '@/hooks/api/useEmployees';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Calendar, Clock, HandCoins } from 'lucide-react';
import { showErrorNotification } from '@/utils/error-handler';
import { Timesheet } from '@/types/api/timesheet.types';
import type { EmployeeTimesheetFilters } from '@/types/api/employee.types';
import {
  createEmployeeCalendarData,
  ProjectCalendarData,
  CalendarDay,
  type CalendarTimesheetEntry
} from './utils/timesheetCalendarHelpers';
import { formatCurrency } from '@/utils/formatters';
import { CalendarDayCell } from './components/CalendarDayCell';
import { MobileDayListView } from './components/MobileDayListView';
import { TimesheetEntryModal } from './components/TimesheetEntryModal';
import { TimesheetDayEntriesModal } from './components/TimesheetDayEntriesModal';
import { useMediaQuery } from '@/hooks/useBreakpoint';

interface Project {
  id: number;
  name: string;
}

interface Employee {
  id: number;
  name: string;
  subtitle?: string;
}

interface TimesheetCalendarViewProps {
  timesheets: Timesheet[];
  isLoading?: boolean;
  error?: unknown;
  // Filter props
  selectedMonth: string;
  onMonthChange: (value: string) => void;
  selectedProject: string;
  onProjectChange: (value: string) => void;
  selectedEmployee: string;
  onEmployeeChange: (value: string) => void;
  statusFilter: Timesheet['status'] | Timesheet['payment_status'] | 'all';
  onStatusChange: (value: Timesheet['status'] | Timesheet['payment_status'] | 'all') => void;
  projects: Project[];
  employees: Employee[];
  projectPendingCount?: number;
  selectedProjectName?: string;
  onProjectBulkApprove?: () => void;
  // Action handlers
  onDelete?: (timesheet: Timesheet) => void;
  canDelete?: (timesheet: Timesheet) => boolean;
  bulkTransferPercentage?: number;
  onRequestEdit?: (timesheet: Timesheet, onSuccess?: () => Promise<void> | void) => Promise<void> | void;
  requestingTimesheetId?: number | null;
}

const WEEKDAYS = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];

export function TimesheetCalendarView({
  timesheets,
  isLoading = false,
  error,
  // Filter props
  selectedMonth,
  onMonthChange,
  selectedProject,
  onProjectChange,
  selectedEmployee,
  onEmployeeChange,
  statusFilter,
  onStatusChange,
  projects,
  employees,
  projectPendingCount,
  selectedProjectName,
  onProjectBulkApprove,
  // Action handlers
  onDelete,
  canDelete,
  bulkTransferPercentage = 0,
  onRequestEdit,
  requestingTimesheetId = null
}: TimesheetCalendarViewProps) {
  const [isEntryModalOpen, setIsEntryModalOpen] = useState(false);
  const [entryModalData, setEntryModalData] = useState<{
    projectId: number;
    projectName: string;
    employeeId: number;
    employeeName: string;
    date: Date;
    existingEntry?: Timesheet | null;
  } | null>(null);

  const [isDayEntriesModalOpen, setIsDayEntriesModalOpen] = useState(false);
  const [dayEntriesModalData, setDayEntriesModalData] = useState<{
    date: Date;
    entries: Timesheet[];
  } | null>(null);

  // Parse the selectedMonth to get year and month, with fallback to current month
  const today = new Date();
  const currentYear = selectedMonth ? parseInt(selectedMonth.split('-')[0]) : today.getFullYear();
  const currentMonth = selectedMonth ? parseInt(selectedMonth.split('-')[1]) : today.getMonth() + 1;
  const currentDate = new Date(currentYear, currentMonth - 1, 1);

  // Get selected employee info
  const selectedEmployeeInfo = employees?.find(emp => emp.id.toString() === selectedEmployee);
  const selectedEmployeeId = selectedEmployeeInfo?.id;

  // Calculate timesheet filters
  const timesheetFilters = useMemo(() => {
    if (!selectedMonth || selectedMonth === 'all') return undefined;

    const lastDay = new Date(currentYear, currentMonth, 0).getDate();
    const filters: EmployeeTimesheetFilters = {
      fromDate: `${selectedMonth}-01`,
      toDate: `${selectedMonth}-${lastDay.toString().padStart(2, '0')}`,
      pageSize: 1000 // Get all timesheets for the month
    };

    // Add project filter if a specific project is selected
    if (selectedProject && selectedProject !== 'all') {
      const projectId = parseInt(selectedProject);
      if (!isNaN(projectId)) {
        filters.project_ids = projectId.toString();
      }
    }

    // Add status filter if specified
    if (statusFilter && statusFilter !== 'all') {
      filters.status = statusFilter;
    }

    return filters;
  }, [selectedMonth, currentYear, currentMonth, selectedProject, statusFilter]);

  // Fetch timesheets using the /timesheets endpoint (works for all cases)
  const {
    data: timesheetData,
    isLoading: isTimesheetLoading,
    error: timesheetError
  } = useEmployeeTimesheet(
    selectedEmployeeId!,
    timesheetFilters,
    !!selectedEmployeeId
  );

  // Create calendar data from timesheets
  const calendarData = useMemo(() => {
    if (!selectedEmployeeInfo) {
      return null;
    }

    if (timesheetData?.data) {
      // For specific project selection, filter projects list
      const availableProjects = selectedProject !== 'all'
        ? projects.filter(p => p.id.toString() === selectedProject)
        : projects;

      const mappedTimesheets: CalendarTimesheetEntry[] = timesheetData.data.map((entry) => ({
        ...entry,
        projectCode: undefined, // Employee API does not return projectCode
      }));
      return createEmployeeCalendarData(
        mappedTimesheets,
        currentYear,
        currentMonth,
        selectedEmployeeInfo,
        availableProjects
      );
    }

    // No data available yet
    return null;
  }, [
    selectedEmployeeInfo,
    timesheetData,
    currentYear,
    currentMonth,
    selectedProject,
    projects
  ]);


  const handleDayClick = (day: CalendarDay, projectData: ProjectCalendarData) => {
    if (day.entries.length === 0 && day.isCurrentMonth) {
      // Empty day - open entry modal for adding new entry
      if (selectedEmployeeInfo) {
        setEntryModalData({
          projectId: projectData.projectId,
          projectName: projectData.projectName,
          employeeId: selectedEmployeeInfo.id,
          employeeName: selectedEmployeeInfo.name,
          date: day.date,
          existingEntry: null
        });
        setIsEntryModalOpen(true);
      }
    } else if (day.entries.length === 1) {
      // Single entry - open entry modal for editing (if not approved) or viewing
      const entry = day.entries[0];
      const fullEntry = timesheets.find(t => t.id === entry.id);
      setEntryModalData({
        projectId: entry.project_id,
        projectName: entry.projectName,
        employeeId: entry.employee_id,
        employeeName: entry.employeeName,
        date: new Date(entry.date),
        existingEntry: fullEntry || null
      });
      setIsEntryModalOpen(true);
    } else if (day.entries.length > 1) {
      // Multiple entries - open list modal to show all entries
      const fullEntries = day.entries
        .map(entry => timesheets.find(t => t.id === entry.id))
        .filter((t): t is Timesheet => !!t);
      setDayEntriesModalData({
        date: day.date,
        entries: fullEntries
      });
      setIsDayEntriesModalOpen(true);
    }
  };

  const handleEntryClick = (entry: Timesheet, event?: React.MouseEvent) => {
    // Always open entry modal for editing/viewing
    setEntryModalData({
      projectId: entry.project_id,
      projectName: entry.projectName,
      employeeId: entry.employee_id,
      employeeName: entry.employeeName,
      date: new Date(entry.date),
      existingEntry: entry
    });
    setIsEntryModalOpen(true);
  };


  const handleEntryModalClose = () => {
    setIsEntryModalOpen(false);
    setEntryModalData(null);
  };

  const handleEntryModalSuccess = async () => {
    // Wait briefly for React Query cache updates to propagate before closing modal
    // This ensures the calendar view updates with the new approval status
    await new Promise(resolve => setTimeout(resolve, 100));
    handleEntryModalClose();
  };

  const handleRequestEditWithClose = useCallback((timesheet: Timesheet) => {
    if (!onRequestEdit) {
      return;
    }
    // Call onRequestEdit with success callback to close modal
    void Promise.resolve(onRequestEdit(timesheet, handleEntryModalClose)).catch((error) => { showErrorNotification(error); });
  }, [onRequestEdit]);

  const handleDayEntriesModalClose = () => {
    setIsDayEntriesModalOpen(false);
    setDayEntriesModalData(null);
  };

  const handleEntrySelectFromList = (entry: Timesheet) => {
    // Open the single entry modal for the selected entry
    setEntryModalData({
      projectId: entry.project_id,
      projectName: entry.projectName,
      employeeId: entry.employee_id,
      employeeName: entry.employeeName,
      date: new Date(entry.date),
      existingEntry: entry
    });
    setIsEntryModalOpen(true);
  };

  if (isLoading || isTimesheetLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-12 w-full" />
        <div className="grid gap-6">
          {Array.from({ length: 2 }).map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-6 w-48" />
              </CardHeader>
              <CardContent>
                <Skeleton className="h-96 w-full" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (error || timesheetError) {
    return (
      <div className="text-center py-12">
        <Calendar className="mx-auto h-12 w-12 text-destructive/50" />
        <h3 className="mt-4 typography-title-large text-destructive">Lỗi tải dữ liệu</h3>
        <p className="mt-2 typography-body-medium text-muted-foreground">
          Không thể tải dữ liệu bảng công. Vui lòng thử lại sau.
        </p>
      </div>
    );
  }

  if (!calendarData) {
    return (
      <div className="text-center py-12">
        <Calendar className="mx-auto h-12 w-12 text-muted-foreground/50" />
        <h3 className="mt-4 typography-title-large">Chưa chọn nhân viên</h3>
        <p className="mt-2 typography-body-medium text-muted-foreground">
          Vui lòng chọn nhân viên để xem bảng công.
        </p>
      </div>
    );
  }

  return (
    <>
      <div className="space-y-6">
        {/* Header */}
        <div className="space-y-2">
          <div>
            <h2 className="text-base font-semibold">
              {calendarData.employeeName}
            </h2>
            <p className="text-xs text-muted-foreground">
              {calendarData.employeeCode} · {calendarData.projects.length} dự án
            </p>
          </div>

          {/* Status Legend — compact horizontal */}
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <div className="flex items-center gap-1"><div className="w-3 h-3 bg-amber-100 border border-amber-300 rounded" /><span>Đã TT</span></div>
            <div className="flex items-center gap-1"><div className="w-3 h-3 bg-green-100 border border-green-200 rounded" /><span>Đã duyệt</span></div>
            <div className="flex items-center gap-1"><div className="w-3 h-3 bg-muted border border-border rounded" /><span>Chờ duyệt</span></div>
            <div className="flex items-center gap-1"><div className="w-3 h-3 bg-red-100 border border-red-200 rounded" /><span>Bị loại</span></div>
          </div>
        </div>

        {/* Project Calendars */}
        <div className="space-y-6">
          {calendarData.projects.map((project) => (
            <ProjectCalendar
              key={project.projectId}
              project={project}
              year={currentYear}
              month={currentMonth}
              onDayClick={(day) => handleDayClick(day, project)}
              onEntryClick={handleEntryClick}
              bulkTransferPercentage={bulkTransferPercentage}
            />
          ))}
        </div>
      </div>


      {/* Day Entries List Modal - shown when multiple entries exist for a day */}
      {dayEntriesModalData && (
        <TimesheetDayEntriesModal
          isOpen={isDayEntriesModalOpen}
          onClose={handleDayEntriesModalClose}
          date={dayEntriesModalData.date}
          entries={dayEntriesModalData.entries}
          onEntrySelect={handleEntrySelectFromList}
        />
      )}

      {/* Entry Modal - shown for single entry view/edit */}
      {entryModalData && (
        <TimesheetEntryModal
          isOpen={isEntryModalOpen}
          onClose={handleEntryModalClose}
          projectId={entryModalData.projectId}
          projectName={entryModalData.projectName}
          employeeId={entryModalData.employeeId}
          employeeName={entryModalData.employeeName}
          date={entryModalData.date}
          existingEntry={entryModalData.existingEntry}
          onSuccess={handleEntryModalSuccess}
          onDelete={onDelete}
          canDelete={canDelete}
          onRequestEdit={handleRequestEditWithClose}
          onRequestEditSuccess={handleEntryModalClose}
          requestingTimesheetId={requestingTimesheetId}
        />
      )}
    </>
  );
}

// Project Calendar Component
interface ProjectCalendarProps {
  project: ProjectCalendarData;
  year: number;
  month: number;
  onDayClick: (day: CalendarDay) => void;
  onEntryClick: (entry: Timesheet) => void;
  bulkTransferPercentage?: number;
}

function ProjectCalendar({ project, year, month, onDayClick, onEntryClick, bulkTransferPercentage = 0 }: ProjectCalendarProps) {
  const isMobile = useMediaQuery('(max-width: 767px)');

  // Mobile: handle day click from MobileDayListView
  const handleMobileDayClick = (date: Date, entries: CalendarTimesheetEntry[]) => {
    // Find the matching CalendarDay from the grid
    const dateKey = date.toISOString().slice(0, 10);
    const flatDays = project.calendarDays.flat();
    const calDay = flatDays.find(d => d.date.toISOString().slice(0, 10) === dateKey);
    if (calDay) {
      onDayClick(calDay);
    } else {
      // Day not in grid (shouldn't happen for current month days) — synthesize one
      onDayClick({
        date,
        dayNumber: date.getDate(),
        isCurrentMonth: true,
        isToday: false,
        entries,
        totalHours: entries.reduce((s, e) => s + e.hours_worked, 0),
        totalAmount: entries.reduce((s, e) => s + e.amount, 0),
        totalPaidAmount: entries.reduce((s, e) => s + (e.paid_amount ?? 0), 0),
        aggregatedStatus: 'none',
      });
    }
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-3">
          <div>
            <CardTitle className="text-base">{project.projectName}</CardTitle>
            {project.projectCode && (
              <p className="text-xs text-muted-foreground mt-0.5">{project.projectCode}</p>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            <Badge variant="secondary" className="text-xs">
              <Clock className="w-3 h-3 mr-1" />
              {project.monthlyStats.totalHours}h
            </Badge>
            <Badge variant="secondary" className="text-xs">
              <HandCoins className="w-3 h-3 mr-1" />
              {formatCurrency(project.monthlyStats.totalAmount)}
            </Badge>
            <Badge variant="secondary" className="text-xs">
              <Calendar className="w-3 h-3 mr-1" />
              {project.monthlyStats.workingDays} ngày
            </Badge>
          </div>
        </div>
      </CardHeader>

      <CardContent className="pt-0">
        {isMobile ? (
          // Mobile: vertical day list — much more usable than tiny 7-col grid
          <MobileDayListView
            project={project}
            year={year}
            month={month}
            bulkTransferPercentage={bulkTransferPercentage}
            onDayClick={handleMobileDayClick}
          />
        ) : (
          // Desktop: traditional calendar grid
          <CalendarGrid
            calendarDays={project.calendarDays}
            onDayClick={onDayClick}
            onEntryClick={onEntryClick}
            bulkTransferPercentage={bulkTransferPercentage}
          />
        )}
      </CardContent>
    </Card>
  );
}

// Desktop Calendar Grid
interface CalendarGridProps {
  calendarDays: CalendarDay[][];
  onDayClick: (day: CalendarDay) => void;
  onEntryClick: (entry: Timesheet) => void;
  bulkTransferPercentage?: number;
}

function CalendarGrid({ calendarDays, onDayClick, onEntryClick, bulkTransferPercentage = 0 }: CalendarGridProps) {
  return (
    <div className="space-y-2">
      {/* Weekday Headers */}
      <div className="grid grid-cols-7 gap-1">
        {WEEKDAYS.map((day, index) => (
          <div
            key={day}
            className={`p-2 text-center typography-table-header ${
              index === 0 ? 'text-red-700 font-semibold' : 'text-muted-foreground'
            }`}
          >
            {day}
          </div>
        ))}
      </div>

      {/* Calendar Weeks */}
      <div className="space-y-1">
        {calendarDays.map((week, weekIndex) => (
          <div key={weekIndex} className="grid grid-cols-7 gap-1">
            {week.map((day, dayIndex) => (
              <CalendarDayCell
                key={`${weekIndex}-${dayIndex}`}
                day={day}
                onDayClick={onDayClick}
                onEntryClick={onEntryClick}
                weekIndex={weekIndex}
                totalWeeks={calendarDays.length}
                className={dayIndex === 0 ? 'sunday-gradient-cell' : undefined}
                bulkTransferPercentage={bulkTransferPercentage}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
