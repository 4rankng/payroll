import { useMemo, useRef, useState, type MouseEvent } from 'react';
import {
  CalendarDays,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Clock3,
  ReceiptText,
  UserRound,
} from 'lucide-react';

import { PageHeader } from '@/components/shared/PageHeader';
import { FilterBar } from '@/components/shared/FilterBar';
import { FilterPill } from '@/components/shared/FilterPill';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
} from '@/components/shared/AdminPageFrame';
import { EmptyState } from '@/components/shared/EmptyState';
import { InlineStatStrip, type InlineStatItem } from '@/components/shared/InlineStatStrip';
import { MobilePagination } from '@/components/shared/MobilePagination';
import { SearchBar } from '@/components/shared/SearchBar';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { PaginationControls } from '@/components/ui/pagination-controls';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Skeleton } from '@/components/ui/skeleton';
import { useBankTransferHistories } from '@/hooks/api/usePayrolls';
import type { BankTransferHistory } from '@/types/api/payroll.types';
import {
  formatBankTransferDate,
  formatBankTransferDateTime,
  formatBankTransferPeriod,
  getCurrentMonthValue,
} from '@/utils/bankTransferHistoryHelpers';
import { formatCurrency } from '@/utils/formatters';

// Shared by the column header and every record summary so the two grids can
// never drift apart.
const RECORD_GRID_COLS = 'xl:grid-cols-[minmax(220px,1.25fr)_minmax(180px,0.9fr)_minmax(150px,0.72fr)_minmax(150px,0.65fr)_36px]';

const CYCLE_OPTIONS = [
  { value: '1', label: 'Kỳ 1 · ngày 1–7' },
  { value: '2', label: 'Kỳ 2 · ngày 8–14' },
  { value: '3', label: 'Kỳ 3 · ngày 15–21' },
  { value: '4', label: 'Kỳ 4 · ngày 22–28' },
];

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
          className={`mt-0 h-11 w-full justify-between rounded-xl border-slate-200 bg-slate-50/70 px-3 font-sans text-[12px] font-normal tabular-nums text-slate-700 shadow-none hover:border-slate-300 hover:bg-slate-50 focus-visible:bg-white sm:h-9 ${className ?? ''}`}
        >
          {displayValue}
          <CalendarDays className="h-4 w-4 text-slate-500" aria-hidden="true" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[280px] rounded-xl p-3" align="start">
        <div className="flex items-center justify-between border-b border-slate-100 pb-2">
          <button
            type="button"
            onClick={() => setPickerYear((year) => year - 1)}
            aria-label="Năm trước"
            className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <p className="font-financial text-[13px] font-bold tabular-nums text-slate-800">{pickerYear}</p>
          <button
            type="button"
            onClick={() => setPickerYear((year) => year + 1)}
            aria-label="Năm sau"
            className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
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
                    : 'border-slate-200 bg-white text-slate-700 hover:border-emerald-300 hover:bg-emerald-50'
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

