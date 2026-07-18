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
  ShieldCheck,
  UserRound,
  X,
} from 'lucide-react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';
import { useBankTransferHistories } from '@/hooks/api/usePayrolls';
import { useDebounce } from '@/hooks/useDebounce';
import type { BankTransferHistory } from '@/types/api/payroll.types';
import {
  BANK_TRANSFER_CYCLE_LABELS,
  formatBankTransferDate,
  getCurrentMonthValue,
} from '@/utils/bankTransferHistoryHelpers';
import { formatCurrency } from '@/utils/formatters';

function TransferReferences({ item }: { item: BankTransferHistory }) {
  return (
    <div className="space-y-2" aria-label={`${item.transfers.length} bút toán ngân hàng`}>
      {item.transfers.map((transfer) => (
        <div
          key={`${transfer.bank_reference}-${transfer.amount}`}
          className="group/reference flex min-w-0 flex-col gap-1 rounded-lg border border-slate-200/80 bg-white px-3 py-2.5 transition-colors hover:border-emerald-300 sm:flex-row sm:items-center sm:justify-between sm:gap-4"
        >
          <span className="min-w-0 break-all font-financial text-[12px] font-semibold tracking-[-0.02em] text-slate-700 group-hover/reference:text-emerald-900">
            {transfer.bank_reference}
          </span>
          <span className="whitespace-nowrap font-financial text-[13px] font-bold tabular-nums text-emerald-700">
            {formatCurrency(transfer.amount)}
          </span>
        </div>
      ))}
    </div>
  );
}

