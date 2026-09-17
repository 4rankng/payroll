import { useState, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { SearchableSelect } from '@/components/ui/searchable-select';
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from '@/components/ui/sheet';
import { CalendarDays, Filter, X } from 'lucide-react';
import { format } from 'date-fns';
import { ledgerService } from '@/services/api/ledger.service';
import { dateToString } from '@/utils/dateHelpers';
import { DateRangePicker } from '@/components/ui/date-range-picker';
import { SearchBar } from '@/components/shared/SearchBar';
import type { LedgerFilters as FiltersType, AccountMetadata } from '@/types/api/financial.types';

interface Project { id: number; name: string; }

interface LedgerFiltersMobileProps {
  filters: FiltersType;
  onFiltersChange: (filters: FiltersType) => void;
  projects: Project[];
  onClearFilters: () => void;
  hasFilters: boolean;
  totalResults?: number;
  accountMetadata?: AccountMetadata[];
  isLoadingAccountMetadata?: boolean;
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

export function LedgerFiltersMobile({
  filters,
  onFiltersChange,
  projects,
  onClearFilters,
  hasFilters,
  totalResults,
  accountMetadata = [],
  isLoadingAccountMetadata = false,
}: LedgerFiltersMobileProps) {
  const [showDateSheet, setShowDateSheet] = useState(false);
  const [showFiltersSheet, setShowFiltersSheet] = useState(false);

  const set = (key: keyof FiltersType, value: unknown) =>
    onFiltersChange({ ...filters, [key]: value, page: 1 });

  const accountOptions = useMemo(() =>
    accountMetadata.length > 0
      ? ledgerService.getAccountOptionsFromMetadata(accountMetadata)
      : ledgerService.getAccountOptions(),
    [accountMetadata]
  );

  const activeFilterCount = useMemo(() => {
    let n = 0;
    if (filters.account) n++;
    if (filters.project_id) n++;
    if (filters.party) n++;
    if (filters.created_by) n++;
    if (filters.has_evidence !== undefined) n++;
    return n;
  }, [filters]);

  const dateLabel = filters.fromDate && filters.toDate
    ? (() => {
        try {
          const from = format(new Date(filters.fromDate + 'T00:00:00'), 'dd/MM/yyyy');
          const to = format(new Date(filters.toDate + 'T00:00:00'), 'dd/MM/yyyy');
          return `${from} – ${to}`;
        } catch {
          return `${filters.fromDate} – ${filters.toDate}`;
        }
      })()
    : 'Khoảng thời gian';

  return (
    <div className="space-y-2">
      {/* Search */}
      <SearchBar
        searchTerm={filters.party || ''}
        onSearchChange={(v) => set('party', v || undefined)}
        placeholder="Tìm diễn giải, đối tượng..."
        className="w-full h-11 text-sm"
      />

      {/* Row: date + filter button + clear */}
      <div className="flex gap-2">
        {/* Date picker */}
        <Sheet open={showDateSheet} onOpenChange={setShowDateSheet}>
          <SheetTrigger asChild>
            <Button variant="outline" className="flex-1 h-11 justify-start gap-2 text-sm font-normal min-w-0">
              <CalendarDays className="h-4 w-4 text-muted-foreground shrink-0" />
              <span className="truncate">{dateLabel}</span>
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

        {/* Advanced filters */}
        <Sheet open={showFiltersSheet} onOpenChange={setShowFiltersSheet}>
          <SheetTrigger asChild>
            <Button variant="outline" className="h-11 px-3 gap-1.5 text-sm shrink-0 relative">
              <Filter className="h-4 w-4" />
              Lọc
              {activeFilterCount > 0 && (
                <Badge className="absolute -top-1.5 -right-1.5 h-4 w-4 p-0 text-xs flex items-center justify-center rounded-full">
                  {activeFilterCount}
                </Badge>
              )}
            </Button>
          </SheetTrigger>
          <SheetContent
            side="bottom"
            className="h-auto max-h-[85dvh] overflow-y-auto pb-[calc(1.25rem+env(safe-area-inset-bottom))]"
          >
            <SheetHeader><SheetTitle>Bộ lọc nâng cao</SheetTitle></SheetHeader>
            <div className="mt-4 space-y-4 pb-6">
              {/* Account type */}
              <div className="space-y-1.5">
                <p className="text-sm font-medium">Loại tài khoản</p>
                <SearchableSelect
                  value={filters.account || 'all'}
                  onChange={(v) => set('account', v === 'all' ? undefined : v)}
                  options={[{ value: 'all', label: 'Tất cả' }, ...accountOptions]}
                  disabled={isLoadingAccountMetadata}
                  searchPlaceholder="Tìm loại tài khoản..."
                />
              </div>

              {/* Project */}
              <div className="space-y-1.5">
                <p className="text-sm font-medium">Dự án</p>
                <SearchableSelect
                  value={filters.project_id?.toString() || 'all'}
                  onChange={(v) => set('project_id', v === 'all' ? undefined : parseInt(v))}
                  options={[
                    { value: 'all', label: 'Tất cả' },
                    ...projects.map((p) => ({ value: p.id.toString(), label: p.name })),
                  ]}
                  searchPlaceholder="Tìm dự án..."
                />
              </div>

              {/* Evidence */}
              <div className="space-y-1.5">
                <p className="text-sm font-medium">Chứng từ</p>
                <Select
                  value={filters.has_evidence === undefined ? 'all' : String(filters.has_evidence)}
                  onValueChange={(v) => set('has_evidence', v === 'all' ? undefined : v === 'true')}
                >
                  <SelectTrigger className="h-11">
                    <SelectValue placeholder="Tất cả" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Tất cả</SelectItem>
                    <SelectItem value="true">Có chứng từ</SelectItem>
                    <SelectItem value="false">Không có chứng từ</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {hasFilters && (
                <Button
                  variant="outline"
                  className="w-full h-11 gap-2"
                  onClick={() => { onClearFilters(); setShowFiltersSheet(false); }}
                >
                  <X className="h-4 w-4" />
                  Xóa tất cả bộ lọc
                </Button>
              )}
            </div>
          </SheetContent>
        </Sheet>

        {hasFilters && (
          <Button variant="ghost" size="icon" className="shrink-0" onClick={onClearFilters} aria-label="Xóa bộ lọc">
            <X className="h-4 w-4" />
          </Button>
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
