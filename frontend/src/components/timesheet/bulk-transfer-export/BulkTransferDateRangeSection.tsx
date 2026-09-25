import { useState, useCallback } from 'react';
import { Label } from '@/components/ui/label';
import { Button } from '@/components/ui/button';
import { ButtonGroup } from '@/components/ui/button-group';
import { Calendar } from '@/components/ui/calendar';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { AlertCircle, Calendar as CalendarIcon } from 'lucide-react';
import { format } from 'date-fns';
import { vi } from 'date-fns/locale';
import { cn } from '@/lib/utils';
import { formatDateForAPI } from '@/utils/formatters';
import { formatWeekPeriodDisplay, type WeekPeriod, type WeekPeriodsResult } from '@/utils/weekPeriodHelpers';
import { type MonthPeriod, type MonthPeriodsResult } from '@/utils/monthPeriodHelpers';
import { type CustomDateRange } from '@/utils/weekPeriodHelpers';

const CUSTOM_VALUE = '__custom__';

interface BulkTransferDateRangeSectionProps {
  paymentSchedule: 'weekly' | 'monthly';
  fromDate: Date | undefined;
  toDate: Date | undefined;
  weekPeriods: WeekPeriodsResult;
  monthPeriods: MonthPeriodsResult;
  customDateRanges: CustomDateRange[];
  selectedCustomRange: string;
  hasDateError: boolean;
  isMobile: boolean;
  onFromDateChange: (date: Date | undefined) => void;
  onToDateChange: (date: Date | undefined) => void;
  onWeekPeriodApply: (period: WeekPeriod) => void;
  onMonthPeriodApply: (period: MonthPeriod) => void;
  onCustomRangeApply: (rangeLabel: string) => void;
}

