import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { Sheet, SheetContent } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  useAdvancePaymentInfo,
  useRequestAdvancePayment,
  useAdvancePaymentHistory,
  useCalculateFee,
  useCancelAdvancePaymentRequest,
} from "@/hooks/api/useAdvancePayments";
import { formatCurrency } from "@/utils/formatters";
import {
  validateAdvancePaymentAmount,
  formatPayrollMonthRange,
  getVietnameseAdvancePaymentStatus,
} from "@/utils/advancePaymentHelpers";
import {
  Wallet, AlertCircle, Clock, TrendingUp,
  CheckCircle, XCircle, DollarSign, Ban, History,
} from "lucide-react";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import type { AdvancePaymentHistoryItem } from "@/types/api/advance-payment.types";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";

const STATUS_CONFIG = {
  PENDING:   { bar: "bg-amber-400",   icon: Clock,        badge: "bg-amber-100 text-amber-700 border-amber-200" },
  APPROVED:  { bar: "bg-blue-500",    icon: CheckCircle,  badge: "bg-blue-100 text-blue-700 border-blue-200" },
  COMPLETED: { bar: "bg-emerald-500", icon: CheckCircle,  badge: "bg-emerald-100 text-emerald-700 border-emerald-200" },
  FAILED:    { bar: "bg-red-400",     icon: XCircle,      badge: "bg-red-100 text-red-700 border-red-200" },
  CANCELLED: { bar: "bg-slate-300",   icon: Ban,          badge: "bg-slate-100 text-muted-foreground border-border" },
} as const;

