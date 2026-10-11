import { useMemo, useState } from 'react';
// UU PRO surface (ADR-012): @untitledui/icons + UU bridge tokens throughout.
import { Calendar, ChevronLeft, ChevronRight, Receipt } from '@untitledui/icons';

import { PageHeader } from '@/components/shared/PageHeader';
import { FilterBar } from '@/components/shared/FilterBar';
import { FilterPill } from '@/components/shared/FilterPill';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
} from '@/components/shared/AdminPageFrame';
import { EmptyState } from '@/components/shared/EmptyState';
import { MobilePagination } from '@/components/shared/MobilePagination';
import { SearchBar } from '@/components/shared/SearchBar';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { PaginationControls } from '@/components/ui/pagination-controls';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Skeleton } from '@/components/ui/skeleton';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { useBankTransferHistories } from '@/hooks/api/usePayrolls';
import {
  BANK_TRANSFER_CYCLE_LABELS,
  getCurrentMonthValue,
} from '@/utils/bankTransferHistoryHelpers';
import { formatCurrency } from '@/utils/formatters';
import { buildLedgerRows, LEDGER_COLUMNS } from './bankTransferLedger';
import { LedgerMobileRecord, LedgerTableRow } from './BankTransferLedgerRow';

// Derived from the shared labels so the filter options can never drift from
// BANK_TRANSFER_CYCLE_LABELS.
const CYCLE_OPTIONS = ([1, 2, 3, 4] as const).map(cycle => ({
  value: String(cycle),
  label: BANK_TRANSFER_CYCLE_LABELS[cycle],
}));

interface PayrollMonthPickerProps {
  value: string;
  onValueChange: (value: string) => void;
  className?: string;
}

function PayrollMonthPicker({ value, onValueChange, className }: PayrollMonthPickerProps) {
  const [selectedYear, selectedMonth] = value.split('-').map(Number);
  const [isOpen, setIsOpen] = useState(false);
  const [pickerYear, setPickerYear] = useState(selectedYear);
  const displayValue = `${String(selectedMonth).padStart(2, '0')}/${selectedYear}`;

  const handleOpenChange = (nextOpen: boolean) => {
    setIsOpen(nextOpen);
    if (nextOpen) setPickerYear(selectedYear);
  };

  const handleMonthSelect = (monthNumber: number) => {
    onValueChange(`${pickerYear}-${String(monthNumber).padStart(2, '0')}`);
    setIsOpen(false);
  };

  return (
    <Popover open={isOpen} onOpenChange={handleOpenChange}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="outline"
          aria-label={`Chọn tháng kỳ lương, hiện tại ${displayValue}`}
          className={`mt-0 h-11 w-full justify-between rounded-xl border-utility-gray-300 bg-utility-gray-50/70 px-3 font-sans text-[12px] font-normal tabular-nums text-fg-secondary shadow-none hover:border-utility-gray-300 hover:bg-utility-gray-50 focus-visible:bg-white sm:h-9 ${className ?? ''}`}
        >
          {displayValue}
          <Calendar className="h-4 w-4 text-fg-tertiary" aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[280px] rounded-xl p-3" align="start">
        <div className="flex items-center justify-between border-b border-utility-gray-200 pb-2">
          <button
            type="button"
            onClick={() => setPickerYear((year) => year - 1)}
            aria-label="Năm trước"
            className="flex h-11 w-11 items-center justify-center rounded-lg text-fg-tertiary transition-colors hover:bg-utility-gray-100 hover:text-fg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <p className="font-financial text-[13px] font-bold tabular-nums text-fg-primary">{pickerYear}</p>
          <button
            type="button"
            onClick={() => setPickerYear((year) => year + 1)}
            aria-label="Năm sau"
            className="flex h-11 w-11 items-center justify-center rounded-lg text-fg-tertiary transition-colors hover:bg-utility-gray-100 hover:text-fg-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
          >
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div className="mt-3 grid grid-cols-3 gap-2" role="group" aria-label={`Chọn tháng năm ${pickerYear}`}>
          {Array.from({ length: 12 }, (_, index) => {
            const monthNumber = index + 1;
            const monthLabel = String(monthNumber).padStart(2, '0');
            const isSelected = pickerYear === selectedYear && monthNumber === selectedMonth;

            return (
              <button
                key={monthNumber}
                type="button"
                onClick={() => handleMonthSelect(monthNumber)}
                aria-label={`Chọn tháng ${monthLabel} năm ${pickerYear}`}
                aria-pressed={isSelected}
                className={`min-h-11 rounded-lg border text-[12px] font-semibold tabular-nums transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600 ${
                  isSelected
                    ? 'border-emerald-700 bg-emerald-700 text-white'
                    : 'border-utility-gray-300 bg-white text-fg-secondary hover:border-emerald-300 hover:bg-emerald-50'
                }`}
              >
                {monthLabel}
              </button>
            );
          })}
        </div>
      </PopoverContent>
    </Popover>
  );
}

