import { useMemo, useState } from 'react';
import {
  Banknote,
  CalendarDays,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
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
          className="mt-0 h-11 w-full justify-between border-slate-200 bg-slate-50/70 px-3 font-sans text-[12px] font-normal tabular-nums text-slate-700 shadow-none hover:border-slate-300 hover:bg-slate-50 focus-visible:bg-white"
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
            className="flex h-10 w-10 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <p className="font-financial text-[13px] font-bold tabular-nums text-slate-800">{pickerYear}</p>
          <button
            type="button"
            onClick={() => setPickerYear((year) => year + 1)}
            aria-label="Năm sau"
            className="flex h-10 w-10 items-center justify-center rounded-lg text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600"
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
                className={`h-10 rounded-lg border text-[12px] font-semibold tabular-nums transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-600 ${
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
    <div
      className="divide-y divide-slate-200/80 overflow-hidden rounded-xl border border-slate-200/80 bg-slate-50/40"
      aria-label={`${item.transfers.length} chi tiết thanh toán`}
    >
      {item.transfers.map((transfer) => {
        return (
          <div
            key={`${transfer.transfer_code}-${transfer.bank_reference}-${transfer.amount}`}
            className="group/reference grid min-w-0 grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 transition-colors hover:bg-emerald-50/45"
          >
            <div className="flex min-w-0 flex-wrap items-baseline gap-x-3 gap-y-0.5">
              <p className="whitespace-nowrap text-[10px] text-slate-500">
                <span className="font-medium">Ghi chú:</span>{' '}
                <span className="font-financial text-[11px] font-bold tracking-[-0.02em] text-slate-800 group-hover/reference:text-emerald-900">
                  {transfer.transfer_code || '—'}
                </span>
              </p>
              <p className="whitespace-nowrap text-[10px] text-slate-400">
                <span className="font-medium">Mã GD:</span>{' '}
                <span className="font-financial text-slate-500">{transfer.bank_reference}</span>
              </p>
            </div>
            <span className="whitespace-nowrap font-financial text-[13px] font-bold tabular-nums text-emerald-700">
              {formatCurrency(transfer.amount)}
            </span>
          </div>
        );
      })}
    </div>
  );
}

function HistoryRecord({ item }: { item: BankTransferHistory }) {
  return (
    <article className="group overflow-hidden rounded-2xl border border-slate-200/90 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.03)] transition-all duration-200 hover:-translate-y-0.5 hover:border-emerald-200 hover:shadow-[0_16px_40px_-28px_rgba(6,95,70,0.45)]">
      <div className="grid gap-4 p-3.5 md:grid-cols-[minmax(190px,1fr)_minmax(190px,0.85fr)_minmax(380px,1.65fr)_minmax(145px,0.62fr)] md:items-center md:p-4">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-100">
            <UserRound className="h-4.5 w-4.5" aria-hidden="true" />
          </div>
          <div className="min-w-0 pt-0.5">
            <p className="break-words font-display text-[14px] font-bold leading-snug text-slate-950">
              {item.employee_name}
            </p>
            <p className="mt-1 font-financial text-[10px] font-medium tabular-nums text-slate-500">
              CCCD: {item.employee_cccd || '—'}
            </p>
            {item.project_names.length > 0 && (
              <p className="mt-1 line-clamp-2 text-[11px] leading-relaxed text-slate-500">
                {item.project_names.join(', ')}
              </p>
            )}
          </div>
        </div>

        <div className="border-t border-slate-100 pt-4 md:border-0 md:pt-0">
          <div className="flex items-center gap-2.5">
            <Badge className="shrink-0 rounded-lg border border-emerald-200 bg-emerald-50 px-2.5 py-1 font-display text-[11px] font-bold text-emerald-800 hover:bg-emerald-50">
              Kỳ {item.cycle}
            </Badge>
            <div className="min-w-0">
              <div className="flex items-center gap-1.5 text-[11px] font-medium tabular-nums text-slate-700">
                <CalendarDays className="h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
                <span
                  className="whitespace-nowrap"
                  aria-label={`Từ ${formatBankTransferDate(item.from_date)} đến ${formatBankTransferDate(item.to_date)}`}
                >
                  {formatBankTransferPeriod(item.from_date, item.to_date)}
                </span>
              </div>
              <p className="mt-1 text-[10px] text-slate-500">
                Thanh toán{' '}
                <time dateTime={item.payment_date} className="font-semibold tabular-nums text-emerald-700">
                  {formatBankTransferDate(item.payment_date)}
                </time>
              </p>
            </div>
          </div>
        </div>

        <div className="border-t border-slate-100 pt-4 md:border-0 md:pt-0">
          <div className="mb-2 flex items-center justify-between gap-3 md:hidden">
            <p className="font-display text-[11px] font-bold uppercase tracking-[0.12em] text-slate-500">
              Chi tiết thanh toán
            </p>
            <span className="text-[11px] text-slate-400">{item.transfers.length} giao dịch</span>
          </div>
          <TransferReferences item={item} />
        </div>

        <div className="relative overflow-hidden rounded-xl bg-emerald-950 px-3.5 py-3 text-left text-white md:min-h-[76px] md:text-right">
          <Banknote className="absolute -bottom-4 -left-2 h-14 w-14 rotate-[-8deg] text-white/[0.06]" aria-hidden="true" />
          <p className="relative font-display text-[10px] font-bold uppercase tracking-[0.14em] text-emerald-200">
            Tổng đã chuyển
          </p>
          <p className="relative mt-1.5 break-words font-financial text-[15px] font-bold leading-tight tabular-nums tracking-[-0.04em] text-white">
            {formatCurrency(item.total_amount)}
          </p>
          <p className="relative mt-1 text-[10px] text-emerald-200/80">
            {item.transfers.length} bút toán
          </p>
        </div>
      </div>
    </article>
  );
}

function HistorySkeleton() {
  return (
    <div className="space-y-3" aria-label="Đang tải lịch sử trả lương" aria-busy="true">
      {Array.from({ length: 4 }).map((_, index) => (
        <Skeleton key={index} className="h-40 w-full rounded-2xl md:h-28" />
      ))}
    </div>
  );
}

export function BankTransferHistoryPageContent() {
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
    <div className="min-h-full bg-[radial-gradient(circle_at_top_right,rgba(16,185,129,0.07),transparent_28%),linear-gradient(to_bottom,#f8faf9,#f8fafc_36rem)] px-3 py-4 sm:px-5 md:p-6">
      <div className="mx-auto max-w-[1480px] space-y-5">
        <header className="rounded-2xl border border-slate-200/80 bg-white px-5 py-5 shadow-[0_1px_2px_rgba(15,23,42,0.03)] sm:px-6">
          <PageHeader
            icon={ReceiptText}
            title="Lịch sử trả lương"
            description="Bút toán ngân hàng theo nhân viên và kỳ lương."
          />
        </header>

        <Card className="overflow-hidden rounded-2xl border-slate-200/80 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.03)]">
          <CardContent className="grid gap-4 p-4 sm:p-5 md:grid-cols-[180px_220px_minmax(260px,1fr)]">
            <label className="space-y-2 font-display text-[11px] font-bold text-slate-600">
              Tháng kỳ lương
              <PayrollMonthPicker
                value={month}
                onValueChange={(value) => { setMonth(value); resetPage(); }}
              />
            </label>
            <label className="space-y-2 font-display text-[11px] font-bold text-slate-600">
              Kỳ lương
              <Select value={cycle} onValueChange={(value) => { setCycle(value); resetPage(); }}>
                <SelectTrigger className="mt-0 h-11 border-slate-200 bg-slate-50/70 text-[12px] shadow-none hover:border-slate-300 focus:bg-white">
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
            <label className="space-y-2 font-display text-[11px] font-bold text-slate-600">
              Tìm kiếm
              <div className="relative mt-0">
                <Search className="pointer-events-none absolute left-3 top-3.5 h-4 w-4 text-slate-400" aria-hidden="true" />
                <Input
                  value={search}
                  onChange={(event) => { setSearch(event.target.value); resetPage(); }}
                  placeholder="Tên nhân viên, mã chuyển khoản hoặc mã ngân hàng"
                  className="border-slate-200 bg-slate-50/70 pl-9 pr-11 text-[12px] shadow-none hover:border-slate-300 focus-visible:bg-white"
                />
                {search && (
                  <button
                    type="button"
                    onClick={clearSearch}
                    aria-label="Xóa tìm kiếm"
                    className="absolute right-0 top-0 flex h-11 w-11 items-center justify-center rounded-r-md text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-600"
                  >
                    <X className="h-4 w-4" aria-hidden="true" />
                  </button>
                )}
              </div>
            </label>
          </CardContent>
        </Card>

        <section aria-labelledby="completed-transfers-heading" className="space-y-3">
          <div className="flex min-h-11 flex-wrap items-center justify-between gap-3 px-1">
            <div className="flex items-center gap-2.5">
              <span className="flex h-8 w-8 items-center justify-center rounded-xl bg-emerald-100 text-emerald-700">
                <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
              </span>
              <div>
                <h2 id="completed-transfers-heading" className="font-display text-[14px] font-bold text-slate-950">
                  Giao dịch đã hoàn tất
                </h2>
                <p className="mt-0.5 text-[11px] text-slate-500" aria-live="polite">
                  {pagination ? `${pagination.totalRecords} kết quả theo nhân viên và kỳ lương` : 'Đang tải dữ liệu'}
                </p>
              </div>
            </div>
          </div>

          {!isLoading && !isError && records.length > 0 && (
            <div className="hidden grid-cols-[minmax(190px,1fr)_minmax(190px,0.85fr)_minmax(380px,1.65fr)_minmax(145px,0.62fr)] gap-4 px-4 py-1 md:grid">
              <span className="font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Nhân viên</span>
              <span className="font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Kỳ thanh toán</span>
              <span className="font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Chi tiết thanh toán</span>
              <span className="text-right font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Tổng cộng</span>
            </div>
          )}

          {isLoading && <HistorySkeleton />}

          {isError && (
            <Card className="rounded-2xl border-rose-200 bg-rose-50/80 shadow-none">
              <CardContent className="flex flex-col items-center gap-3 py-12 text-center">
                <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-rose-100 text-rose-700">
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
            <Card className="rounded-2xl border-dashed border-slate-300 bg-white/70 shadow-none">
              <CardContent className="flex flex-col items-center gap-3 py-14 text-center">
                <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-slate-100 text-slate-500">
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
            <div className="space-y-3">
              {records.map((item) => (
                <HistoryRecord key={`${item.work_month}-${item.cycle}-${item.employee_id}`} item={item} />
              ))}
            </div>
          )}
        </section>

        {pagination && pagination.totalPages > 1 && (
          <nav className="flex items-center justify-center gap-3 pt-2" aria-label="Phân trang lịch sử trả lương">
            <Button variant="outline" size="icon" disabled={page <= 1} onClick={() => setPage((value) => value - 1)} aria-label="Trang trước" className="rounded-xl border-slate-200 bg-white shadow-sm">
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="min-w-28 text-center font-display text-[11px] font-semibold text-slate-600">
              Trang <strong className="text-slate-950">{pagination.page}</strong> trên {pagination.totalPages}
            </span>
            <Button variant="outline" size="icon" disabled={page >= pagination.totalPages} onClick={() => setPage((value) => value + 1)} aria-label="Trang sau" className="rounded-xl border-slate-200 bg-white shadow-sm">
              <ChevronRight className="h-4 w-4" />
            </Button>
          </nav>
        )}
      </div>
    </div>
  );
}