function TransferReferences({ item }: { item: BankTransferHistory }) {
  return (
    <div className="border-t border-slate-200/80 bg-slate-50/80">
      <div className="hidden grid-cols-[36px_minmax(160px,0.8fr)_minmax(220px,1.1fr)_minmax(180px,0.9fr)_minmax(130px,auto)] gap-4 border-b border-slate-200/80 px-4 py-1.5 xl:grid">
        <span className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500">STT</span>
        <span className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500">Ghi chú chuyển khoản</span>
        <span className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500">Mã giao dịch ngân hàng</span>
        <span className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500">Thời gian xử lý</span>
        <span className="text-right font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500">Số tiền</span>
      </div>

      <div className="divide-y divide-slate-200/70 px-1 pb-1 xl:px-0 xl:pb-0" aria-label={`${item.transfers.length} chi tiết thanh toán`} role="list">
        {item.transfers.map((transfer, index) => (
          <div
            key={`${transfer.transfer_code}-${transfer.bank_reference}-${transfer.amount}`}
            className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] items-start gap-x-3 gap-y-1.5 px-2 py-2.5 xl:min-h-10 xl:grid-cols-[36px_minmax(160px,0.8fr)_minmax(220px,1.1fr)_minmax(180px,0.9fr)_minmax(130px,auto)] xl:items-center xl:gap-4 xl:px-4 xl:py-1.5"
            role="listitem"
          >
            <span className="inline-flex h-6 items-center justify-center rounded-md border border-slate-200 bg-white px-2 font-display text-xs font-bold uppercase tracking-[0.08em] text-slate-500 xl:h-6 xl:w-6 xl:px-0 xl:font-financial xl:text-xs xl:tracking-normal">
              <span className="mr-1 xl:hidden">Giao dịch</span>
              {String(index + 1).padStart(2, '0')}
            </span>

            <div className="min-w-0 text-right xl:col-start-5 xl:row-start-1">
              <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Số tiền</p>
              <p className="mt-0.5 whitespace-nowrap font-financial text-[14px] font-bold tabular-nums text-emerald-700 xl:mt-0 xl:text-[13px]">
                {formatCurrency(transfer.amount)}
              </p>
            </div>

            <div className="col-span-2 min-w-0 xl:col-span-1 xl:col-start-2 xl:row-start-1">
              <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Ghi chú chuyển khoản</p>
              <p className="mt-1 break-all font-financial text-xs font-bold text-slate-800 xl:mt-0">{transfer.transfer_code || '—'}</p>
            </div>

            <div className="col-span-2 min-w-0 sm:col-span-1 xl:col-start-3 xl:row-start-1">
              <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Mã giao dịch ngân hàng</p>
              <p className="mt-1 break-all font-financial text-xs font-semibold text-slate-800 xl:mt-0">{transfer.bank_reference}</p>
            </div>

            <div className="col-span-2 min-w-0 sm:col-span-1 xl:col-start-4 xl:row-start-1">
              <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Thời gian xử lý</p>
              <p className="mt-1 flex items-center gap-1.5 xl:mt-0">
                <Clock3 className="h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" />
                <time className="font-financial text-xs font-semibold tabular-nums text-slate-700" dateTime={transfer.paid_at}>
                  {formatBankTransferDateTime(transfer.paid_at)}
                </time>
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function HistoryRecord({ item, isOpen, onToggle }: { item: BankTransferHistory; isOpen: boolean; onToggle: () => void }) {
  const detailsId = `payment-details-${item.employee_id}-${item.work_month}-${item.cycle}`;
  const detailsRef = useRef<HTMLDetailsElement | null>(null);

  // The accordion state is controlled so only one record can stay open; the
  // native toggle is suppressed to keep React the single source of truth.
  const handleSummaryClick = (event: MouseEvent) => {
    event.preventDefault();
    const willOpen = !isOpen;
    onToggle();
    // Short rows + tall panels push the anchor below the fold on mobile, so
    // snap the freshly opened record to the top of the viewport.
    if (willOpen && typeof window.matchMedia === 'function' && !window.matchMedia('(min-width: 1280px)').matches) {
      detailsRef.current?.scrollIntoView?.({ behavior: 'smooth', block: 'start' });
    }
  };

  return (
    <article>
      <details
        ref={detailsRef}
        open={isOpen}
        className="group/record scroll-mt-3 open:shadow-[inset_4px_0_0_0_#059669]"
      >
        <summary
          aria-controls={detailsId}
          aria-label={`Chi tiết giao dịch của ${item.employee_name}`}
          onClick={handleSummaryClick}
          className={`group/summary relative grid cursor-pointer list-none grid-cols-2 gap-x-3 gap-y-2.5 p-3 outline-none transition-colors marker:content-none hover:bg-slate-50/70 group-open/record:hover:bg-transparent focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-600 xl:min-h-[48px] ${RECORD_GRID_COLS} xl:items-center xl:gap-4 xl:px-4 xl:py-1.5 [&::-webkit-details-marker]:hidden`}
        >
          <div className="col-span-2 flex min-w-0 items-center gap-2.5 pr-12 xl:col-span-1 xl:gap-2 xl:pr-0">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-emerald-200/80 bg-emerald-50 text-emerald-700 xl:h-5 xl:w-5 xl:rounded-md">
              <UserRound className="h-4 w-4 xl:h-3 xl:w-3" aria-hidden="true" />
            </div>
            <div className="min-w-0 xl:flex xl:min-w-0 xl:flex-wrap xl:items-baseline xl:gap-x-1.5">
              <p className="break-words font-display text-[13px] font-bold leading-snug text-slate-950 xl:truncate">
                {item.employee_name}
              </p>
              <span className="hidden text-slate-300 xl:inline" aria-hidden="true">·</span>
              <p className="mt-0.5 font-financial text-xs font-semibold tabular-nums tracking-[0.02em] text-slate-600 xl:mt-0">
                CCCD {item.employee_cccd || '—'}
              </p>
              {item.project_names.length > 0 && (
                <>
                  <span className="hidden text-slate-300 xl:inline" aria-hidden="true">·</span>
                  <p className="mt-0.5 truncate text-xs text-slate-500 xl:mt-0" title={item.project_names.join(', ')}>
                    {item.project_names.join(', ')}
                  </p>
                </>
              )}
            </div>
          </div>

          <div className="order-3 min-w-0 border-l-2 border-slate-100 pl-2.5 xl:order-none xl:border-0 xl:pl-0">
            <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Kỳ thanh toán</p>
            <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs font-semibold tabular-nums text-slate-800 xl:mt-0">
              <span className="flex min-w-0 items-center gap-1.5">
                <CalendarDays className="h-3.5 w-3.5 shrink-0 text-slate-500" aria-hidden="true" />
                <span
                  className="whitespace-nowrap"
                  aria-label={`Từ ${formatBankTransferDate(item.from_date)} đến ${formatBankTransferDate(item.to_date)}`}
                >
                  {formatBankTransferPeriod(item.from_date, item.to_date)}
                </span>
              </span>
              <Badge className="shrink-0 rounded-full border border-slate-200 bg-slate-50 px-1.5 py-0 font-display text-xs font-bold text-slate-600 hover:bg-slate-50">
                Kỳ {item.cycle}
              </Badge>
            </div>
          </div>

          <div className="order-3 min-w-0 border-l-2 border-slate-100 pl-2.5 xl:order-none xl:border-0 xl:pl-0">
            <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Ngày thanh toán</p>
            <time dateTime={item.payment_date} className="mt-1 block font-financial text-xs font-bold tabular-nums text-slate-800 xl:mt-0 xl:text-right">
              {formatBankTransferDate(item.payment_date)}
            </time>
          </div>

          <div className="order-2 col-span-2 flex items-end justify-between border-y border-slate-100 py-2.5 xl:order-none xl:col-span-1 xl:block xl:border-0 xl:py-0 xl:text-right">
            <div>
              <p className="font-display text-xs font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Thực nhận</p>
              <p className="mt-0.5 whitespace-nowrap font-financial text-[20px] font-bold tabular-nums tracking-[-0.03em] text-emerald-700 xl:mt-0 xl:text-[13px] xl:tracking-normal">
                {formatCurrency(item.total_amount)}
              </p>
            </div>
            <p className="pb-0.5 text-right text-xs leading-snug text-slate-500 xl:mt-0.5 xl:pb-0">
              {item.transfers.length} bút toán ngân hàng
            </p>
          </div>

          <div className="absolute right-3 top-3 flex items-center justify-end xl:static">
            <span className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 transition-[color,background-color] group-hover/summary:bg-emerald-50 group-hover/summary:text-emerald-700 xl:h-8 xl:w-8">
              <ChevronDown className="h-4 w-4 transition-transform duration-200 group-open/record:rotate-180 group-open/record:text-emerald-700" aria-hidden="true" />
            </span>
          </div>
        </summary>

        <div id={detailsId} className="border-t border-slate-200/80 group-open/record:border-t-0">
          <TransferReferences item={item} />
        </div>
      </details>
    </article>
  );
}

function HistorySkeleton() {
  return (
    <div className="divide-y divide-slate-200/80" aria-label="Đang tải lịch sử trả lương" aria-busy="true">
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} className="h-36 w-full rounded-none xl:h-16" />
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
  // Controlled accordion key — exactly one record can be expanded at a time.
  const [openRecordKey, setOpenRecordKey] = useState<string | null>(null);

  const filters = useMemo(() => ({
    month,
    cycle: cycle === 'all' ? undefined : Number(cycle),
    search: search.trim() || undefined,
    page,
    pageSize,
  }), [cycle, month, page, pageSize, search]);

  const { data, isLoading, isError, refetch } = useBankTransferHistories(filters);
  const records = data?.data ?? [];
  const pagination = data?.pagination;
  const summary = data?.summary;

  const resetPage = () => {
    setPage(1);
    setOpenRecordKey(null);
  };

  const handlePageChange = (nextPage: number) => {
    setPage(nextPage);
    setOpenRecordKey(null);
  };

  const statItems: InlineStatItem[] = [
    { label: 'Tổng đã chuyển', value: formatCurrency(summary?.total_amount ?? 0), highlight: true },
    { label: 'Bút toán ngân hàng', value: summary?.transfer_count ?? 0 },
    { label: 'Nhân viên', value: summary?.employee_count ?? 0 },
  ];

  // The records workspace (filters + accordion rows + pagination) is shared by
  // both variants; only the page shell and the card treatment differ.
  const workspace = (
    <Card
      data-slot="payment-history-workspace"
      className={
        isAdmin
          ? 'ct-card admin-payment-history-workspace overflow-hidden rounded-2xl border-slate-200/80 bg-white shadow-[0_1px_2px_rgba(16,24,40,0.04),0_20px_56px_-42px_rgba(8,120,62,0.26)] xl:overflow-visible'
          : 'overflow-hidden rounded-xl border-slate-200/80 bg-white shadow-[0_12px_28px_-26px_rgba(15,23,42,0.42)] xl:overflow-visible xl:rounded-none'
      }
    >
        <div className="border-b border-slate-200/80 p-2.5 sm:p-3 xl:py-1.5">
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
            <div
              data-slot="payment-history-header"
              className={`hidden gap-4 border-b border-slate-200/90 bg-slate-50/80 px-4 py-2 xl:sticky xl:top-0 xl:z-10 xl:grid xl:bg-slate-50/95 xl:backdrop-blur ${RECORD_GRID_COLS}`}
            >
              <span className="font-display text-xs font-bold uppercase tracking-[0.11em] text-slate-500">Nhân viên</span>
              <span className="font-display text-xs font-bold uppercase tracking-[0.11em] text-slate-500">Kỳ thanh toán</span>
              <span className="font-display text-xs font-bold uppercase tracking-[0.11em] text-slate-500 xl:text-right">Ngày thanh toán</span>
              <span className="text-right font-display text-xs font-bold uppercase tracking-[0.11em] text-slate-500">Thực nhận</span>
              <span className="sr-only">Chi tiết</span>
            </div>
            <div data-slot="payment-history-records" aria-label="Giao dịch đã hoàn tất" className="divide-y divide-slate-200/80">
              {records.map((item) => {
                const recordKey = `${item.work_month}-${item.cycle}-${item.employee_id}`;
                return (
                  <HistoryRecord
                    key={recordKey}
                    item={item}
                    isOpen={openRecordKey === recordKey}
                    onToggle={() => setOpenRecordKey((current) => (current === recordKey ? null : recordKey))}
                  />
                );
              })}
            </div>
          </>
        )}

        {pagination && pagination.totalPages > 1 && (
          <div className="border-t border-slate-200/80 px-2.5 sm:px-3">
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
            icon={ReceiptText}
            title="Lịch sử trả lương"
            description="Lịch sử trả lương theo nhân viên và kỳ lương."
          />
        </AdminPageHeaderCard>
        <InlineStatStrip items={statItems} isLoading={isLoading} />
        {workspace}
      </AdminPageCanvas>
    );
  }

  return (
    <div className="admin-payment-history-page space-y-3 px-3 pb-3 pt-[calc(env(safe-area-inset-top,0px)+0.75rem)] sm:space-y-3.5 sm:px-5 sm:pb-4 sm:pt-[calc(env(safe-area-inset-top,0px)+1rem)] md:px-6 md:pb-6 md:pt-[calc(env(safe-area-inset-top,0px)+1.5rem)]">
      <header className="admin-payment-history-header px-0.5 py-0.5">
        <PageHeader
          className="[&_h1]:text-lg sm:[&_h1]:text-xl [&_[data-slot=page-header-description]]:text-xs"
          icon={ReceiptText}
          title="Bút toán ngân hàng"
          description="Bút toán ngân hàng theo nhân viên và kỳ lương."
        />
      </header>
      <InlineStatStrip items={statItems} isLoading={isLoading} />
      {workspace}
    </div>
  );
}