const HistoryItemCard = ({ item, onCancel }: { item: AdvancePaymentHistoryItem; onCancel?: (id: number) => void }) => {
  const [showConfirmCancel, setShowConfirmCancel] = useState(false);
  const cfg = STATUS_CONFIG[item.status as keyof typeof STATUS_CONFIG] ?? STATUS_CONFIG.CANCELLED;
  const StatusIcon = cfg.icon;

  const handleCancelClick = () => {
    if (showConfirmCancel) { onCancel?.(item.id); setShowConfirmCancel(false); }
    else { setShowConfirmCancel(true); setTimeout(() => setShowConfirmCancel(false), 3000); }
  };

  const safeFormat = (v: number | null | undefined) => v == null || isNaN(v as number) ? "—" : formatCurrency(v);
  const safeDate = (d: string | null | undefined) => {
    if (!d) return "—";
    const dt = new Date(d);
    return isNaN(dt.getTime()) ? "—" : format(dt, "dd/MM/yyyy", { locale: vi });
  };

  return (
    <div className="rounded-xl overflow-hidden glass-card">
      <div className="flex">
        <div className={`w-1 shrink-0 ${cfg.bar}`} />
        <div className="flex-1 min-w-0 px-3.5 py-3">
          <div className="flex items-center gap-2 mb-1.5">
            <span className="text-sm font-bold text-slate-800 tabular-nums mr-auto">{safeFormat(item.requestAmount)}</span>
            <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border shrink-0 ${cfg.badge}`}>
              <StatusIcon className="h-3 w-3" />
              {getVietnameseAdvancePaymentStatus(item.status)}
            </span>
            {item.status === "PENDING" && !showConfirmCancel && (
              <button onClick={handleCancelClick} className="text-xs font-medium text-red-500 border border-red-200 rounded px-1.5 py-0.5 hover:bg-red-50 transition-colors shrink-0">
                Hủy
              </button>
            )}
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground flex-wrap">
            <span className="shrink-0">{safeDate(item.createdAt)}</span>
            <span className="text-slate-300">·</span>
            <span className="shrink-0">Nhận <span className="font-semibold text-foreground">{safeFormat(item.netAmount)}</span></span>
            <span className="text-slate-300">·</span>
            <span className="shrink-0">Phí {safeFormat(item.fee)}</span>
          </div>
          {showConfirmCancel && item.status === "PENDING" && (
            <div className="flex items-center gap-3 mt-2 pt-2 border-t border-border">              <span className="text-xs text-muted-foreground flex-1">Xác nhận hủy?</span>
              <button onClick={() => setShowConfirmCancel(false)} className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">Không</button>
              <button onClick={handleCancelClick} className="text-xs font-semibold text-red-500 hover:text-red-700 transition-colors">Xác nhận</button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

interface AdvancePaymentSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export const AdvancePaymentSheet = ({ isOpen, onClose }: AdvancePaymentSheetProps) => {
  const [amount, setAmount] = useState<string>("");
  const [error, setError] = useState<string | null>(null);
  const [isTouched, setIsTouched] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const { data: infoResponse, isLoading: infoLoading, refetch: refetchInfo } = useAdvancePaymentInfo({ enabled: isOpen });
  const { data: historyResponse, isLoading: historyLoading } = useAdvancePaymentHistory({ page: 1, pageSize: 10 }, { enabled: isOpen });
  const calculateFeeMutation = useCalculateFee();
  const requestMutation = useRequestAdvancePayment();
  const cancelMutation = useCancelAdvancePaymentRequest();

  const info = infoResponse?.data;
  const history = historyResponse?.data || [];

  const numericAmount = useMemo(() => {
    const parsed = parseInt(amount.replace(/\D/g, ""), 10);
    return isNaN(parsed) ? 0 : parsed;
  }, [amount]);

  const [serverFeeDetails, setServerFeeDetails] = useState<{ fee: number; netAmount: number } | null>(null);

  useEffect(() => {
    if (numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) { setServerFeeDetails(null); return; }
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      calculateFeeMutation.mutate({ amount: numericAmount }, {
        onSuccess: (r) => setServerFeeDetails(r),
        onError: () => setServerFeeDetails(null),
      });
    }, 300);
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, [numericAmount, calculateFeeMutation]);

  const feeDetails = serverFeeDetails;

  const handleAmountChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value.replace(/\D/g, "");
    setAmount(value); setIsTouched(true); setError(null);
  }, []);

  const formatAmountInput = useCallback((v: string) => !v ? "" : parseInt(v, 10).toLocaleString("vi-VN"), []);

  const handleSubmit = useCallback(async () => {
    if (!info) return;
    const validationError = validateAdvancePaymentAmount(numericAmount, info.remainingAmount);
    if (validationError) { setError(validationError); return; }
    try {
      await requestMutation.mutateAsync({ amount: numericAmount, forMonth: info.forMonth });
      setAmount(""); setIsTouched(false); setServerFeeDetails(null);
      refetchInfo();
    } catch { /* handled */ }
  }, [numericAmount, info, requestMutation, refetchInfo]);

  const handleCancelRequest = useCallback((id: number) => { cancelMutation.mutate(id); }, [cancelMutation]);

  useEffect(() => {
    if (!isOpen) { setAmount(""); setError(null); setIsTouched(false); setServerFeeDetails(null); }
  }, [isOpen]);

  const remainingRatio = info && info.maxAdvanceAmount > 0
    ? Math.min((info.remainingAmount / info.maxAdvanceAmount) * 100, 100) : 0;

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="bottom"
        className="h-[92vh] rounded-t-2xl p-0 flex flex-col overflow-hidden bg-primary/5 shadow-[0_-4px_24px_rgba(0,0,0,0.08)]"
      >
        {/* Drag handle */}
        <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
          <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
        </div>
        {/* Header */}
        <div className="px-5 pt-5 pb-4 border-b border-border" style={{ background: "rgba(255,255,255,0.72)", backdropFilter: "blur(16px)" }}>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <div className="p-2 bg-sky-100 rounded-xl">
                <Wallet className="h-4 w-4 text-sky-600" />
              </div>
              <div>
                <h2 className="text-base font-bold text-slate-800">Ứng lương</h2>
                <p className="text-xs text-muted-foreground">{info ? formatPayrollMonthRange(info.forMonth) : "Đang tải..."}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
          {infoLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full rounded-xl bg-card/60" />)}
            </div>
          ) : (
            <>
              {/* Limit card */}
              <div className="rounded-xl p-4 glass-card">
                <div className="flex items-center gap-2 mb-3">
                  <TrendingUp className="h-4 w-4 text-sky-600" />
                  <span className="text-sm font-bold text-slate-800">Hạn mức tháng này</span>
                </div>
                {info && (
                  <>
                    <div className="flex items-end justify-between mb-3">
                      <div>
                        <p className="text-xs text-slate-400 mb-0.5">Còn lại</p>
                        <p className="text-2xl font-bold text-sky-600">{formatCurrency(info.remainingAmount)}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-xs text-slate-400 mb-0.5">Tối đa</p>
                        <p className="text-sm font-semibold text-muted-foreground">{formatCurrency(info.maxAdvanceAmount)}</p>
                      </div>
                    </div>
                    <div className="h-1.5 bg-sky-100 rounded-full overflow-hidden mb-3">
                      <div className="h-full rounded-full transition-all" style={{ width: `${remainingRatio}%`, background: "linear-gradient(90deg, hsl(var(--primary)), hsl(220 80% 35%))" }} />
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      <div className="bg-emerald-50/80 rounded-xl px-3 py-2 border border-emerald-100">
                        <p className="text-[10px] text-slate-400 mb-0.5">Đã ứng</p>
                        <p className="text-xs font-bold text-emerald-600">{formatCurrency(info.completedAmount)}</p>
                      </div>
                      <div className="bg-amber-50/80 rounded-xl px-3 py-2 border border-amber-100">
                        <p className="text-[10px] text-slate-400 mb-0.5">Đang chờ</p>
                        <p className="text-xs font-bold text-amber-600">{formatCurrency(info.pendingAmount)}</p>
                      </div>
                    </div>
                  </>
                )}
              </div>

              {/* Request form */}
              {info?.canRequest ? (
                <div className="rounded-xl p-4 glass-card">
                  <div className="flex items-center gap-2 mb-3">
                    <DollarSign className="h-4 w-4 text-sky-600" />
                    <h3 className="text-sm font-bold text-slate-800">Yêu cầu ứng lương</h3>
                  </div>
                  <div className="space-y-3">
                    <div className="relative">
                      <input
                        type="text" inputMode="numeric" placeholder="Nhập số tiền..."
                        value={formatAmountInput(amount)} onChange={handleAmountChange}
                        className={`w-full h-11 px-4 pr-14 text-sm font-medium rounded-xl border bg-card/80 focus:bg-card focus:outline-none focus:ring-2 transition-all ${
                          error ? "border-red-300 focus:ring-red-200" : "border-border focus:ring-sky-200 focus:border-border"                        }`}
                      />
                      <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs text-slate-400 font-medium">VND</span>
                    </div>

                    {error && (
                      <p className="text-xs text-red-500 flex items-center gap-1">
                        <AlertCircle className="h-3 w-3 shrink-0" />{error}
                      </p>
                    )}
                    {isTouched && numericAmount > 0 && numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT && (
                      <p className="text-xs text-slate-400">Tối thiểu: {formatCurrency(ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT)}</p>
                    )}

                    {feeDetails && numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT && (
                      <div className="bg-sky-50/80 rounded-xl p-3 space-y-1.5 text-xs border border-border">                        <div className="flex justify-between text-muted-foreground">
                          <span>Yêu cầu</span>
                          <span className="font-medium text-foreground">{formatCurrency(numericAmount)}</span>
                        </div>
                        <div className="flex justify-between text-muted-foreground">
                          <span>Phí giao dịch</span>
                          <span className="font-medium text-red-500">-{formatCurrency(feeDetails.fee)}</span>
                        </div>
                        <div className="flex justify-between pt-1.5 border-t border-border">                          <span className="font-semibold text-foreground">Thực nhận</span>
                          <span className="font-bold text-emerald-600">{formatCurrency(feeDetails.netAmount)}</span>
                        </div>
                      </div>
                    )}

                    <Button
                      onClick={handleSubmit}
                      disabled={!numericAmount || numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT || requestMutation.isPending}
                      className="w-full h-11 text-sm font-bold rounded-xl border-0 text-white"
                      style={{ background: "linear-gradient(135deg, hsl(var(--primary)), hsl(220 80% 35%))" }}
                    >
                      {requestMutation.isPending ? (
                        <span className="flex items-center gap-2">
                          <span className="animate-spin h-4 w-4 border-2 border-white/30 border-t-white rounded-full" />
                          Đang gửi...
                        </span>
                      ) : "Gửi yêu cầu"}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="rounded-xl p-4 bg-amber-50/80 border border-amber-200">
                  <div className="flex items-center gap-2 text-amber-800">
                    <AlertCircle className="h-4 w-4 shrink-0" />
                    <span className="text-sm font-medium">Bạn đã hết hạn mức ứng lương cho tháng này</span>
                  </div>
                </div>
              )}

              {/* History */}
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <History className="h-4 w-4 text-sky-600" />
                  <h3 className="text-sm font-bold text-slate-800">Lịch sử yêu cầu</h3>
                </div>
                {historyLoading ? (
                  <div className="space-y-2.5">
                    {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 w-full rounded-xl bg-card/60" />)}
                  </div>
                ) : history.length === 0 ? (
                  <div className="text-center py-8 rounded-xl border border-dashed border-border" style={{ background: "rgba(255,255,255,0.50)" }}>
                    <div className="w-10 h-10 bg-sky-100/80 rounded-full flex items-center justify-center mx-auto mb-2">
                      <History className="h-5 w-5 text-sky-400" />
                    </div>
                    <p className="text-sm text-muted-foreground">Chưa có yêu cầu nào</p>
                  </div>
                ) : (
                  <div className="space-y-2">
                    {history.map((item: AdvancePaymentHistoryItem) => (
                      <HistoryItemCard key={item.id} item={item} onCancel={handleCancelRequest} />
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
};

export default AdvancePaymentSheet;
