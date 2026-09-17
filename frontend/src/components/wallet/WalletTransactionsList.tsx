// Unified wallet transactions list — filters, pagination, click-to-inquire,
// mobile-card fallback, skeleton + empty states.

import { useMemo, useState } from "react";
import {
  RefreshCw,
  ArrowDownLeft,
  ArrowUpRight,
  ChevronRight,
  Copy,
  Check,
  X,
  Loader2,
  AlertCircle,
  CheckCircle2,
  Clock,
  Ban,
  SlidersHorizontal,
} from "lucide-react";
import { useQuery } from "@tanstack/react-query";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetClose,
} from "@/components/ui/sheet";
import { DateRangePicker } from "@/components/ui/date-range-picker";
import { FilterPill } from "@/components/shared/FilterPill";
import { useIsMobile } from '@/hooks/useBreakpoint';
import { cn } from "@/lib/utils";
import { EmptyState as SharedEmptyState } from "@/components/shared/EmptyState";
import { walletService } from "@/services/api/wallet.service";
import { formatDateTime, formatCurrency as formatVND } from "@/utils/formatters";
import type {
  UnifiedTransaction,
  UnifiedTransactionFilter,
  WalletPayment,
  WalletTopup,
} from "@/types/api/wallet.types";

interface WalletTransactionsListProps {}

type StatusFilter = "all" | "pending" | "verified" | "authorised" | "completed" | "failed" | "reversed";
type TypeFilter = "all" | "topup" | "payment";

const PAGE_SIZE = 20;

const STATUS_LABELS: Record<string, string> = {
  pending: "Đang chờ",
  verified: "Đã xác minh",
  authorised: "Đã duyệt",
  completed: "Hoàn thành",
  failed: "Thất bại",
  reversed: "Đã hoàn tiền",
};

const STATUS_BADGE_CLASS: Record<string, string> = {
  pending: "bg-amber-100 text-amber-800 hover:bg-amber-100",
  verified: "bg-cyan-100 text-cyan-800 hover:bg-cyan-100",
  authorised: "bg-blue-100 text-blue-800 hover:bg-blue-100",
  completed: "bg-emerald-100 text-emerald-800 hover:bg-emerald-100",
  failed: "bg-rose-100 text-rose-800 hover:bg-rose-100",
  reversed: "bg-teal-100 text-teal-800 hover:bg-teal-100",
};

const STATUS_ICON: Record<string, React.ReactNode> = {
  pending:    <Clock className="h-3.5 w-3.5" />,
  verified:   <CheckCircle2 className="h-3.5 w-3.5" />,
  authorised: <CheckCircle2 className="h-3.5 w-3.5" />,
  completed:  <CheckCircle2 className="h-3.5 w-3.5" />,
  failed:     <AlertCircle className="h-3.5 w-3.5" />,
  reversed:   <AlertCircle className="h-3.5 w-3.5" />,
};

function parseCounterparty(raw: string): { name: string; detail: string | null } {
  const sep = raw.indexOf(" — ");
  if (sep === -1) return { name: raw, detail: null };
  return { name: raw.slice(0, sep), detail: raw.slice(sep + 3) };
}

function formatAmount(amount: number): string {
  const formatted = formatVND(Math.abs(amount));
  if (amount > 0) return `+${formatted}`;
  return `−${formatted}`;
}

// Provider success codes — "00" (OnePay), "000"/"0" (9Pay) — must not be shown as errors.
const PROVIDER_SUCCESS_CODES = new Set(["00", "0", "000", ""]);

function isRealErrorCode(code: string | null | undefined): boolean {
  if (!code) return false;
  return !PROVIDER_SUCCESS_CODES.has(code.trim());
}

// ── Copy-to-clipboard mini hook ──────────────────────────────────────────────

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = (e: React.MouseEvent) => {
    e.stopPropagation();
    navigator.clipboard.writeText(value).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    });
  };
  return (
    <button
      type="button"
      onClick={handleCopy}
      className="ml-1 inline-flex h-11 w-11 sm:h-6 sm:w-6 shrink-0 items-center justify-center rounded text-slate-500 hover:text-slate-700 hover:bg-slate-100 transition-colors"
      title="Sao chép"
    >
      {copied ? <Check className="h-3 w-3 text-emerald-500" /> : <Copy className="h-3 w-3" />}
    </button>
  );
}