export function BulkTransferDateRangeSection({
  paymentSchedule,
  fromDate,
  toDate,
  weekPeriods,
  monthPeriods,
  customDateRanges,
  selectedCustomRange,
  hasDateError,
  isMobile,
  onFromDateChange,
  onToDateChange,
  onWeekPeriodApply,
  onMonthPeriodApply,
  onCustomRangeApply,
}: BulkTransferDateRangeSectionProps) {
  const [fromDateOpen, setFromDateOpen] = useState(false);
  const [toDateOpen, setToDateOpen] = useState(false);
  const [showCustomPicker, setShowCustomPicker] = useState(false);

  // Determine if custom date pickers are active (dates don't match any preset period)
  const buttonGroupValue = (() => {
    if (showCustomPicker) return CUSTOM_VALUE;

    if (!(fromDate instanceof Date && toDate instanceof Date)) return '';

    const apiFrom = formatDateForAPI(fromDate);
    const apiTo = formatDateForAPI(toDate);

    // Check custom ranges
    if (customDateRanges.length > 0) {
      const match = customDateRanges.find(r => r.label === selectedCustomRange && r.from === apiFrom && r.to === apiTo);
      if (match) return match.label;
    }

    // Check standard week periods
    const weekMatch = weekPeriods.availablePeriods.find(p => p.from === apiFrom && p.to === apiTo);
    if (weekMatch) return `${weekMatch.from}_${weekMatch.to}`;

    // Dates don't match any preset — must be custom
    return CUSTOM_VALUE;
  })();

  const handleButtonGroupChange = useCallback((value: string) => {
    if (value === CUSTOM_VALUE) {
      setShowCustomPicker(true);
      return;
    }

    setShowCustomPicker(false);

    // Try custom range
    if (customDateRanges.length > 0) {
      const range = customDateRanges.find(r => r.label === value);
      if (range) {
        onCustomRangeApply(value);
        return;
      }
    }

    // Standard week period
    const [from, to] = value.split('_');
    const period = weekPeriods.availablePeriods.find(p => p.from === from && p.to === to);
    if (period) onWeekPeriodApply(period);
  }, [customDateRanges, onCustomRangeApply, weekPeriods.availablePeriods, onWeekPeriodApply]);

  const dateRangeSummary = (() => {
    if (!fromDate || !toDate) return null;
    const from = format(fromDate, 'dd/MM', { locale: vi });
    const to = format(toDate, 'dd/MM/yyyy', { locale: vi });
    return `${from} — ${to}`;
  })();

  // Build period options with shorter labels + "Tùy chỉnh" appended
  const periodOptions = (() => {
    if (customDateRanges.length > 0) {
      return [
        ...customDateRanges.map(r => {
          // Shorten "15 - 21 Tháng 04" → "15/04 – 21/04"
          const shortLabel = r.label.replace(/(\d+)\s*-\s*(\d+)\s*Tháng\s*(\d+)/, '$1/$3 – $2/$3');
          return { value: r.label, label: shortLabel };
        }),
        { value: CUSTOM_VALUE, label: 'Tùy chỉnh' },
      ];
    }
    return [
      ...weekPeriods.availablePeriods.map(p => ({
        value: `${p.from}_${p.to}`,
        label: formatWeekPeriodDisplay(p, isMobile),
      })),
      { value: CUSTOM_VALUE, label: 'Tùy chỉnh' },
    ];
  })();

  return (
    <div className="space-y-2.5">
      <Label className="block text-xs font-medium text-muted-foreground uppercase tracking-wide">
        Khoảng thời gian <span className="text-red-600">*</span>
      </Label>

      {paymentSchedule === 'weekly' ? (
        <>
          <ButtonGroup
            options={periodOptions}
            value={buttonGroupValue}
            onChange={handleButtonGroupChange}
            fullWidth
          />

          {/* Date range summary */}
          {dateRangeSummary && !showCustomPicker && (
            <p className="text-sm text-muted-foreground pl-0.5">{dateRangeSummary}</p>
          )}

          {/* Custom Date Inputs */}
          {showCustomPicker && (
            <div className="space-y-2 pt-1">
              <div className="grid grid-cols-1 gap-3 min-[420px]:grid-cols-2">
                <div className="space-y-1.5">
                  <Label htmlFor="from-date" className="text-xs">Từ ngày</Label>
                  <Popover open={fromDateOpen} onOpenChange={setFromDateOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        className={cn(
                          "min-h-11 w-full justify-start text-left font-normal",
                          !fromDate && "text-muted-foreground",
                          hasDateError && 'border-red-500 focus:ring-red-500'
                        )}
                      >
                        <CalendarIcon className="mr-1.5 h-3.5 w-3.5" />
                        {fromDate ? (
                          format(fromDate, "dd/MM/yyyy", { locale: vi })
                        ) : (
                          <span>Chọn ngày</span>
                        )}
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-auto p-0" align="start">
                      <Calendar
                        mode="single"
                        selected={fromDate}
                        onSelect={(date) => {
                          onFromDateChange(date);
                          setFromDateOpen(false);
                        }}
                        initialFocus
                        locale={vi}
                      />
                    </PopoverContent>
                  </Popover>
                </div>
                <div className="space-y-1.5">
                  <Label htmlFor="to-date" className="text-xs">Đến ngày</Label>
                  <Popover open={toDateOpen} onOpenChange={setToDateOpen}>
                    <PopoverTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        className={cn(
                          "min-h-11 w-full justify-start text-left font-normal",
                          !toDate && "text-muted-foreground",
                          hasDateError && 'border-red-500 focus:ring-red-500'
                        )}
                      >
                        <CalendarIcon className="mr-1.5 h-3.5 w-3.5" />
                        {toDate ? (
                          format(toDate, "dd/MM/yyyy", { locale: vi })
                        ) : (
                          <span>Chọn ngày</span>
                        )}
                      </Button>
                    </PopoverTrigger>
                    <PopoverContent className="w-auto p-0" align="start">
                      <Calendar
                        mode="single"
                        selected={toDate}
                        onSelect={(date) => {
                          onToDateChange(date);
                          setToDateOpen(false);
                        }}
                        initialFocus
                        locale={vi}
                        disabled={(date) => fromDate ? date < fromDate : false}
                      />
                    </PopoverContent>
                  </Popover>
                </div>
              </div>

              {hasDateError && (
                <div className="flex items-center gap-1.5 text-xs text-red-600">
                  <AlertCircle className="w-3.5 h-3.5" />
                  <span>Ngày kết thúc phải sau ngày bắt đầu</span>
                </div>
              )}
            </div>
          )}
        </>
      ) : (
        <>
          <ButtonGroup
            options={monthPeriods.availablePeriods.map((period) => ({
              value: period.month,
              label: period.label
            }))}
            value={
              fromDate instanceof Date && toDate instanceof Date
                ? monthPeriods.availablePeriods.find(p =>
                    p.from === formatDateForAPI(fromDate) &&
                    p.to === formatDateForAPI(toDate)
                  )?.month || ''
                : ''
            }
            onChange={(value) => {
              const period = monthPeriods.availablePeriods.find(p => p.month === value);
              if (period) onMonthPeriodApply(period);
            }}
          />

          {dateRangeSummary && (
            <p className="text-sm text-muted-foreground pl-0.5">{dateRangeSummary}</p>
          )}
        </>
      )}
    </div>
  );
}
