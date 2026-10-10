import { cn } from '@/lib/utils';
import { CalendarDay } from '../utils/timesheetCalendarHelpers';
import { formatCurrency } from '@/utils/formatters';
import { Timesheet } from '@/types/api/timesheet.types';
import { Star } from 'lucide-react';
import { getCalendarCellClasses } from '../utils/timesheetStatusColors';

interface CalendarDayCellProps {
  day: CalendarDay;
  onDayClick?: (day: CalendarDay) => void;
  onEntryClick?: (entry: Timesheet, event?: React.MouseEvent) => void;
  className?: string;
  weekIndex?: number;
  totalWeeks?: number;
  bulkTransferPercentage?: number;
}

export function CalendarDayCell({
  day,
  onDayClick,
  onEntryClick,
  className,
  weekIndex = 0,
  totalWeeks = 1,
  bulkTransferPercentage = 0
}: CalendarDayCellProps) {

  const hasEntries = day.entries.length > 0;
  const isCurrentMonth = day.isCurrentMonth;
  const isToday = day.isToday;
  const isSundayColumn = className?.includes('sunday-gradient-cell');

  // Calculate payment limit for employee view using aggregated total
  const paymentLimit = hasEntries ? day.totalAmount * bulkTransferPercentage : 0;

  // Calculate gradient intensity based on week position (0 = lightest, 1 = darkest)

  // Generate slate background color based on gradient intensity
  const getSundayBackgroundClass = () => {
    if (!isSundayColumn) return '';

    // Sunday columns carry white text and light indicator icons, so every
    // rung of the week gradient stays dark enough for 4.5:1 (white/90) —
    // flattening the ramp to slate-600 was cheaper than per-rung text colors.
    // (bg-muted/500 was never a valid Tailwind class and rendered transparent.)
    return 'bg-slate-600 border-border hover:bg-slate-500';
  };

  const handleDayClick = () => {
    if (onDayClick) {
      onDayClick(day);
    }
  };

  // Get status-based styling using aggregated status
  const getStatusStyling = () => {
    if (!hasEntries) return '';
    return getCalendarCellClasses(day.aggregatedStatus);
  };

  return (
    <div
      className={cn(
        "min-h-[120px] md:min-h-[140px] border relative flex flex-col group",
        "transition-colors cursor-pointer",
        !isCurrentMonth && !isSundayColumn ? "hover:bg-slate-700" : (!isCurrentMonth && isSundayColumn ? "" : "hover:bg-muted/30"),
        {
          "bg-slate-400 border-border" : !isCurrentMonth,
          "opacity-80": !isCurrentMonth,
          "ring-2 ring-primary ring-inset border-primary": isToday,
          "hover:ring-2 hover:ring-primary/40 hover:ring-inset border-border": !hasEntries && isCurrentMonth && !isToday,
          "border-border": !hasEntries && isCurrentMonth && !isToday,
        },
        // Apply Sunday gradient only when there are no entries
        isSundayColumn && !hasEntries ? getSundayBackgroundClass() : '',
        // Status styling takes precedence when there are entries
        hasEntries && isCurrentMonth ? getStatusStyling() : '',
        className
      )}
      onClick={handleDayClick}
    >
      {/* Day Number */}
      <div className="flex justify-between items-start p-2">
        <span
          className={cn(
            "typography-label-large font-medium",
            !isCurrentMonth && !isSundayColumn && "text-slate-900",
            !isCurrentMonth && isSundayColumn && "text-white/90",
            isCurrentMonth && !hasEntries && isToday && !isSundayColumn && "text-primary font-semibold",
            isCurrentMonth && !hasEntries && isToday && isSundayColumn && "text-white font-semibold",
            isCurrentMonth && !hasEntries && !isToday && !isSundayColumn && "text-foreground",
            isCurrentMonth && !hasEntries && !isToday && isSundayColumn && "text-white",
            hasEntries && isToday && "text-foreground font-semibold",
            hasEntries && !isToday && isCurrentMonth && "text-foreground"
          )}
        >
          {day.dayNumber}
        </span>
        {/* Force Payroll Icon */}
        {day.hasForcePayroll && (
          <span className="inline-flex items-center" aria-label="Đánh dấu xuất hiện trong kỳ trả lương tiếp theo">
            <Star className={cn('h-4 w-4', isSundayColumn && !hasEntries ? 'text-yellow-800 fill-yellow-800' : 'text-yellow-800 fill-yellow-800')} />
          </span>
        )}
      </div>

      {/* Entry Content */}
      {hasEntries && isCurrentMonth && (
        <div className="flex-1 px-2 pb-2 space-y-0.5">
          {/* Entry Stats - Using aggregated totals */}
          <div className="space-y-0.5 typography-body-small">
            {/* Ca làm việc (Hours worked) */}
            <div className="text-muted-foreground">
              Ca làm việc: <span className="text-foreground font-medium">
                {day.totalHours} giờ
              </span>
            </div>

            {/* Lương (Salary) */}
            <div className="text-muted-foreground">
              Lương: <span className="text-foreground font-medium tabular-nums">
                {formatCurrency(day.totalAmount || 0)}
              </span>
            </div>

            {/* Hạn mức (Payment Limit) */}
            <div className="text-muted-foreground">
              Mức: <span className="text-foreground font-medium tabular-nums">
                {formatCurrency(paymentLimit)}
              </span>
            </div>

            {/* Đã trả (Paid Amount) */}
            <div className="text-muted-foreground">
              Đã trả: <span className="text-foreground font-medium tabular-nums">
                {formatCurrency(day.totalPaidAmount || 0)}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Empty Day Placeholder */}
      {!hasEntries && isCurrentMonth && (
        <div className="flex-1 flex items-start justify-center pt-4">
          <div className={cn(
            "text-4xl font-bold opacity-0 group-hover:opacity-100 group-hover:animate-bounce group-hover:scale-150 transition-all duration-300 ease-out",
            isSundayColumn ? "text-white/40" : "text-primary"
          )}>
            +
          </div>
        </div>
      )}
    </div>
  );
}

// Mobile variant with condensed layout
export function CalendarDayCellMobile({
  day,
  onDayClick,
  className,
  weekIndex = 0,
  totalWeeks = 1,
  bulkTransferPercentage = 0
}: CalendarDayCellProps) {

  const hasEntries = day.entries.length > 0;
  const isCurrentMonth = day.isCurrentMonth;
  const isToday = day.isToday;
  const isSundayColumn = className?.includes('sunday-gradient-cell');

  // Calculate payment limit for employee view using aggregated total
  const paymentLimit = hasEntries ? day.totalAmount * bulkTransferPercentage : 0;

  // Calculate gradient intensity based on week position (0 = lightest, 1 = darkest)

  // Generate slate background color based on gradient intensity
  const getSundayBackgroundClass = () => {
    if (!isSundayColumn) return '';

    // Sunday columns carry white text and light indicator icons, so every
    // rung of the week gradient stays dark enough for 4.5:1 (white/90) —
    // flattening the ramp to slate-600 was cheaper than per-rung text colors.
    // (bg-muted/500 was never a valid Tailwind class and rendered transparent.)
    return 'bg-slate-600 border-border hover:bg-slate-500';
  };

  const handleDayClick = () => {
    if (onDayClick) {
      onDayClick(day);
    }
  };

  // Get status-based styling for mobile using aggregated status
  const getStatusStyling = () => {
    if (!hasEntries) return '';
    return getCalendarCellClasses(day.aggregatedStatus);
  };

  return (
    <div
      className={cn(
        "min-h-[60px] border relative flex flex-col text-xs group",
        "transition-colors cursor-pointer",
        !isCurrentMonth && !isSundayColumn ? "hover:bg-slate-700" : (!isCurrentMonth && isSundayColumn ? "" : "hover:bg-muted/30"),
        {
          "bg-slate-800 border-border" : !isCurrentMonth,
          "opacity-80": !isCurrentMonth,
          "ring-1 ring-primary ring-inset border-primary": isToday,
          "hover:ring-1 hover:ring-primary/40 hover:ring-inset border-border": !hasEntries && isCurrentMonth && !isToday,
          "border-border": !hasEntries && isCurrentMonth && !isToday,
        },
        // Apply Sunday gradient only when there are no entries
        isSundayColumn && !hasEntries ? getSundayBackgroundClass() : '',
        // Status styling takes precedence when there are entries
        hasEntries && isCurrentMonth ? getStatusStyling() : '',
        className
      )}
      onClick={handleDayClick}
    >
      {/* Day Number & Status Dot */}
      <div className="flex justify-between items-center p-1.5">
        <span
          className={cn(
            "typography-label-medium font-medium",
            !isCurrentMonth && !isSundayColumn && "text-slate-900",
            !isCurrentMonth && isSundayColumn && "text-white/90",
            isCurrentMonth && !hasEntries && isToday && !isSundayColumn && "text-primary font-semibold",
            isCurrentMonth && !hasEntries && isToday && isSundayColumn && "text-white font-semibold",
            isCurrentMonth && !hasEntries && !isToday && !isSundayColumn && "text-foreground",
            isCurrentMonth && !hasEntries && !isToday && isSundayColumn && "text-white",
            hasEntries && isToday && "text-foreground font-semibold",
            hasEntries && !isToday && isCurrentMonth && "text-foreground"
          )}
        >
          {day.dayNumber}
        </span>
        {/* Force Payroll Icon */}
        {day.hasForcePayroll && (
          <span className="inline-flex items-center" aria-label="Đánh dấu xuất hiện trong kỳ trả lương tiếp theo">
            <Star className={cn('h-3 w-3', isSundayColumn && !hasEntries ? 'text-yellow-800 fill-yellow-800' : 'text-yellow-800 fill-yellow-800')} />
          </span>
        )}
      </div>

      {/* Condensed Entry Info - Using aggregated totals */}
      {hasEntries && isCurrentMonth && (
        <div className="flex-1 px-1.5 pb-1.5">
          <div className="space-y-0.5 typography-label-small">
            {/* Ca làm việc */}
            <div className="text-muted-foreground">
              GL: <span className="text-foreground font-medium">{day.totalHours} giờ</span>
            </div>
            {/* Lương */}
            <div className="text-muted-foreground">
              L: <span className="text-foreground font-medium tabular-nums">{formatCurrency(day.totalAmount || 0)}</span>
            </div>
            {/* Hạn mức */}
            <div className="text-muted-foreground">
              HM: <span className="text-foreground font-medium tabular-nums">{formatCurrency(paymentLimit)}</span>
            </div>
            {/* Đã trả */}
            <div className="text-muted-foreground">
              ĐT: <span className="text-foreground font-medium tabular-nums">{formatCurrency(day.totalPaidAmount || 0)}</span>
            </div>
          </div>
        </div>
      )}

      {/* Empty Day Placeholder - Mobile */}
      {!hasEntries && isCurrentMonth && (
        <div className="flex-1 flex items-start justify-center pt-2">
          <div className={cn(
            "text-2xl font-bold opacity-0 group-hover:opacity-100 group-hover:animate-bounce group-hover:scale-150 transition-all duration-300 ease-out",
            isSundayColumn ? "text-white/40" : "text-primary"
          )}>
            +
          </div>
        </div>
      )}
    </div>
  );
}