// ── Status badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status, size = "sm" }: { status: string; size?: "sm" | "lg" }) {
  const cls = STATUS_BADGE_CLASS[status] ?? "bg-slate-100 text-slate-700";
  const label = STATUS_LABELS[status] ?? status;
  const icon = STATUS_ICON[status];
  if (size === "lg") {
    return (
      <span className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-semibold ${cls}`}>
        {icon}
        {label}
      </span>
    );
  }
  return (
    <Badge variant="secondary" className={`${cls} font-semibold border-transparent tracking-wide`}>
      {label}
    </Badge>
  );
}

// ── Type label ───────────────────────────────────────────────────────────────

function TypeLabel({ type }: { type: UnifiedTransaction["type"] }) {
  if (type === "topup") {
    return (
      <span className="inline-flex items-center gap-1.5">
        <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-50 ring-1 ring-emerald-100">
          <ArrowDownLeft className="h-3.5 w-3.5 text-emerald-600" />
        </span>
        <span className="text-xs text-emerald-700 font-medium whitespace-nowrap">Nạp tiền</span>
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-slate-100 ring-1 ring-slate-200">
        <ArrowUpRight className="h-3.5 w-3.5 text-slate-500" />
      </span>
      <span className="text-xs text-slate-500 whitespace-nowrap">Chi trả</span>
    </span>
  );
}

// ── Detail field row ─────────────────────────────────────────────────────────

function DetailRow({
  label,
  value,
  mono,
  copyable,
  className,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  copyable?: string;
  className?: string;
}) {
  return (
    <div className={`flex flex-col gap-0.5 py-2.5 ${className ?? ""}`}>
      <span className="text-xs font-medium uppercase tracking-wider text-slate-500">{label}</span>
      <div className={`flex items-center gap-1 text-sm text-slate-800 ${mono ? "font-mono text-xs" : ""}`}>
        {typeof value === "string" && mono ? (
          <span className="break-all">{value}</span>
        ) : (
          value
        )}
        {copyable && <CopyButton value={copyable} />}
      </div>
    </div>
  );
}

// ── Transaction detail sheet ─────────────────────────────────────────────────

function TransactionDetailSheet({
  tx,
  onClose,
}: {
  tx: UnifiedTransaction | null;
  onClose: () => void;
}) {
  const isOpen = tx !== null;

  // Fetch detailed payment data
  const { data: payment, isLoading: loadingPayment } = useQuery<WalletPayment>({
    queryKey: ["wallet", "payment", tx?.id],
    queryFn: () => walletService.getPaymentById(tx!.id),
    enabled: isOpen && tx?.type === "payment",
    staleTime: 30_000,
  });

  // Fetch detailed topup data
  const { data: topup, isLoading: loadingTopup } = useQuery<WalletTopup>({
    queryKey: ["wallet", "topup", tx?.id],
    queryFn: () => walletService.getTopupById(tx!.id),
    enabled: isOpen && tx?.type === "topup",
    staleTime: 30_000,
  });

  const isLoading = tx?.type === "payment" ? loadingPayment : loadingTopup;
  const isInflow = (tx?.amount ?? 0) > 0;

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="right"
        className="w-full sm:max-w-[440px] p-0 flex flex-col"
        title="Chi tiết giao dịch"
        description="Thông tin đầy đủ của giao dịch này"
      >
        {/* Header */}
        <div className={`px-6 pt-6 pb-5 ${isInflow ? "bg-emerald-50/60" : "bg-slate-50/60"}`}>
          <div className="flex items-start justify-between mb-4">
            <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${isInflow ? "bg-emerald-100" : "bg-slate-200"}`}>
              {isInflow
                ? <ArrowDownLeft className="h-5 w-5 text-emerald-600" />
                : <ArrowUpRight className="h-5 w-5 text-slate-600" />
              }
            </div>
            <SheetClose aria-label="Đóng chi tiết giao dịch" className="flex h-11 w-11 items-center justify-center rounded-lg text-slate-500 hover:text-slate-700 hover:bg-white/70 transition-colors">
              <X className="h-4 w-4" />
            </SheetClose>
          </div>

          <p className="text-xs font-medium uppercase tracking-wider text-slate-500 mb-1">
            {tx?.type === "topup" ? "Nạp tiền" : "Chi trả"}
          </p>
          <p className={`text-3xl font-bold tabular-nums leading-none mb-3 ${isInflow ? "text-emerald-700" : "text-slate-900"}`}>
            {tx ? formatAmount(tx.amount) : "—"}
          </p>

          {tx && <StatusBadge status={tx.status} size="lg" />}
        </div>

        <Separator />

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-6 py-2">
          {isLoading ? (
            <div className="flex flex-col gap-4 py-6">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="flex flex-col gap-1.5">
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-4 w-48" />
                </div>
              ))}
            </div>
          ) : tx?.type === "payment" && payment ? (
            <PaymentDetail payment={payment} tx={tx} />
          ) : tx?.type === "topup" && topup ? (
            <TopupDetail topup={topup} tx={tx} />
          ) : (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="h-5 w-5 animate-spin text-slate-500" />
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

function PaymentDetail({ payment, tx }: { payment: WalletPayment; tx: UnifiedTransaction }) {
  return (
    <div className="divide-y divide-slate-100">
      <DetailRow label="Ngày giao dịch" value={formatDateTime(tx.occurred_at)} />
      <DetailRow
        label="Người nhận"
        value={
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-semibold text-slate-800">{payment.recipient_name}</span>
            <span className="text-xs text-slate-500">
              {payment.recipient_account_no} · {payment.recipient_bank}
            </span>
          </div>
        }
      />
      <DetailRow
        label="Số tiền yêu cầu"
        value={<span className="font-semibold">{formatVND(payment.requested_amount)}</span>}
      />
      {payment.charged_amount != null && payment.charged_amount !== payment.requested_amount && (
        <DetailRow
          label="Số tiền thực thu"
          value={<span className="font-semibold">{formatVND(payment.charged_amount)}</span>}
        />
      )}
      {payment.fee != null && payment.fee > 0 && (
        <DetailRow label="Phí giao dịch" value={formatVND(payment.fee)} />
      )}
      {payment.txn_id && (
        <DetailRow
          label="Mã giao dịch (TXN)"
          value={payment.txn_id}
          mono
          copyable={payment.txn_id}
        />
      )}
      {payment.request_id && (
        <DetailRow
          label="Mã yêu cầu"
          value={payment.request_id}
          mono
          copyable={payment.request_id}
        />
      )}
      {payment.invoice_no && (
        <DetailRow
          label="Số hóa đơn"
          value={payment.invoice_no}
          mono
          copyable={payment.invoice_no}
        />
      )}
      {payment.error_code && isRealErrorCode(payment.error_code) && (
        <DetailRow
          label="Lỗi"
          value={
            <span className="inline-flex items-center gap-1.5 text-rose-600">
              <Ban className="h-3.5 w-3.5 shrink-0" />
              {payment.error_message || "Giao dịch thất bại"}
            </span>
          }
        />
      )}
      {payment.error_code && !isRealErrorCode(payment.error_code) && payment.error_message && (
        <DetailRow
          label="Mã phản hồi"
          value={
            <span className="inline-flex items-center gap-1.5 text-emerald-600">
              <CheckCircle2 className="h-3.5 w-3.5 shrink-0" />
              {payment.error_code} — {payment.error_message}
            </span>
          }
        />
      )}
      <DetailRow label="Tạo lúc" value={formatDateTime(payment.created_at)} />
      {payment.settled_at && (
        <DetailRow label="Thanh toán lúc" value={formatDateTime(payment.settled_at)} />
      )}
    </div>
  );
}

function TopupDetail({ topup, tx }: { topup: WalletTopup; tx: UnifiedTransaction }) {
  return (
    <div className="divide-y divide-slate-100">
      <DetailRow label="Ngày giao dịch" value={formatDateTime(tx.occurred_at)} />
      <DetailRow
        label="Mã tham chiếu ngân hàng"
        value={topup.bank_ref}
        mono
        copyable={topup.bank_ref}
      />
      {topup.note && (
        <DetailRow label="Ghi chú" value={topup.note} />
      )}
      <DetailRow label="Tạo lúc" value={formatDateTime(topup.created_at)} />
      <DetailRow label="Cập nhật lúc" value={formatDateTime(topup.updated_at)} />
    </div>
  );
}

// ── Main component ────────────────────────────────────────────────────────────

export default function WalletTransactionsList(
  _props: WalletTransactionsListProps,
) {
  const isMobile = useIsMobile();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all");
  const [typeFilter, setTypeFilter] = useState<TypeFilter>("all");
  const [startDate, setStartDate] = useState<string>("");
  const [endDate, setEndDate] = useState<string>("");
  const [selectedTx, setSelectedTx] = useState<UnifiedTransaction | null>(null);

  const filter: UnifiedTransactionFilter = useMemo(() => {
    const f: UnifiedTransactionFilter = { page, page_size: PAGE_SIZE };
    if (statusFilter !== "all") f.status = statusFilter;
    if (typeFilter !== "all") f.type = typeFilter;
    if (startDate) f.start_date = startDate;
    if (endDate) f.end_date = endDate;
    return f;
  }, [page, statusFilter, typeFilter, startDate, endDate]);

  const { data, isLoading, isFetching, refetch, isError } = useQuery({
    queryKey: ["wallet", "transactions", filter],
    queryFn: () => walletService.getTransactions(filter),
    placeholderData: (prev) => prev,
    staleTime: 15_000,
  });

  const transactions = data?.data ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const hasFilters = statusFilter !== "all" || typeFilter !== "all" || !!startDate || !!endDate;

  const onResetFilters = () => {
    setStatusFilter("all");
    setTypeFilter("all");
    setStartDate("");
    setEndDate("");
    setPage(1);
  };

  const content = (
    <div className={isMobile ? "flex flex-col gap-4" : "flex flex-col gap-3 p-4 md:p-5"}>
      {/* Header */}
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h2 id="wallet-transaction-history-title" className="text-[15px] font-semibold text-slate-800 tracking-tight">
            Lịch sử giao dịch
          </h2>
          {total > 0 && (
            <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-slate-100 px-1.5 text-xs font-bold text-slate-600 tabular-nums tracking-wide">
              {total.toLocaleString("vi-VN")}
            </span>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => refetch()}
          disabled={isFetching}
          className={cn(
            "px-2.5 text-xs font-medium text-slate-500 hover:text-slate-800 tracking-wide",
            isMobile ? "h-11 min-h-11" : "h-8",
          )}
        >
          <RefreshCw className={`h-3.5 w-3.5 mr-1.5 ${isFetching ? "animate-spin" : ""}`} />
          Làm mới
        </Button>
      </div>

      {/* Filters — single compact row for both desktop and mobile */}
      <div className="flex items-center gap-2 flex-wrap">
        <SlidersHorizontal className="h-3.5 w-3.5 text-slate-500 shrink-0" />
        {!isMobile && (
          <DateRangePicker
            variant="default"
            startDate={startDate}
            endDate={endDate}
            onStartDateChange={(d) => { setStartDate(d); setPage(1); }}
            onEndDateChange={(d) => { setEndDate(d); setPage(1); }}
          />
        )}
        <FilterPill
          value={typeFilter}
          onChange={(v) => { setTypeFilter(v as TypeFilter); setPage(1); }}
          placeholder="Loại"
          options={[
            { value: "topup", label: "Nạp tiền" },
            { value: "payment", label: "Chi trả" },
          ]}
        />
        <FilterPill
          value={statusFilter}
          onChange={(v) => { setStatusFilter(v as StatusFilter); setPage(1); }}
          placeholder="Trạng thái"
          options={[
            { value: "pending",    label: "Đang chờ" },
            { value: "authorised", label: "Đã duyệt" },
            { value: "completed",  label: "Hoàn thành" },
            { value: "failed",     label: "Thất bại" },
          ]}
        />
        {isMobile && (
          <DateRangePicker
            variant="mobile"
            startDate={startDate}
            endDate={endDate}
            onStartDateChange={(d) => { setStartDate(d); setPage(1); }}
            onEndDateChange={(d) => { setEndDate(d); setPage(1); }}
          />
        )}
        {hasFilters && (
          <button
            type="button"
            onClick={onResetFilters}
            className={cn(
              "text-xs text-slate-500 hover:text-slate-700 transition-colors ml-auto",
              isMobile && "min-h-11 px-2",
            )}
          >
            Xoá lọc
          </button>
        )}
      </div>

      {/* Body */}
      {isLoading ? (
        isMobile ? <SkeletonCards /> : <SkeletonTable />
      ) : isError ? (
        <ErrorState onRetry={() => refetch()} />
      ) : transactions.length === 0 ? (
        <EmptyState hasFilters={hasFilters} onResetFilters={onResetFilters} />
      ) : isMobile ? (
        <MobileTransactionList transactions={transactions} onSelect={setSelectedTx} />
      ) : (
        <DesktopTable transactions={transactions} onSelect={setSelectedTx} />
      )}

      {/* Pagination */}
      {transactions.length > 0 && totalPages > 1 && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-t pt-4">
          <p className="text-sm text-slate-500">
            {Math.min((page - 1) * PAGE_SIZE + 1, total)}–{Math.min(page * PAGE_SIZE, total)} / {total} giao dịch
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className={isMobile ? "min-h-11" : undefined}
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1 || isFetching}
            >
              Trước
            </Button>
            <span className="text-sm text-slate-500 tabular-nums">
              {page} / {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              className={isMobile ? "min-h-11" : undefined}
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages || isFetching}
            >
              Sau
            </Button>
          </div>
        </div>
      )}
    </div>
  );

  return (
    <>
      {isMobile ? (
        <section aria-labelledby="wallet-transaction-history-title">
          {content}
        </section>
      ) : (
        <Card className="border-border/40 shadow-sm">
          {content}
        </Card>
      )}

      {/* Transaction detail sheet */}
      <TransactionDetailSheet tx={selectedTx} onClose={() => setSelectedTx(null)} />
    </>
  );
}