/**
 * One slim summary line instead of KPI cards — the month's answer at a glance
 * without spending vertical space the ledger needs.
 */
function SummaryBand({
  totalAmount,
  transferCount,
  employeeCount,
  monthLabel,
  isLoading,
}: {
  totalAmount?: number;
  transferCount?: number;
  employeeCount?: number;
  monthLabel: string;
  isLoading: boolean;
}) {
  return (
    <div
      data-slot="payment-history-summary"
      className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 border-b border-utility-gray-300 bg-utility-gray-50/60 px-3 py-1.5"
    >
      <span className="font-display text-[11px] font-bold uppercase tracking-[0.06em] text-fg-tertiary">
        Tổng đã chuyển
      </span>
      {isLoading ? (
        <Skeleton className="h-4 w-24 rounded-none" />
      ) : totalAmount === undefined ? (
        <span className="font-financial text-[13px] font-bold tabular-nums text-fg-tertiary">—</span>
      ) : (
        <span className="font-financial text-[13px] font-bold tabular-nums text-emerald-700">
          {formatCurrency(totalAmount)}
        </span>
      )}
      <span className="text-fg-tertiary" aria-hidden="true">·</span>
      <span className="font-financial text-xs font-semibold tabular-nums text-fg-primary">
        {isLoading || transferCount === undefined ? '—' : transferCount}
      </span>
      <span className="text-xs text-fg-tertiary">bút toán ngân hàng</span>
      <span className="text-fg-tertiary" aria-hidden="true">·</span>
      <span className="font-financial text-xs font-semibold tabular-nums text-fg-primary">
        {isLoading || employeeCount === undefined ? '—' : employeeCount}
      </span>
      <span className="text-xs text-fg-tertiary">nhân viên</span>
      <span className="ml-auto font-financial text-xs font-semibold tabular-nums text-fg-tertiary">
        {monthLabel}
      </span>
    </div>
  );
}

function HistorySkeleton() {
  return (
    <div className="divide-y divide-utility-gray-300" aria-label="Đang tải lịch sử trả lương" aria-busy="true">
      {Array.from({ length: 8 }).map((_, index) => (
        <Skeleton key={index} className="h-16 w-full rounded-none xl:h-10" />
      ))}
    </div>
  );
}

interface BankTransferHistoryPageContentProps {
  variant?: 'admin' | 'partner';
}

