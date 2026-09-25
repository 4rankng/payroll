import { useMemo } from 'react';
import { format, isSameMonth, isToday, getDay, eachDayOfInterval, startOfMonth, endOfMonth } from 'date-fns';
import { vi } from 'date-fns/locale';
import { Plus, Clock, Banknote, Star } from 'lucide-react';
import { cn } from '@/lib/utils';
import { Timesheet } from '@/types/api/timesheet.types';
import { getAggregatedStatus, type CalendarTimesheetEntry } from '../utils/timesheetCalendarHelpers';
import { formatCurrency } from '@/utils/formatters';
import type { ProjectCalendarData } from '../utils/timesheetCalendarHelpers';

interface MobileDayListViewProps {
  project: ProjectCalendarData;
  year: number;
  month: number;
  bulkTransferPercentage?: number;
  onDayClick: (date: Date, entries: CalendarTimesheetEntry[]) => void;
}

const WEEKDAY_SHORT = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];

const STATUS_DOT: Record<string, string> = {
  paid: 'bg-amber-500',
  approved: 'bg-green-500',
  pending_approval: 'bg-gray-400',
  rejected: 'bg-red-500',
  mixed: 'bg-teal-500',
  none: 'bg-transparent',
};

const STATUS_ROW_BG: Record<string, string> = {
  paid: 'bg-amber-50 border-amber-200',
  approved: 'bg-green-50 border-green-200',
  pending_approval: 'bg-muted/50 border-border',
  rejected: 'bg-red-50 border-red-200',
  mixed: 'bg-teal-50 border-teal-200',
  none: 'bg-card border-border'
};

const STATUS_LABEL: Record<string, string> = {
  paid: 'Đã TT',
  approved: 'Đã duyệt',
  pending_approval: 'Chờ duyệt',
  rejected: 'Bị loại',
  mixed: 'Hỗn hợp',
};

