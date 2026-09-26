import { useState } from 'react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { CalendarDays } from 'lucide-react';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { cn } from '@/lib/utils';

interface DateRangeCalendarPickerProps {
  startDate: string;
  endDate: string;
  onStartDateChange: (date: string) => void;
  onEndDateChange: (date: string) => void;
  disabled?: boolean;
  className?: string;
  ariaLabelledBy?: string;
}

function parseLocalDate(value: string): Date | undefined {
  if (!value) return undefined;
  const [year, month, day] = value.split('-').map(Number);
  if (!year || !month || !day) return undefined;
  return new Date(year, month - 1, day);
}

function formatApiDate(date: Date): string {
  return format(date, 'yyyy-MM-dd');
}

function formatDisplayDate(value: string): string {
  const date = parseLocalDate(value);
  return date ? format(date, 'dd/MM/yyyy', { locale: vi }) : 'Chọn ngày';
}

export function DateRangeCalendarPicker({
  startDate,
  endDate,
  onStartDateChange,
  onEndDateChange,
  disabled = false,
  className,
  ariaLabelledBy,
}: DateRangeCalendarPickerProps) {
  const [startOpen, setStartOpen] = useState(false);
  const [endOpen, setEndOpen] = useState(false);
  const selectedStartDate = parseLocalDate(startDate);
  const selectedEndDate = parseLocalDate(endDate);

  return (
    <div
      role="group"
      aria-labelledby={ariaLabelledBy}
      className={cn(
        'flex min-h-11 w-full min-w-0 items-center gap-2 overflow-hidden rounded-lg border border-input bg-card px-3',
        disabled && 'cursor-not-allowed opacity-50',
        className,
      )}
    >
      <CalendarDays className="h-4 w-4 shrink-0 text-muted-foreground" aria-hidden="true" />

      <Popover open={startOpen} onOpenChange={setStartOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            disabled={disabled}
            className="min-h-11 min-w-0 flex-1 truncate rounded-md px-1 text-center text-xs font-medium tabular-nums outline-none transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none"
            aria-label={`Mở lịch chọn ngày bắt đầu, ${formatDisplayDate(startDate)}`}
          >
            {formatDisplayDate(startDate)}
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="start">
          <Calendar
            mode="single"
            selected={selectedStartDate}
            defaultMonth={selectedStartDate}
            onSelect={(date) => {
              if (!date) return;
              onStartDateChange(formatApiDate(date));
              setStartOpen(false);
            }}
            disabled={(date) => Boolean(selectedEndDate && date > selectedEndDate)}
            initialFocus
            locale={vi}
          />
        </PopoverContent>
      </Popover>

      <span className="shrink-0 select-none text-xs text-muted-foreground" aria-hidden="true">–</span>

      <Popover open={endOpen} onOpenChange={setEndOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            disabled={disabled}
            className="min-h-11 min-w-0 flex-1 truncate rounded-md px-1 text-center text-xs font-medium tabular-nums outline-none transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none"
            aria-label={`Mở lịch chọn ngày kết thúc, ${formatDisplayDate(endDate)}`}
          >
            {formatDisplayDate(endDate)}
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="end">
          <Calendar
            mode="single"
            selected={selectedEndDate}
            defaultMonth={selectedEndDate}
            onSelect={(date) => {
              if (!date) return;
              onEndDateChange(formatApiDate(date));
              setEndOpen(false);
            }}
            disabled={(date) => Boolean(selectedStartDate && date < selectedStartDate)}
            initialFocus
            locale={vi}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}