export function BankTransferHistoryPageContent({ variant = 'partner' }: BankTransferHistoryPageContentProps) {
  const isAdmin = variant === 'admin';
  const [month, setMonth] = useState(getCurrentMonthValue);
  const [cycle, setCycle] = useState('all');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);

  const filters = useMemo(() => ({
    month,
    cycle: cycle === 'all' ? undefined : Number(cycle),
    search: search.trim() || undefined,
    page,
    pageSize,
  }), [cycle, month, page, pageSize, search]);

  const { data, isLoading, isError, refetch } = useBankTransferHistories(filters);
  const records = useMemo(() => data?.data ?? [], [data]);
  const pagination = data?.pagination;
  const summary = data?.summary;

  const ledgerRows = useMemo(() => buildLedgerRows(records), [records]);

  const [yearPart, monthPart] = month.split('-');
  const monthLabel = `${monthPart}/${yearPart}`;

  const resetPage = () => setPage(1);

  const handlePageChange = (nextPage: number) => setPage(nextPage);

  // The records workspace (filters + summary + ledger + pagination) is shared
  // by both variants; only the page shell and the card treatment differ.
  const workspace = (
    <Card
      data-slot="payment-history-workspace"
      className={
        isAdmin
          ? 'admin-payment-history-workspace relative flex flex-col overflow-hidden rounded-2xl border-utility-gray-300 bg-white shadow-[0_1px_2px_rgba(16,24,40,0.04),0_20px_56px_-42px_rgba(8,120,62,0.26)] xl:overflow-visible'
          : 'overflow-hidden rounded-xl border-utility-gray-300 bg-white shadow-[0_12px_28px_-26px_rgba(15,23,42,0.42)] xl:overflow-visible xl:rounded-none'
      }
    >
      <div className="border-b border-utility-gray-300 p-2.5 sm:p-3 xl:py-1.5">
        <FilterBar className="gap-2">
          <PayrollMonthPicker
            value={month}
            onValueChange={(value) => { setMonth(value); resetPage(); }}
            className="sm:w-[128px]"
          />
          <FilterPill
            value={cycle}
            onChange={(value) => { setCycle(value); resetPage(); }}
            placeholder="Tất cả kỳ lương"
            options={CYCLE_OPTIONS}
            className="w-full sm:w-auto"
          />
          <SearchBar
            searchTerm={search}
            onSearchChange={(value) => { setSearch(value); resetPage(); }}
            placeholder="Tên nhân viên, mã chuyển khoản hoặc mã ngân hàng"
            className="w-full min-w-0 sm:min-w-[220px] sm:flex-1"
          />
        </FilterBar>
      </div>

      <SummaryBand
        totalAmount={summary?.total_amount}
        transferCount={summary?.transfer_count}
        employeeCount={summary?.employee_count}
        monthLabel={monthLabel}
        isLoading={isLoading}
      />

      {isLoading && <HistorySkeleton />}

      {isError && (
        <div className="bg-rose-50/70">
          <EmptyState
            title="Không thể tải lịch sử trả lương"
            description="Vui lòng kiểm tra kết nối và thử lại."
            action={{ label: 'Thử lại', onClick: () => refetch() }}
          />
        </div>
      )}

      {!isLoading && !isError && records.length === 0 && (
        <EmptyState
          title="Chưa có giao dịch đã hoàn tất"
          description="Chọn tháng hoặc kỳ lương khác để xem lịch sử."
        />
      )}

      {!isLoading && !isError && records.length > 0 && (
        <>
          {/* Wide screens: semantic ledger table, sticky column header. */}
          <div data-slot="payment-history-records" className="hidden xl:block">
            <Table className="w-full table-fixed">
              <colgroup>
                {LEDGER_COLUMNS.map((column) => (
                  <col key={column.id} style={column.width === 'auto' ? undefined : { width: column.width }} />
                ))}
              </colgroup>
              <TableHeader className="sticky top-0 z-10">
                <TableRow className="border-b border-utility-gray-300 bg-utility-gray-50/95 hover:bg-transparent">
                  {LEDGER_COLUMNS.map((column) => (
                    <TableHead
                      key={column.id}
                      scope="col"
                      className={`whitespace-normal px-2 font-display text-[11px] font-bold uppercase leading-tight tracking-[0.06em] text-fg-tertiary ${column.alignRight ? 'text-right' : ''}`}
                    >
                      {column.label}
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody className="[&_tr:last-child]:border-0">
                {ledgerRows.map((row) => (
                  <LedgerTableRow key={row.key} row={row} />
                ))}
              </TableBody>
            </Table>
          </div>

          {/* Narrow screens: dense record blocks, same data, nothing hidden. */}
          <div
            data-slot="payment-history-mobile-records"
            className="divide-y divide-utility-gray-300 xl:hidden"
            aria-label="Giao dịch đã hoàn tất"
          >
            {records.map((item) => (
              <LedgerMobileRecord
                key={`${item.work_month}-${item.cycle}-${item.employee_id}`}
                record={item}
              />
            ))}
          </div>
        </>
      )}

      {pagination && pagination.totalPages > 1 && (
        <div className="border-t border-utility-gray-300 px-2.5 sm:px-3">
          <div className="hidden sm:block">
            <PaginationControls
              pagination={pagination}
              onPageChange={handlePageChange}
              onPageSizeChange={(size) => { setPageSize(size); resetPage(); }}
            />
          </div>
          <div className="sm:hidden">
            <MobilePagination pagination={pagination} onPageChange={handlePageChange} />
          </div>
        </div>
      )}
    </Card>
  );

  // Admin variant rides the standard admin shell (ambient canvas + glass
  // header card) shared with the other redesigned admin list pages.
  if (isAdmin) {
    return (
      <AdminPageCanvas>
        <AdminPageHeaderCard>
          <PageHeader
            icon={Receipt}
            title="Lịch sử trả lương"
            description="Lịch sử trả lương theo nhân viên và kỳ lương."
          />
        </AdminPageHeaderCard>
        {workspace}
      </AdminPageCanvas>
    );
  }

  return (
    <div className="admin-payment-history-page space-y-3 px-3 pb-3 pt-[calc(env(safe-area-inset-top,0px)+0.75rem)] sm:space-y-3.5 sm:px-5 sm:pb-4 sm:pt-[calc(env(safe-area-inset-top,0px)+1rem)] md:px-6 md:pb-6 md:pt-[calc(env(safe-area-inset-top,0px)+1.5rem)]">
      <header className="admin-payment-history-header px-0.5 py-0.5">
        <PageHeader
          className="[&_h1]:text-lg sm:[&_h1]:text-xl [&_[data-slot=page-header-description]]:text-xs"
          icon={Receipt}
          title="Bút toán ngân hàng"
          description="Bút toán ngân hàng theo nhân viên và kỳ lương."
        />
      </header>
      {workspace}
    </div>
  );
}