export function MobileDayListView({
  project,
  year,
  month,
  bulkTransferPercentage = 0,
  onDayClick,
}: MobileDayListViewProps) {
  // Build flat list of all days in the month from the calendar grid
  const days = useMemo(() => {
    const monthStart = startOfMonth(new Date(year, month - 1));
    const monthEnd = endOfMonth(monthStart);
    return eachDayOfInterval({ start: monthStart, end: monthEnd });
  }, [year, month]);

  // Build a lookup from date string → entries
  const entriesByDate = useMemo(() => {
    const map: Record<string, CalendarTimesheetEntry[]> = {};
    project.calendarDays.flat().forEach((day) => {
      if (day.isCurrentMonth && day.entries.length > 0) {
        const key = format(day.date, 'yyyy-MM-dd');
        map[key] = day.entries;
      }
    });
    return map;
  }, [project.calendarDays]);

  // Group days by week for visual separation
  const weeks = useMemo(() => {
    const result: Date[][] = [];
    let week: Date[] = [];
    days.forEach((day) => {
      week.push(day);
      if (week.length === 7 || day === days[days.length - 1]) {
        result.push(week);
        week = [];
      }
    });
    return result;
  }, [days]);

  return (
    <div className="space-y-1">
      {/* Project header */}
      <div className="flex items-center justify-between px-1 pb-1">
        <div>
          <p className="text-sm font-semibold text-foreground">{project.projectName}</p>
          {project.projectCode && (
            <p className="text-xs text-muted-foreground">{project.projectCode}</p>
          )}
        </div>
        <div className="flex items-center gap-3 text-xs text-muted-foreground">
          <span className="font-semibold text-foreground">{project.monthlyStats.totalHours}h</span>
          <span className="font-semibold text-foreground">{formatCurrency(project.monthlyStats.totalAmount)}</span>
        </div>
      </div>

      {/* Day rows */}
      <div className="space-y-0.5">
        {days.map((day) => {
          const dateKey = format(day, 'yyyy-MM-dd');
          const entries = entriesByDate[dateKey] ?? [];
          const hasEntries = entries.length > 0;
          const weekday = getDay(day); // 0=Sun
          const isSun = weekday === 0;
          const isSat = weekday === 6;
          const today = isToday(day);
          const totalHours = entries.reduce((s, e) => s + e.hours_worked, 0);
          const totalAmount = entries.reduce((s, e) => s + e.amount, 0);
          const totalPaid = entries.reduce((s, e) => s + (e.paid_amount ?? 0), 0);
          const paymentLimit = totalAmount * bulkTransferPercentage;
          const status = getAggregatedStatus(entries);
          const hasForcePayroll = entries.some((e) => e.force_payroll);

          return (
            <div
              key={dateKey}
              onClick={() => onDayClick(day, entries)}
              className={cn(
                'flex items-center gap-3 px-3 py-2.5 rounded-xl border cursor-pointer touch-manipulation transition-all card-lift',
                hasEntries ? STATUS_ROW_BG[status] ?? 'bg-card border-border' : 'bg-card border-border hover:bg-muted/50',
                today && 'ring-2 ring-primary ring-offset-0',
              )}
              role="button"
              tabIndex={0}
              onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') onDayClick(day, entries); }}
              aria-label={`${format(day, 'dd/MM/yyyy')} - ${hasEntries ? `${totalHours}h` : 'Chưa có công'}`}
            >
              {/* Date column */}
              <div className="w-10 shrink-0 text-center">
                <p className={cn(
                  'text-xs font-medium leading-none mb-0.5',
                  isSun ? 'text-red-600' : isSat ? 'text-blue-600' : 'text-muted-foreground',
                )}>
                  {WEEKDAY_SHORT[weekday]}
                </p>
                <p className={cn(
                  'text-base font-bold leading-none',
                  today ? 'text-primary' : isSun ? 'text-red-600' : 'text-foreground',
                )}>
                  {format(day, 'd')}
                </p>
              </div>

              {/* Status dot */}
              <div className={cn(
                'w-2 h-2 rounded-full shrink-0',
                hasEntries ? STATUS_DOT[status] ?? 'bg-gray-400' : 'bg-gray-200',
              )} />

              {/* Content */}
              {hasEntries ? (
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-3 flex-wrap">
                    <div className="flex items-center gap-1">
                      <Clock className="h-3 w-3 text-muted-foreground shrink-0" />
                      <span className="text-xs font-semibold text-foreground">{totalHours}h</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <Banknote className="h-3 w-3 text-muted-foreground shrink-0" />
                      <span className="text-xs font-semibold text-foreground tabular-nums">{formatCurrency(totalAmount)}</span>
                    </div>
                    {bulkTransferPercentage > 0 && (
                      <span className="text-xs text-muted-foreground tabular-nums">
                        Mức: {formatCurrency(paymentLimit)}
                      </span>
                    )}
                    {totalPaid > 0 && (
                      <span className="text-xs text-amber-700 tabular-nums">
                        Đã TT: {formatCurrency(totalPaid)}
                      </span>
                    )}
                  </div>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className={cn(
                      'text-xs font-medium',
                      status === 'paid' ? 'text-amber-700' :
                      status === 'approved' ? 'text-green-700' :
                      status === 'rejected' ? 'text-red-700' :
                      'text-muted-foreground'
                    )}>
                      {STATUS_LABEL[status] ?? status}
                    </span>
                    {entries.length > 1 && (
                      <span className="text-xs text-muted-foreground">{entries.length} bản ghi</span>
                    )}
                    {hasForcePayroll && (
                      <Star className="h-3 w-3 text-yellow-700 fill-yellow-500 shrink-0" />
                    )}
                  </div>
                </div>
              ) : (
                <div className="flex-1 flex items-center gap-1.5">
                  <Plus className="h-3.5 w-3.5 text-muted-foreground/50" />
                  <span className="text-xs text-muted-foreground/60">Thêm công</span>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
