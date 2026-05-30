import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import { useMetadata } from '@/contexts';
import { transactionService } from '@/services/api/transaction.service';
import type { TransactionFilters as FiltersType } from '@/services/api/transaction.service';

interface TransactionFiltersProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
}

export function TransactionFilters({
  filters,
  onFiltersChange,
  onClearFilters,
  hasFilters,
}: TransactionFiltersProps) {
  const { transactionMetadata } = useMetadata();

  const set = (key: keyof FiltersType, value: unknown) =>
    onFiltersChange({ ...filters, [key]: value, page: 1 });

  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      <SearchBar
        searchTerm={filters.search ?? ''}
        onSearchChange={(value) => set('search', value || undefined)}
        placeholder="Tìm kiếm diễn giải, đối tượng..."
      />

      <DateRangePicker
        startDate={filters.fromDate}
        endDate={filters.toDate}
        onStartDateChange={(date) => set('fromDate', date)}
        onEndDateChange={(date) => set('toDate', date)}
      />

      <FilterPill
        value={filters.transaction_type ?? 'all'}
        onChange={(v) => set('transaction_type', v === 'all' ? undefined : v)}
        placeholder="Tất cả loại"
        options={(transactionMetadata?.transaction_types ?? []).map(t => ({ value: t.type, label: t.label }))}
      />

      <FilterPill
        value={filters.status ?? 'all'}
        onChange={(v) => set('status', v === 'all' ? undefined : v)}
        placeholder="Trạng thái"
        options={(transactionMetadata?.statuses ?? []).map(s => ({
          value: s.type,
          label: transactionService.getStatusDisplay(s.type, transactionMetadata!),
        }))}
      />

      {hasFilters && (
        <button
          onClick={onClearFilters}
          className="text-xs text-muted-foreground hover:text-foreground transition-colors"
        >
          Xóa lọc
        </button>
      )}
    </div>
  );
}
