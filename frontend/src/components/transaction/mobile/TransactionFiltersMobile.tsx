import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import { MobileSearchInput } from '@/components/shared/MobileSearchInput';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { CalendarDays, X } from 'lucide-react';
import { dateToString } from '@/utils/dateHelpers';
import { useMetadata } from '@/contexts';
import { transactionService } from '@/services/api/transaction.service';
import type { TransactionFilters as FiltersType } from '@/services/api/transaction.service';

interface TransactionFiltersMobileProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
}

const DATE_PRESETS = [
  { label: 'Hôm nay', value: 'today' },
  { label: 'Tuần này', value: 'this-week' },
  { label: 'Tháng này', value: 'this-month' },
  { label: 'Tháng trước', value: 'last-month' },
  { label: 'Quý này', value: 'this-quarter' },
  { label: '6 tháng', value: 'last-6-months' },
  { label: 'Năm này', value: 'this-year' },
];

function getDatePreset(preset: string) {
  const now = new Date();
  switch (preset) {
    case 'today': return { fromDate: dateToString(now), toDate: dateToString(now) };
    case 'this-week': {
      const s = new Date(now); s.setDate(now.getDate() - now.getDay());
      const e = new Date(s); e.setDate(s.getDate() + 6);
      return { fromDate: dateToString(s), toDate: dateToString(e) };
    }
    case 'this-month': return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth(), 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
    };
    case 'last-month': return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth() - 1, 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth(), 0)),
    };
    case 'this-quarter': {
      const q = Math.floor(now.getMonth() / 3);
      return {
        fromDate: dateToString(new Date(now.getFullYear(), q * 3, 1)),
        toDate: dateToString(new Date(now.getFullYear(), q * 3 + 3, 0)),
      };
    }
    case 'last-6-months': return {
      fromDate: dateToString(new Date(now.getFullYear(), now.getMonth() - 6, 1)),
      toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
    };
    case 'this-year': return {
      fromDate: dateToString(new Date(now.getFullYear(), 0, 1)),
      toDate: dateToString(new Date(now.getFullYear(), 11, 31)),
    };
    default: return null;
  }
}

export function TransactionFiltersMobile({
  filters,
  onFiltersChange,
  onClearFilters,
  hasFilters,
  totalResults,
}: TransactionFiltersMobileProps) {
  const [showDateSheet, setShowDateSheet] = useState(false);
  const { transactionMetadata, isLoadingTransactionMetadata } = useMetadata();

  const set = (key: keyof FiltersType, value: unknown) =>
    onFiltersChange({ ...filters, [key]: value, page: 1 });

  const dateLabel = filters.fromDate && filters.toDate
    ? `${filters.fromDate} – ${filters.toDate}`
    : 'Khoảng thời gian';

  return (
    <div className="space-y-2">
      {/* Search */}
      <MobileSearchInput
        value={filters.search ?? ''}
        onSearch={(value) => set('search', value || undefined)}
        placeholder="Tìm kiếm diễn giải, đối tượng..."
      />

      {/* Row 1: date picker + clear */}
      <div className="flex gap-2">
        <Sheet open={showDateSheet} onOpenChange={setShowDateSheet}>
          <SheetTrigger asChild>
            <Button variant="outline" className="flex-1 h-11 justify-start gap-2 text-sm font-normal">
              <CalendarDays className="h-4 w-4 text-muted-foreground shrink-0" />
              <span className="truncate text-left">{dateLabel}</span>
            </Button>
          </SheetTrigger>
          <SheetContent
            side="bottom"
            className="h-auto max-h-[85dvh] overflow-y-auto pb-[calc(1.25rem+env(safe-area-inset-bottom))]"
          >
            <SheetHeader><SheetTitle>Chọn khoảng thời gian</SheetTitle></SheetHeader>
            <div className="mt-4 space-y-4 pb-6">
              <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
                {DATE_PRESETS.map((p) => (
                  <Button
                    key={p.value}
                    variant="outline"
                    className="h-11 text-sm"
                    onClick={() => {
                      const range = getDatePreset(p.value);
                      if (range) { onFiltersChange({ ...filters, ...range, page: 1 }); setShowDateSheet(false); }
                    }}
                  >
                    {p.label}
                  </Button>
                ))}
              </div>
              <div className="border-t pt-4 space-y-2">
                <p className="text-sm font-medium text-muted-foreground">Tùy chỉnh</p>
                <DateRangePicker
                  variant="mobile"
                  startDate={filters.fromDate}
                  endDate={filters.toDate}
                  onStartDateChange={(date) => set('fromDate', date)}
                  onEndDateChange={(date) => set('toDate', date)}
                />
              </div>
            </div>
          </SheetContent>
        </Sheet>

        {hasFilters && (
          <Button variant="ghost" size="icon" className="shrink-0" onClick={onClearFilters} aria-label="Xóa bộ lọc">
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>

      {/* Row 2: type + status */}
      <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2">
        {isLoadingTransactionMetadata ? (
          <>
            <Skeleton className="h-10 rounded-xl" />
            <Skeleton className="h-10 rounded-xl" />
          </>
        ) : (
          <>
            <SearchableSelect
              value={filters.transaction_type || 'all'}
              onChange={(v) => set('transaction_type', v === 'all' ? undefined : v)}
              placeholder="Loại GD"
              searchPlaceholder="Tìm loại giao dịch..."
              triggerClassName="text-sm"
              options={[
                { value: 'all', label: 'Tất cả loại' },
                ...(transactionMetadata?.transaction_types.map((t) => ({
                  value: t.type,
                  label: t.label,
                })) ?? []),
              ]}
            />

            <Select
              value={filters.status || 'all'}
              onValueChange={(v) => set('status', v === 'all' ? undefined : v)}
            >
              <SelectTrigger className="h-11 text-sm">
                <SelectValue placeholder="Trạng thái" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Tất cả</SelectItem>
                {transactionMetadata?.statuses.map((s) => (
                  <SelectItem key={s.type} value={s.type}>
                    {transactionService.getStatusDisplay(s.type, transactionMetadata)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </>
        )}
      </div>

      {totalResults !== undefined && (
        <p className="text-xs text-muted-foreground text-center">
          {totalResults.toLocaleString('vi-VN')} kết quả
        </p>
      )}
    </div>
  );
}
