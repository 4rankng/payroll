import { useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/shared/PageHeader';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Calendar } from '@/components/ui/calendar';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon } from 'lucide-react';
import { format, addMonths, subMonths, startOfMonth, parse } from 'date-fns';
import { vi } from 'date-fns/locale';

interface DashboardHeaderProps {
  value: string; // 'all' | 'yyyy-MM'
  onChange: (value: string) => void;
}

export const DashboardHeader: React.FC<DashboardHeaderProps> = ({ value, onChange }) => {
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

  return (
    <PageHeader
      title="Tổng quan"
      description="Bảng điều khiển hệ thống quản lý nhân sự & lương"
    >
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleShowAll}
          className={`h-8 px-2 text-xs font-medium ${value === 'all' ? 'bg-muted text-foreground' : 'text-muted-foreground'}`}
        >
          Tất cả
        </Button>

        <Button
          variant="outline"
          size="sm"
          onClick={handlePreviousMonth}
          className="h-8 w-8 p-0"
          aria-label="Tháng trước"
        >
          <ChevronLeft className="h-4 w-4" />
        </Button>

        <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
          <PopoverTrigger asChild>
            <Button
              variant="outline"
              className={`h-8 px-3 typography-body-medium min-w-[120px] ${value === 'all' ? 'opacity-50' : ''}`}
            >
              <CalendarIcon className="mr-2 h-4 w-4" />
              {value === 'all' ? 'Tất cả' : format(selectedDate, 'MM/yyyy', { locale: vi })}
            </Button>
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

        <Button
          variant="outline"
          size="sm"
          onClick={handleNextMonth}
          className="h-8 w-8 p-0"
          aria-label="Tháng sau"
        >
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </PageHeader>
  );
};
