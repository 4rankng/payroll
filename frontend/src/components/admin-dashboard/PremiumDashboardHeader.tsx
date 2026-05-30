import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Calendar } from '@/components/ui/calendar';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from 'lucide-react';
import { format, addMonths, subMonths, startOfMonth } from 'date-fns';
import { vi } from 'date-fns/locale';

interface PremiumDashboardHeaderProps {
  selectedMonth: Date;
  onMonthChange: (date: Date) => void;
}

export const PremiumDashboardHeader: React.FC<PremiumDashboardHeaderProps> = ({ 
  selectedMonth, 
  onMonthChange 
}) => {
  const [isCalendarOpen, setIsCalendarOpen] = useState(false);

  const handlePreviousMonth = () => {
    onMonthChange(subMonths(selectedMonth, 1));
  };

  const handleNextMonth = () => {
    onMonthChange(addMonths(selectedMonth, 1));
  };

  const handleMonthSelect = (date: Date | undefined) => {
    if (date) {
      onMonthChange(startOfMonth(date));
      setIsCalendarOpen(false);
    }
  };

  return (
    <div className="mb-6 animate-hero-reveal">
      {/* Premium Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-4">
        <div>
          <h1 className="font-display font-extrabold text-4xl text-foreground tracking-tight">
            Dashboard
          </h1>
          <p className="text-base text-muted-foreground mt-1 font-medium">
            Payroll Management Overview
          </p>
        </div>

        {/* Premium Month Selector */}
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handlePreviousMonth}
            className="h-10 w-10 p-0 rounded-xl hover:bg-muted transition-colors"
            aria-label="Previous month"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>

          <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                className="h-10 px-4 font-semibold min-w-[140px] rounded-xl hover:bg-muted transition-colors"
              >
                <CalendarIcon className="mr-2 h-4 w-4" />
                {format(selectedMonth, 'MM/yyyy', { locale: vi })}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="end">
              <Calendar
                mode="single"
                selected={selectedMonth}
                onSelect={handleMonthSelect}
                defaultMonth={selectedMonth}
                locale={vi}
              />
            </PopoverContent>
          </Popover>

          <Button
            variant="outline"
            size="sm"
            onClick={handleNextMonth}
            className="h-10 w-10 p-0 rounded-xl hover:bg-muted transition-colors"
            aria-label="Next month"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
};
