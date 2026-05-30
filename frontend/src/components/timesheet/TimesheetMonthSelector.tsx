import { useState, useCallback } from 'react';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Calendar } from '@/components/ui/calendar';
import { ChevronLeft, ChevronRight, CalendarDays } from 'lucide-react';
import { format, addMonths, subMonths, startOfMonth, parse } from 'date-fns';
import { vi } from 'date-fns/locale';
import { cn } from '@/lib/utils';

interface TimesheetMonthSelectorProps {
  value: string;
  onChange: (value: string) => void;
  className?: string;
}

export const TimesheetMonthSelector: React.FC<TimesheetMonthSelectorProps> = ({
  value,
  onChange,
  className,
}) => {
  const [isCalendarOpen, setIsCalendarOpen] = useState(false);

  const selectedDate = value !== 'all'
    ? startOfMonth(parse(value + '-01', 'yyyy-MM-dd', new Date()))
    : startOfMonth(new Date());

  const handlePreviousMonth = useCallback(() => {
    const current = value !== 'all' ? selectedDate : new Date();
    onChange(format(subMonths(current, 1), 'yyyy-MM'));
  }, [value, selectedDate, onChange]);

  const handleNextMonth = useCallback(() => {
    const current = value !== 'all' ? selectedDate : new Date();
    onChange(format(addMonths(current, 1), 'yyyy-MM'));
  }, [value, selectedDate, onChange]);

  const handleMonthSelect = useCallback((date: Date | undefined) => {
    if (date) {
      onChange(format(startOfMonth(date), 'yyyy-MM'));
      setIsCalendarOpen(false);
    }
  }, [onChange]);

  const handleShowAll = useCallback(() => {
    onChange('all');
  }, [onChange]);

  // "Tháng 5" / "2026"
  const monthLabel = format(selectedDate, 'MMMM', { locale: vi });   // "tháng 5"
  const monthShort = format(selectedDate, 'MM/yy');                   // "05/26"
  const yearLabel  = format(selectedDate, 'yyyy');

  return (
    <div className={cn('flex items-center gap-2', className)}>
      {/* "Tất cả" toggle */}
      <button
        onClick={handleShowAll}
        className={cn(
          'h-8 rounded-lg px-3 text-xs font-semibold transition-colors whitespace-nowrap',
          value === 'all'
            ? 'bg-foreground text-background'
            : 'text-muted-foreground hover:text-foreground hover:bg-muted',
        )}
      >
        Tất cả
      </button>

      {/* Unified navigator pill */}
      <div
        className={cn(
          'flex items-center overflow-hidden rounded-xl border border-border/70 bg-card shadow-sm transition-shadow hover:shadow-md',
          value === 'all' && 'opacity-50 pointer-events-none',
        )}
      >
        {/* ← Prev */}
        <button
          onClick={handlePreviousMonth}
          aria-label="Tháng trước"
          className="flex h-8 w-8 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <ChevronLeft className="h-3.5 w-3.5" />
        </button>

        <div className="h-4 w-px bg-border/70" />

        {/* Month display — opens calendar popover */}
        <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
          <PopoverTrigger asChild>
            <button
              className="flex h-8 items-center gap-1.5 px-3 transition-colors hover:bg-muted"
              aria-label="Chọn tháng"
            >
              <CalendarDays className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              {/* Full label on sm+, short on xs */}
              <span className="hidden sm:flex items-baseline gap-1 whitespace-nowrap text-sm">
                <span className="capitalize font-medium text-muted-foreground">{monthLabel}</span>
                <span className="text-muted-foreground/40">·</span>
                <span className="font-semibold text-foreground">{yearLabel}</span>
              </span>
              <span className="sm:hidden text-sm font-semibold text-foreground whitespace-nowrap">
                {monthShort}
              </span>
            </button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-0" align="end">
            <Calendar
              mode="single"
              selected={value !== 'all' ? selectedDate : undefined}
              onSelect={handleMonthSelect}
              defaultMonth={selectedDate}
              locale={vi}
            />
          </PopoverContent>
        </Popover>

        <div className="h-4 w-px bg-border/70" />

        {/* → Next */}
        <button
          onClick={handleNextMonth}
          aria-label="Tháng sau"
          className="flex h-8 w-8 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
        >
          <ChevronRight className="h-3.5 w-3.5" />
        </button>
      </div>
    </div>
  );
};