function HistoryRecord({ item }: { item: BankTransferHistory }) {
  return (
    <article className="group overflow-hidden rounded-2xl border border-slate-200/90 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.03)] transition-all duration-200 hover:-translate-y-0.5 hover:border-emerald-200 hover:shadow-[0_16px_40px_-28px_rgba(6,95,70,0.45)]">
      <div className="grid gap-5 p-4 md:grid-cols-[minmax(170px,1.05fr)_minmax(180px,0.9fr)_minmax(280px,1.5fr)_minmax(145px,0.65fr)] md:items-start md:p-5">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-emerald-700 ring-1 ring-inset ring-emerald-100">
            <UserRound className="h-4.5 w-4.5" aria-hidden="true" />
          </div>
          <div className="min-w-0 pt-0.5">
            <p className="break-words font-display text-[14px] font-bold leading-snug text-slate-950">
              {item.employee_name}
            </p>
            {item.project_names.length > 0 && (
              <p className="mt-1.5 line-clamp-2 text-[11px] leading-relaxed text-slate-500">
                {item.project_names.join(', ')}
              </p>
            )}
          </div>
        </div>

        <div className="border-t border-slate-100 pt-4 md:border-0 md:pt-0">
          <Badge className="rounded-full border border-amber-200 bg-amber-50 px-2.5 py-1 font-display text-[11px] font-bold text-amber-900 hover:bg-amber-50">
            {BANK_TRANSFER_CYCLE_LABELS[item.cycle]}
          </Badge>
          <div className="mt-3 flex items-start gap-2 text-[11px] leading-relaxed text-slate-500">
            <CalendarDays className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-400" aria-hidden="true" />
            <span>
              {formatBankTransferDate(item.from_date)}–{formatBankTransferDate(item.to_date)}
              <span className="block font-medium text-slate-700">
                Trả ngày {formatBankTransferDate(item.payment_date)}
              </span>
            </span>
          </div>
        </div>

        <div className="border-t border-slate-100 pt-4 md:border-0 md:pt-0">
          <div className="mb-2.5 flex items-center justify-between gap-3 md:hidden">
            <p className="font-display text-[11px] font-bold uppercase tracking-[0.12em] text-slate-500">
              Bút toán ngân hàng
            </p>
            <span className="text-[11px] text-slate-400">{item.transfers.length} giao dịch</span>
          </div>
          <TransferReferences item={item} />
        </div>

        <div className="relative overflow-hidden rounded-xl bg-emerald-950 px-4 py-4 text-left text-white md:min-h-[92px] md:text-right">
          <Banknote className="absolute -bottom-3 -left-2 h-16 w-16 rotate-[-8deg] text-white/[0.06]" aria-hidden="true" />
          <p className="relative font-display text-[10px] font-bold uppercase tracking-[0.14em] text-emerald-200">
            Tổng đã chuyển
          </p>
          <p className="relative mt-2 break-words font-financial text-[16px] font-bold leading-tight tabular-nums tracking-[-0.04em] text-white">
            {formatCurrency(item.total_amount)}
          </p>
          <p className="relative mt-1.5 text-[10px] text-emerald-200/80">
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
        <Skeleton key={index} className="h-40 w-full rounded-2xl md:h-32" />
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
        <header className="relative isolate overflow-hidden rounded-[24px] bg-emerald-950 px-5 py-6 text-white shadow-[0_20px_55px_-35px_rgba(6,78,59,0.9)] sm:px-7 sm:py-7 md:px-8">
          <div className="absolute inset-0 -z-10 bg-[radial-gradient(circle_at_85%_0%,rgba(251,191,36,0.18),transparent_28%),linear-gradient(115deg,transparent_45%,rgba(255,255,255,0.04))]" />
          <div className="absolute -right-12 -top-20 -z-10 h-52 w-52 rounded-full border border-white/10" />
          <div className="absolute -right-3 -top-12 -z-10 h-36 w-36 rounded-full border border-amber-300/20" />

          <div className="flex flex-col gap-5 sm:flex-row sm:items-end sm:justify-between">
            <div className="max-w-2xl">
              <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-2xl border border-white/10 bg-white/10 text-amber-300 shadow-inner">
                <ReceiptText className="h-5 w-5" aria-hidden="true" />
              </div>
              <h1 className="font-display text-[24px] font-extrabold leading-tight tracking-[-0.03em] sm:text-[28px]">
                Lịch sử trả lương
              </h1>
              <p className="mt-2 max-w-xl text-[12px] leading-relaxed text-emerald-100/75 sm:text-[13px]">
                Tra cứu các bút toán ngân hàng theo từng nhân viên và kỳ lương tuần.
              </p>
            </div>

            <div className="flex w-fit items-center gap-2 rounded-full border border-emerald-300/20 bg-emerald-900/70 px-3 py-2 text-[11px] font-semibold text-emerald-100 backdrop-blur-sm">
              <ShieldCheck className="h-3.5 w-3.5 text-emerald-300" aria-hidden="true" />
              Chỉ giao dịch đã hoàn tất
            </div>
          </div>
        </header>

        <Card className="relative z-10 -mt-8 overflow-hidden rounded-2xl border-white/80 bg-white/95 shadow-[0_18px_50px_-38px_rgba(15,23,42,0.55)] backdrop-blur-sm sm:mx-4">
          <CardContent className="grid gap-4 p-4 sm:p-5 md:grid-cols-[180px_220px_minmax(260px,1fr)]">
            <label className="space-y-2 font-display text-[11px] font-bold text-slate-600">
              Tháng kỳ lương
              <Input
                type="month"
                value={month}
                onChange={(event) => { setMonth(event.target.value); resetPage(); }}
                className="mt-0 border-slate-200 bg-slate-50/70 font-sans text-[12px] shadow-none hover:border-slate-300 focus-visible:bg-white"
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
                  placeholder="Tên nhân viên hoặc mã bút toán"
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
            <div className="hidden grid-cols-[minmax(170px,1.05fr)_minmax(180px,0.9fr)_minmax(280px,1.5fr)_minmax(145px,0.65fr)] gap-5 px-5 py-1 md:grid">
              <span className="font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Nhân viên</span>
              <span className="font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Kỳ thanh toán</span>
              <span className="font-display text-[10px] font-bold uppercase tracking-[0.12em] text-slate-400">Bút toán ngân hàng</span>
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
