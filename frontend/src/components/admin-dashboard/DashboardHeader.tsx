import { useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/shared/PageHeader';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Calendar } from '@/components/ui/calendar';
import { ChevronLeft, ChevronRight, Calendar as CalendarIcon, RefreshCcw } from 'lucide-react';
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
      description="Theo dõi số liệu vận hành, nhân sự và tài chính theo kỳ"
    >
      <div className="admin-dashboard-header flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleShowAll}
          className={`min-h-11 justify-center rounded-2xl border px-3 text-xs font-semibold sm:h-10 sm:min-h-10 ${
            value === 'all'
              ? 'border-primary/20 bg-primary/10 text-primary'
              : 'border-border/60 bg-background text-muted-foreground hover:bg-muted/40 hover:text-foreground'
          }`}
        >
          <RefreshCcw className="mr-2 h-3.5 w-3.5" />
          Tất cả kỳ
        </Button>

        <div className="grid grid-cols-[44px_minmax(0,1fr)_44px] gap-1.5 sm:flex sm:items-center">
          <Button
            variant="outline"
            size="sm"
            onClick={handlePreviousMonth}
            className="h-11 w-11 rounded-2xl border-border/60 bg-background p-0 sm:h-10 sm:w-10"
            aria-label="Tháng trước"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>

          <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                className={`h-11 min-w-0 rounded-2xl border-border/60 bg-background px-3 text-sm font-semibold sm:h-10 sm:min-w-[148px] ${
                  value === 'all' ? 'opacity-70' : ''
                }`}
              >
                <CalendarIcon className="mr-2 h-4 w-4 shrink-0" />
                <span className="truncate">
                  {value === 'all' ? 'Tất cả' : format(selectedDate, 'MM/yyyy', { locale: vi })}
                </span>
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
            className="h-11 w-11 rounded-2xl border-border/60 bg-background p-0 sm:h-10 sm:w-10"
            aria-label="Tháng sau"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </PageHeader>
  );
};