// ── Desktop table ─────────────────────────────────────────────────────────────

function DesktopTable({
  transactions,
  onSelect,
}: {
  transactions: UnifiedTransaction[];
  onSelect: (tx: UnifiedTransaction) => void;
}) {
  return (
    <div className="rounded-xl border border-slate-200 overflow-x-auto">
      <Table className="min-w-[820px]">
        <TableHeader>
          <TableRow className="bg-slate-50 hover:bg-slate-50">
            <TableHead className="w-[148px] text-xs font-semibold text-slate-500">Ngày</TableHead>
            <TableHead className="w-[100px] text-xs font-semibold text-slate-500">Loại</TableHead>
            <TableHead className="w-[120px] text-right text-xs font-semibold text-slate-500">Số tiền</TableHead>
            <TableHead className="text-xs font-semibold text-slate-500">Người nhận</TableHead>
            <TableHead className="w-[148px] text-xs font-semibold text-slate-500">Tham chiếu</TableHead>
            <TableHead className="w-[110px] text-xs font-semibold text-slate-500">Trạng thái</TableHead>
            <TableHead className="w-6" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {transactions.map((tx) => {
            const isInflow = tx.amount > 0;
            return (
              <TableRow
                key={`${tx.type}-${tx.id}-${tx.reference}`}
                onClick={() => onSelect(tx)}
                className="cursor-pointer hover:bg-slate-50/80 group transition-colors"
              >
                <TableCell className="text-sm text-slate-600 tabular-nums">
                  {formatDateTime(tx.occurred_at)}
                </TableCell>
                <TableCell>
                  <TypeLabel type={tx.type} />
                </TableCell>
                <TableCell
                  className={`text-right font-semibold tabular-nums text-sm ${
                    isInflow ? "text-emerald-700" : "text-slate-800"
                  }`}
                >
                  {formatAmount(tx.amount)}
                </TableCell>
                <TableCell className="text-sm">
                  {tx.counterparty ? (() => {
                    const { name, detail } = parseCounterparty(tx.counterparty);
                    return (
                      <div>
                        <div className="text-slate-800 font-medium truncate max-w-[180px]">{name}</div>
                        {detail && <div className="text-slate-500 text-xs truncate max-w-[180px]">{detail}</div>}
                      </div>
                    );
                  })() : (
                    <span className="text-slate-500 text-sm">
                      {tx.type === "topup" ? "Nạp ví nội bộ" : "—"}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-0.5">
                    <span className="text-slate-500 font-mono text-xs truncate max-w-[140px]">
                      {tx.reference || "—"}
                    </span>
                    {tx.reference && <CopyButton value={tx.reference} />}
                  </div>
                </TableCell>
                <TableCell>
                  <StatusBadge status={tx.status} />
                </TableCell>
                <TableCell className="pr-3">
                  <ChevronRight className="h-4 w-4 text-slate-300 group-hover:text-slate-500 transition-colors" />
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </div>
  );
}

// ── Mobile transaction list ───────────────────────────────────────────────────

function MobileTransactionList({
  transactions,
  onSelect,
}: {
  transactions: UnifiedTransaction[];
  onSelect: (tx: UnifiedTransaction) => void;
}) {
  return (
    <div className="-mx-4 flex flex-col divide-y divide-slate-100 border-y border-slate-200">
      {transactions.map((tx) => {
        const isInflow = tx.amount > 0;
        const cpName = tx.counterparty
          ? parseCounterparty(tx.counterparty).name
          : tx.type === "topup" ? "Nạp ví nội bộ" : "—";
        const cpDetail = tx.counterparty ? parseCounterparty(tx.counterparty).detail : null;
        return (
          <button
            key={`${tx.type}-${tx.id}-${tx.reference}`}
            type="button"
            onClick={() => onSelect(tx)}
            className="flex items-center gap-2 px-3 py-2.5 text-left hover:bg-slate-50 active:bg-slate-100 transition-colors w-full"
          >
            {/* Icon */}
            <div className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ${isInflow ? "bg-emerald-50 ring-1 ring-emerald-100" : "bg-slate-100 ring-1 ring-slate-200"}`}>
              {isInflow
                ? <ArrowDownLeft className="h-4 w-4 text-emerald-600" />
                : <ArrowUpRight className="h-4 w-4 text-slate-500" />
              }
            </div>

            {/* Content */}
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5 flex-wrap mb-0.5">
                <span className="text-[13.5px] text-slate-800 font-semibold tracking-tight line-clamp-2 break-words">{cpName}</span>
                <StatusBadge status={tx.status} />
              </div>
              {cpDetail && <div className="text-xs text-slate-500 truncate leading-snug">{cpDetail}</div>}
              <div className="text-xs text-slate-500 mt-1 tabular-nums tracking-wide">{formatDateTime(tx.occurred_at)}</div>
            </div>

            {/* Amount + chevron */}
            <div className="flex items-center gap-1 shrink-0">
              <span className={`text-[13.5px] font-bold tabular-nums tracking-tight ${isInflow ? "text-emerald-700" : "text-slate-800"}`}>
                {formatAmount(tx.amount)}
              </span>
              <ChevronRight className="h-4 w-4 text-slate-300" />
            </div>
          </button>
        );
      })}
    </div>
  );
}

// ── Skeleton states ───────────────────────────────────────────────────────────

function SkeletonTable() {
  return (
    <div className="rounded-xl border border-slate-200 overflow-x-auto">
      <div className="grid grid-cols-7 gap-4 px-4 py-3 bg-slate-50 border-b border-slate-200 text-xs font-semibold text-slate-500 min-w-[820px]">
        <div>Ngày</div><div>Loại</div><div className="text-right">Số tiền</div><div>Người nhận</div><div>Tham chiếu</div><div>Trạng thái</div><div />
      </div>
      <div className="divide-y divide-slate-100">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="grid grid-cols-7 gap-4 px-4 py-4 items-center">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-7 w-20 rounded-full" />
            <Skeleton className="h-4 w-28 ml-auto" />
            <Skeleton className="h-4 w-36" />
            <Skeleton className="h-3 w-28" />
            <Skeleton className="h-6 w-20 rounded-full" />
            <Skeleton className="h-4 w-4 rounded" />
          </div>
        ))}
      </div>
    </div>
  );
}

function SkeletonCards() {
  return (
    <div className="-mx-4 flex flex-col divide-y divide-slate-100 border-y border-slate-200">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-4 py-3.5">
          <Skeleton className="h-10 w-10 shrink-0 rounded-xl" />
          <div className="flex-1 space-y-1.5">
            <Skeleton className="h-3.5 w-32" />
            <Skeleton className="h-2.5 w-20" />
          </div>
          <Skeleton className="h-4 w-20" />
        </div>
      ))}
    </div>
  );
}

// ── Empty / error states ──────────────────────────────────────────────────────

function EmptyState({
  hasFilters,
  onResetFilters,
}: {
  hasFilters: boolean;
  onResetFilters: () => void;
}) {
  if (hasFilters) {
    return (
      <SharedEmptyState
        title="Không tìm thấy giao dịch khớp bộ lọc"
        action={{ label: 'Xoá bộ lọc', onClick: onResetFilters }}
        size="sm"
      />
    );
  }
  return (
    <SharedEmptyState
      title="Chưa có giao dịch nào"
      description="Số dư được đồng bộ tự động từ nhà cung cấp."
      size="sm"
      className="rounded-xl border-2 border-dashed border-slate-200 bg-slate-50/50 px-4"
    />
  );
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="flex flex-col items-center justify-center py-14 text-center">
      <div className="flex h-12 w-12 items-center justify-center rounded-full bg-rose-50 mb-3">
        <AlertCircle className="h-5 w-5 text-rose-500" />
      </div>
      <p className="text-sm font-medium text-rose-700">Không thể tải lịch sử giao dịch</p>
      <p className="text-xs text-slate-500 mt-1 mb-4">Đã xảy ra lỗi khi gọi API. Vui lòng thử lại.</p>
      <Button variant="outline" size="sm" onClick={onRetry}>
        <RefreshCw className="h-4 w-4 mr-2" />
        Thử lại
      </Button>
    </div>
  );
}
