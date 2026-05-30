import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Calendar } from '@/components/ui/calendar';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from 'lucide-react';
import { format, addMonths, subMonths, startOfMonth } from 'date-fns';
import { vi } from 'date-fns/locale';

interface TimesheetDateHeaderProps {
  selectedDate: Date;
  onDateChange: (date: Date) => void;
}

export function TimesheetDateHeader({ selectedDate, onDateChange }: TimesheetDateHeaderProps) {
  const [isCalendarOpen, setIsCalendarOpen] = useState(false);

  const handlePreviousMonth = () => {
    onDateChange(subMonths(selectedDate, 1));
  };

  const handleNextMonth = () => {
    onDateChange(addMonths(selectedDate, 1));
  };

  const handleMonthSelect = (date: Date | undefined) => {
    if (date) {
      onDateChange(startOfMonth(date));
      setIsCalendarOpen(false);
    }
  };

  return (
    <div className="flex items-center gap-1">
      <Button
        variant="outline"
        size="sm"
        onClick={handlePreviousMonth}
        className="h-8 w-8 p-0"
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>

      <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
        <PopoverTrigger asChild>
          <Button 
            variant="outline" 
            className="h-8 px-3 typography-body-medium"
          >
            <CalendarIcon className="mr-2 h-4 w-4" />
            {format(selectedDate, 'MM/yyyy', { locale: vi })}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="start">
          <Calendar
            mode="single"
            selected={selectedDate}
            onSelect={handleMonthSelect}
            defaultMonth={selectedDate}
            locale={vi}
          />
        </PopoverContent>
      </Popover>

      <Button
        variant="outline"
        size="sm"
        onClick={handleNextMonth}
        className="h-8 w-8 p-0"
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
}