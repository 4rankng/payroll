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
    <div
      className={cn(
        'ct-join grid w-full grid-cols-[auto_auto_minmax(0,1fr)_auto] overflow-hidden rounded-xl border border-border/80 bg-background shadow-sm sm:w-auto',
        className,
      )}
    >
      <button
        type="button"
        onClick={handleShowAll}
        className={cn(
          'ct-join-item h-11 border-r border-border/70 px-3 text-xs font-semibold transition-colors whitespace-nowrap',
          value === 'all'
            ? 'bg-foreground text-background'
            : 'bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        )}
      >
        Tất cả
      </button>

      <button
        type="button"
        onClick={handlePreviousMonth}
        aria-label="Tháng trước"
        className="ct-join-item flex h-11 w-11 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <ChevronLeft className="h-3.5 w-3.5" />
      </button>

      <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="ct-join-item flex h-11 min-w-0 items-center justify-center gap-1.5 bg-muted/35 px-2.5 transition-colors hover:bg-muted sm:px-3"
            aria-label="Chọn tháng"
          >
            <CalendarDays className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <span className="hidden items-baseline gap-1 whitespace-nowrap text-sm sm:flex">
              <span className="capitalize font-medium text-muted-foreground">{monthLabel}</span>
              <span className="text-muted-foreground/40">·</span>
              <span className="font-semibold text-foreground">{yearLabel}</span>
            </span>
            <span className="whitespace-nowrap text-sm font-semibold text-foreground sm:hidden">
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

      <button
        type="button"
        onClick={handleNextMonth}
        aria-label="Tháng sau"
        className="ct-join-item flex h-11 w-11 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        <ChevronRight className="h-3.5 w-3.5" />
      </button>
    </div>
  );
};
