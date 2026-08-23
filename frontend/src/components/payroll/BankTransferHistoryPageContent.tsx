import { useMemo, useState } from 'react';
import {
  CalendarDays,
  CheckCircle2,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Clock3,
  Landmark,
  ReceiptText,
  Search,
  UserRound,
  X,
} from 'lucide-react';

import { PageHeader } from '@/components/shared/PageHeader';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useBankTransferHistories } from '@/hooks/api/usePayrolls';
import { useDebounce } from '@/hooks/useDebounce';
import type { BankTransferHistory } from '@/types/api/payroll.types';
import {
  formatBankTransferDate,
  formatBankTransferDateTime,
  formatBankTransferPeriod,
  getCurrentMonthValue,
} from '@/utils/bankTransferHistoryHelpers';
import { formatCurrency } from '@/utils/formatters';

interface PayrollMonthPickerProps {
  value: string;
  onValueChange: (value: string) => void;
}

function PayrollMonthPicker({ value, onValueChange }: PayrollMonthPickerProps) {
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
          className="mt-0 h-11 w-full justify-between border-slate-200 bg-slate-50/70 px-3 font-sans text-[12px] font-normal tabular-nums text-slate-700 shadow-none hover:border-slate-300 hover:bg-slate-50 focus-visible:bg-white sm:h-8"
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
    <div className="bg-slate-50/90">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200/80 px-3 py-2.5 sm:px-4">
        <div>
          <p className="flex items-center gap-2 font-display text-[11px] font-bold text-slate-900">
            <Landmark className="h-3.5 w-3.5 text-emerald-700" aria-hidden="true" />
            Chi tiết giao dịch ngân hàng
          </p>
          <p className="mt-0.5 text-[10px] text-slate-500">Mã đối soát và thời gian xử lý của từng bút toán</p>
        </div>
        <span className="rounded-full border border-slate-200 bg-white px-2.5 py-1 font-financial text-[10px] font-bold tabular-nums text-slate-600">
          {item.transfers.length} bút toán
        </span>
      </div>

      <div className="hidden grid-cols-[36px_minmax(160px,0.8fr)_minmax(220px,1.1fr)_minmax(180px,0.9fr)_minmax(130px,auto)] gap-4 border-b border-slate-200/80 bg-white/70 px-4 py-2 xl:grid">
        <span className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-400">STT</span>
        <span className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-400">Ghi chú chuyển khoản</span>
        <span className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-400">Mã giao dịch ngân hàng</span>
        <span className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-400">Thời gian xử lý</span>
        <span className="text-right font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-400">Số tiền</span>
      </div>

      <div className="space-y-2 p-2.5 xl:space-y-0 xl:divide-y xl:divide-slate-200/80 xl:p-0" aria-label={`${item.transfers.length} chi tiết thanh toán`} role="list">
        {item.transfers.map((transfer, index) => (
          <div
            key={`${transfer.transfer_code}-${transfer.bank_reference}-${transfer.amount}`}
            className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-2.5 rounded-lg border border-slate-200/90 bg-white p-3 shadow-[0_8px_24px_-24px_rgba(15,23,42,0.4)] xl:min-h-11 xl:grid-cols-[36px_minmax(160px,0.8fr)_minmax(220px,1.1fr)_minmax(180px,0.9fr)_minmax(130px,auto)] xl:items-center xl:gap-4 xl:rounded-none xl:border-0 xl:bg-transparent xl:px-4 xl:py-2.5 xl:shadow-none"
            role="listitem"
          >
            <span className="inline-flex h-7 items-center justify-center rounded-md bg-slate-100 px-2 font-display text-[9px] font-bold uppercase tracking-[0.08em] text-slate-500 xl:h-7 xl:w-7 xl:px-0 xl:font-financial xl:text-[10px] xl:tracking-normal">
              <span className="mr-1 xl:hidden">Giao dịch</span>
              {String(index + 1).padStart(2, '0')}
            </span>

            <div className="min-w-0 text-right xl:col-start-5 xl:row-start-1">
              <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Số tiền</p>
              <p className="mt-0.5 whitespace-nowrap font-financial text-[14px] font-bold tabular-nums text-emerald-700 xl:mt-0 xl:text-[13px]">
                {formatCurrency(transfer.amount)}
              </p>
            </div>

            <div className="col-span-2 min-w-0 border-t border-slate-100 pt-2.5 xl:col-span-1 xl:col-start-2 xl:row-start-1 xl:border-0 xl:pt-0">
              <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Ghi chú chuyển khoản</p>
              <p className="mt-1 break-all font-financial text-[11px] font-bold text-slate-800 xl:mt-0">{transfer.transfer_code || '—'}</p>
            </div>

            <div className="col-span-2 min-w-0 sm:col-span-1 xl:col-start-3 xl:row-start-1">
              <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Mã giao dịch ngân hàng</p>
              <p className="mt-1 break-all font-financial text-[11px] font-semibold text-slate-800 xl:mt-0">{transfer.bank_reference}</p>
            </div>

            <div className="col-span-2 min-w-0 sm:col-span-1 xl:col-start-4 xl:row-start-1">
              <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Thời gian xử lý</p>
              <p className="mt-1 flex items-center gap-1.5 xl:mt-0">
                <Clock3 className="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
                <time className="font-financial text-[11px] font-semibold tabular-nums text-slate-700" dateTime={transfer.paid_at}>
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

function HistoryRecord({ item }: { item: BankTransferHistory }) {
  const detailsId = `payment-details-${item.employee_id}-${item.work_month}-${item.cycle}`;

  return (
    <article>
      <details className="group/record overflow-hidden rounded-xl border border-slate-200/90 bg-white shadow-[0_8px_24px_-24px_rgba(15,23,42,0.42)] transition-[border-color,background-color,box-shadow] hover:border-slate-300 open:border-emerald-300 open:bg-emerald-50/80 open:shadow-[inset_4px_0_0_0_#059669,0_16px_40px_-24px_rgba(5,150,105,0.5)] xl:rounded-none xl:border-0 xl:shadow-none xl:open:shadow-[inset_4px_0_0_0_#059669]">
        <summary
          aria-controls={detailsId}
          aria-label={`Chi tiết giao dịch của ${item.employee_name}`}
          className="group/summary relative grid cursor-pointer list-none grid-cols-2 gap-x-3 gap-y-2.5 p-3 outline-none transition-colors marker:content-none hover:bg-slate-50/70 group-open/record:bg-emerald-100/40 group-open/record:hover:bg-transparent focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-600 xl:min-h-[64px] xl:grid-cols-[minmax(220px,1.25fr)_minmax(180px,0.9fr)_minmax(150px,0.72fr)_minmax(150px,0.65fr)_36px] xl:items-center xl:gap-4 xl:px-4 xl:py-2 [&::-webkit-details-marker]:hidden"
        >
          <div className="col-span-2 flex min-w-0 items-center gap-2.5 pr-12 xl:col-span-1 xl:pr-0">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-emerald-200/80 bg-emerald-50 text-emerald-700 xl:h-8 xl:w-8">
              <UserRound className="h-4 w-4" aria-hidden="true" />
            </div>
            <div className="min-w-0">
              <p className="break-words font-display text-[13px] font-bold leading-snug text-slate-950 xl:truncate">
                {item.employee_name}
              </p>
              <p className="mt-0.5 font-financial text-[10px] font-semibold tabular-nums tracking-[0.02em] text-slate-600">
                CCCD {item.employee_cccd || '—'}
              </p>
              {item.project_names.length > 0 && (
                <p className="mt-0.5 truncate text-[10px] text-slate-500" title={item.project_names.join(', ')}>
                  {item.project_names.join(', ')}
                </p>
              )}
            </div>
          </div>

          <div className="order-3 min-w-0 border-l-2 border-slate-100 pl-2.5 xl:order-none xl:border-0 xl:pl-0">
            <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Kỳ thanh toán</p>
            <div className="mt-1 flex flex-wrap items-center gap-1.5 text-[11px] font-semibold tabular-nums text-slate-800 xl:mt-0">
              <span className="flex min-w-0 items-center gap-1.5">
                <CalendarDays className="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
                <span
                  className="whitespace-nowrap"
                  aria-label={`Từ ${formatBankTransferDate(item.from_date)} đến ${formatBankTransferDate(item.to_date)}`}
                >
                  {formatBankTransferPeriod(item.from_date, item.to_date)}
                </span>
              </span>
              <Badge className="shrink-0 rounded-full border border-slate-200 bg-slate-50 px-1.5 py-0 font-display text-[9px] font-bold text-slate-600 hover:bg-slate-50">
                Kỳ {item.cycle}
              </Badge>
            </div>
          </div>

          <div className="order-3 min-w-0 border-l-2 border-slate-100 pl-2.5 xl:order-none xl:border-0 xl:pl-0">
            <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Ngày thanh toán</p>
            <div className="mt-1 flex flex-wrap items-center gap-1.5 xl:mt-0">
              <time dateTime={item.payment_date} className="font-financial text-[11px] font-bold tabular-nums text-slate-800">
                {formatBankTransferDate(item.payment_date)}
              </time>
              <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-1.5 py-0.5 font-display text-[9px] font-bold text-emerald-700">
                <CheckCircle2 className="h-3 w-3" aria-hidden="true" />
                Đã chuyển
              </span>
            </div>
          </div>

          <div className="order-2 col-span-2 flex items-end justify-between border-y border-slate-100 py-2.5 xl:order-none xl:col-span-1 xl:block xl:border-0 xl:py-0 xl:text-right">
            <div>
              <p className="font-display text-[9px] font-bold uppercase tracking-[0.1em] text-slate-500 xl:hidden">Thực nhận</p>
              <p className="mt-0.5 whitespace-nowrap font-financial text-[20px] font-bold tabular-nums tracking-[-0.03em] text-emerald-700 xl:mt-0 xl:text-[14px] xl:tracking-normal">
                {formatCurrency(item.total_amount)}
              </p>
            </div>
            <p className="pb-0.5 text-right text-[10px] leading-snug text-slate-500 xl:mt-0.5 xl:pb-0">
              {item.transfers.length} bút toán ngân hàng
            </p>
          </div>

          <div className="absolute right-3 top-3 flex items-center justify-end xl:static">
            <span className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-400 transition-[color,background-color] group-hover/summary:bg-emerald-50 group-hover/summary:text-emerald-700 xl:h-8 xl:w-8">
              <ChevronDown className="h-4 w-4 transition-transform duration-200 group-open/record:rotate-180 group-open/record:text-emerald-600" aria-hidden="true" />
            </span>
          </div>
        </summary>

        <div id={detailsId} className="border-t border-slate-200/80 group-open/record:border-emerald-300">
          <TransferReferences item={item} />
        </div>
      </details>
    </article>
  );
}

function HistorySkeleton() {
  return (
    <div className="space-y-2 xl:space-y-px" aria-label="Đang tải lịch sử trả lương" aria-busy="true">
      {Array.from({ length: 6 }).map((_, index) => (
        <Skeleton key={index} className="h-36 w-full rounded-xl xl:h-16 xl:rounded-none" />
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
  const debouncedSearch = useDebounce(search, 300);

  const filters = useMemo(() => ({
    month,
    cycle: cycle === 'all' ? undefined : Number(cycle),
    search: debouncedSearch.trim() || undefined,
    page,
    pageSize: 20,
  }), [cycle, debouncedSearch, month, page]);

  const { data, isLoading, isError, refetch } = useBankTransferHistories(filters);
  const records = data?.data ?? [];
  const pagination = data?.pagination;

  const resetPage = () => setPage(1);
  const clearSearch = () => {
    setSearch('');
    resetPage();
  };

  return (
    <div className="admin-payment-history-page min-h-full bg-[linear-gradient(to_bottom,#f8faf9,#f8fafc_28rem)] px-3 pb-3 pt-[calc(env(safe-area-inset-top,0px)+0.75rem)] sm:px-5 sm:pb-4 sm:pt-[calc(env(safe-area-inset-top,0px)+1rem)] md:px-6 md:pb-6 md:pt-[calc(env(safe-area-inset-top,0px)+1.5rem)]">
      <div className="mx-auto max-w-[1600px] space-y-3 sm:space-y-3.5">
        <header className="admin-payment-history-header px-0.5 py-0.5">
          <PageHeader
            className="[&_h1]:text-lg sm:[&_h1]:text-xl [&_[data-slot=page-header-description]]:text-xs"
            icon={ReceiptText}
            title={isAdmin ? "Lịch sử trả lương" : "Bút toán ngân hàng"}
            description={isAdmin ? "Lịch sử trả lương theo nhân viên và kỳ lương." : "Bút toán ngân hàng theo nhân viên và kỳ lương."}
          />
        </header>

        <Card className={`${isAdmin ? 'ct-card admin-payment-history-filters ' : ''}overflow-hidden rounded-xl border-slate-200/80 bg-white shadow-[0_10px_24px_-24px_rgba(15,23,42,0.38)]`}>
          <CardContent className="grid grid-cols-2 gap-2.5 p-2.5 sm:gap-3 sm:p-3 lg:grid-cols-[160px_200px_minmax(260px,1fr)]">
            <label className="space-y-1.5 font-display text-[10px] font-bold text-slate-600">
              Tháng kỳ lương
              <PayrollMonthPicker
                value={month}
                onValueChange={(value) => { setMonth(value); resetPage(); }}
              />
            </label>
            <label className="space-y-1.5 font-display text-[10px] font-bold text-slate-600">
              Kỳ lương
              <Select value={cycle} onValueChange={(value) => { setCycle(value); resetPage(); }}>
                <SelectTrigger className="mt-0 h-11 border-slate-200 bg-slate-50/70 text-[12px] shadow-none hover:border-slate-300 focus:bg-white sm:h-8">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả kỳ lương</SelectItem>
                  <SelectItem value="1">Kỳ 1 · ngày 1–7</SelectItem>
                  <SelectItem value="2">Kỳ 2 · ngày 8–14</SelectItem>
                  <SelectItem value="3">Kỳ 3 · ngày 15–21</SelectItem>
                  <SelectItem value="4">Kỳ 4 · ngày 22–28</SelectItem>
                </SelectContent>
              </Select>
            </label>
            <label className="col-span-2 space-y-1.5 font-display text-[10px] font-bold text-slate-600 lg:col-span-1">
              Tìm kiếm
              <div className="relative mt-0">
                <Search className="pointer-events-none absolute left-3 top-3.5 h-4 w-4 text-slate-400 sm:top-2" aria-hidden="true" />
                <Input
                  value={search}
                  onChange={(event) => { setSearch(event.target.value); resetPage(); }}
                  placeholder="Tên nhân viên, mã chuyển khoản hoặc mã ngân hàng"
                  className="h-11 border-slate-200 bg-slate-50/70 pl-9 pr-11 text-[12px] shadow-none hover:border-slate-300 focus-visible:bg-white sm:h-8 sm:pl-9 sm:pr-9"
                />
                {search && (
                  <button
                    type="button"
                    onClick={clearSearch}
                    aria-label="Xóa tìm kiếm"
                    className="absolute right-0 top-0 flex h-11 w-11 items-center justify-center rounded-r-md text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-600 sm:h-8 sm:w-8"
                  >
                    <X className="h-4 w-4" aria-hidden="true" />
                  </button>
                )}
              </div>
            </label>
          </CardContent>
        </Card>

        <section aria-labelledby="completed-transfers-heading" className="space-y-2">
          <div className="flex min-h-9 flex-wrap items-center justify-between gap-2 px-0.5">
            <div className="flex min-w-0 items-center gap-2">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-emerald-100 text-emerald-700">
                <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
              </span>
              <h2 id="completed-transfers-heading" className="font-display text-[13px] font-bold text-slate-950">
                Giao dịch đã hoàn tất
              </h2>
              <p className="whitespace-nowrap text-[10px] text-slate-500" aria-live="polite">
                {pagination ? `· ${pagination.totalRecords} kết quả` : '· Đang tải'}
              </p>
            </div>
          </div>

          {isLoading && <HistorySkeleton />}

          {isError && (
            <Card className="rounded-xl border-rose-200 bg-rose-50/80 shadow-none">
              <CardContent className="flex flex-col items-center gap-2.5 py-8 text-center">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100 text-rose-700">
                  <Landmark className="h-5 w-5" aria-hidden="true" />
                </div>
                <div>
                  <p className="font-display text-[14px] font-bold text-rose-900">Không thể tải lịch sử trả lương</p>
                  <p className="mt-1 text-[11px] text-rose-700/75">Vui lòng kiểm tra kết nối và thử lại.</p>
                </div>
                <Button variant="outline" onClick={() => refetch()} className="border-rose-200 bg-white text-rose-800 hover:bg-rose-100">
                  Thử lại
                </Button>
              </CardContent>
            </Card>
          )}

          {!isLoading && !isError && records.length === 0 && (
            <Card className="rounded-xl border-dashed border-slate-300 bg-white/70 shadow-none">
              <CardContent className="flex flex-col items-center gap-2.5 py-8 text-center">
                <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-slate-100 text-slate-500">
                  <Landmark className="h-5 w-5" aria-hidden="true" />
                </div>
                <div>
                  <p className="font-display text-[14px] font-bold text-slate-900">Chưa có giao dịch đã hoàn tất</p>
                  <p className="mt-1.5 text-[11px] text-slate-500">Chọn tháng hoặc kỳ lương khác để xem lịch sử.</p>
                </div>
              </CardContent>
            </Card>
          )}

          {!isLoading && !isError && records.length > 0 && (
            <div
              data-slot="payment-history-workspace"
              className={`${isAdmin ? 'ct-card admin-payment-history-workspace ' : ''}xl:overflow-hidden xl:rounded-xl xl:border xl:border-slate-200/90 xl:bg-white xl:shadow-[0_12px_28px_-26px_rgba(15,23,42,0.42)]`}
            >
              <div className="hidden grid-cols-[minmax(220px,1.25fr)_minmax(180px,0.9fr)_minmax(150px,0.72fr)_minmax(150px,0.65fr)_36px] gap-4 border-b border-slate-200/90 bg-slate-50/80 px-4 py-2 xl:grid">
                <span className="font-display text-[9px] font-bold uppercase tracking-[0.11em] text-slate-500">Nhân viên</span>
                <span className="font-display text-[9px] font-bold uppercase tracking-[0.11em] text-slate-500">Kỳ thanh toán</span>
                <span className="font-display text-[9px] font-bold uppercase tracking-[0.11em] text-slate-500">Ngày thanh toán</span>
                <span className="text-right font-display text-[9px] font-bold uppercase tracking-[0.11em] text-slate-500">Thực nhận</span>
                <span className="sr-only">Chi tiết</span>
              </div>
              <div data-slot="payment-history-records" className="space-y-2 xl:space-y-0 xl:divide-y xl:divide-slate-200/80">
                {records.map((item) => (
                  <HistoryRecord key={`${item.work_month}-${item.cycle}-${item.employee_id}`} item={item} />
                ))}
              </div>
            </div>
          )}
        </section>

        {pagination && pagination.totalPages > 1 && (
          <nav className="flex items-center justify-center gap-2 pt-1" aria-label="Phân trang lịch sử trả lương">
            <Button variant="outline" size="icon" disabled={page <= 1} onClick={() => setPage((value) => value - 1)} aria-label="Trang trước" className="rounded-lg border-slate-200 bg-white shadow-sm sm:h-8 sm:w-8">
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="min-w-28 text-center font-display text-[11px] font-semibold text-slate-600">
              Trang <strong className="text-slate-950">{pagination.page}</strong> trên {pagination.totalPages}
            </span>
            <Button variant="outline" size="icon" disabled={page >= pagination.totalPages} onClick={() => setPage((value) => value + 1)} aria-label="Trang sau" className="rounded-lg border-slate-200 bg-white shadow-sm sm:h-8 sm:w-8">
              <ChevronRight className="h-4 w-4" />
            </Button>
          </nav>
        )}
      </div>
    </div>
  );
}
