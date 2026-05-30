import { ChevronLeft, ChevronRight, Calendar } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Calendar as CalendarComponent } from '@/components/ui/calendar';
import { format, addDays, subDays, isToday } from 'date-fns';
import { vi } from 'date-fns/locale';
import { useState } from 'react';

interface TimesheetDateNavigationProps {
  selectedDate: Date;
  onDateChange: (date: Date) => void;
}

export function TimesheetDateNavigation({
  selectedDate,
  onDateChange
}: TimesheetDateNavigationProps) {
  const [isCalendarOpen, setIsCalendarOpen] = useState(false);

  const goToPreviousDay = () => {
    onDateChange(subDays(selectedDate, 1));
  };

  const goToNextDay = () => {
    onDateChange(addDays(selectedDate, 1));
  };

  const goToToday = () => {
    onDateChange(new Date());
  };

  const handleDateSelect = (date: Date | undefined) => {
    if (date) {
      onDateChange(date);
      setIsCalendarOpen(false);
    }
  };

  return (
    <div className="flex items-center gap-2 bg-muted/50 px-2 py-1 rounded-xl">
      <Button
        variant="outline"
        size="sm"
        onClick={goToPreviousDay}
        className="h-6 w-6 p-0 bg-card border-border hover:bg-muted/50 hover:border-border"      >
        <ChevronLeft className="h-3 w-3" />
      </Button>
      
      <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            className="bg-card border border-border px-3 py-1 rounded-xl min-w-[120px] text-center hover:bg-muted/50 typography-body-small h-6"
          >
            {format(selectedDate, 'dd/MM/yyyy', { locale: vi })}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="center">
          <CalendarComponent
            mode="single"
            selected={selectedDate}
            onSelect={handleDateSelect}
            initialFocus
            locale={vi}
          />
        </PopoverContent>
      </Popover>
      
      <Button
        variant="outline"
        size="sm"
        onClick={goToNextDay}
        className="h-6 w-6 p-0 bg-card border-border hover:bg-muted/50 hover:border-border"      >
        <ChevronRight className="h-3 w-3" />
      </Button>
      
      {!isToday(selectedDate) && (
        <Button
          size="sm"
          onClick={goToToday}
          className="bg-blue-600 hover:bg-blue-700 text-white font-medium px-3 py-1 h-6 typography-body-small"
        >
          Hôm nay
        </Button>
      )}
    </div>
  );
}