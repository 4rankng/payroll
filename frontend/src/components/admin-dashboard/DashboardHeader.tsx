import { useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/shared/PageHeader';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { MonthPicker } from '@/components/ui/month-picker';
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

  const handleMonthSelect = useCallback((month: string) => {
    onChange(month);
    setIsCalendarOpen(false);
  }, [onChange]);

  const handleShowAll = useCallback(() => {
    onChange('all');
  }, [onChange]);

  return (
    <PageHeader
      title="Tổng quan"
      description="Theo dõi số liệu vận hành, nhân sự và tài chính theo kỳ"
    >
      <div className="admin-dashboard-header flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center sm:gap-1 sm:rounded-xl sm:border sm:border-border/70 sm:bg-card sm:p-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleShowAll}
          className={`h-11 min-h-11 justify-center rounded-xl border px-3 text-xs font-semibold sm:h-9 sm:min-h-9 ${
            value === 'all'
              ? 'border-primary/20 bg-primary/10 text-primary'
              : 'border-transparent bg-transparent text-muted-foreground hover:bg-muted/60 hover:text-foreground'
          }`}
        >
          <RefreshCcw className="mr-2 h-3.5 w-3.5" />
          Tất cả kỳ
        </Button>

        <div className="grid grid-cols-[44px_minmax(0,1fr)_44px] gap-1.5 sm:flex sm:items-center sm:gap-1">
          <Button
            variant="outline"
            size="sm"
            onClick={handlePreviousMonth}
            className="h-11 w-11 rounded-xl border-transparent bg-transparent p-0 hover:bg-muted/60 sm:h-9 sm:w-9"
            aria-label="Tháng trước"
          >
            <ChevronLeft className="h-4 w-4" />
          </Button>

          <Popover open={isCalendarOpen} onOpenChange={setIsCalendarOpen}>
            <PopoverTrigger asChild>
              <Button
                variant="outline"
                aria-label="Chọn tháng"
                className={`h-11 min-w-0 rounded-xl border-transparent bg-muted/55 px-3 text-sm font-semibold sm:h-9 sm:min-w-[136px] ${
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
              <MonthPicker value={value} onChange={handleMonthSelect} />
            </PopoverContent>
          </Popover>

          <Button
            variant="outline"
            size="sm"
            onClick={handleNextMonth}
            className="h-11 w-11 rounded-xl border-transparent bg-transparent p-0 hover:bg-muted/60 sm:h-9 sm:w-9"
            aria-label="Tháng sau"
          >
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </PageHeader>
  );
};
