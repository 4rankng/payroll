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
  PENDING:   { bar: "bg-warning",    icon: Clock,        badge: "bg-warning/10 text-warning border-warning/30" },
  APPROVED:  { bar: "bg-info",       icon: CheckCircle,  badge: "bg-info/10 text-info border-info/30" },
  COMPLETED: { bar: "bg-success",    icon: CheckCircle,  badge: "bg-success/10 text-success border-success/30" },
  FAILED:    { bar: "bg-destructive",  icon: XCircle,      badge: "bg-destructive/10 text-destructive border-destructive/30" },
  CANCELLED: { bar: "bg-muted",       icon: Ban,          badge: "bg-muted text-muted-foreground border-border" },
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
    <div className="rounded-lg overflow-hidden border border-border/65 bg-card">
      <div className="flex">
        <div className={`w-1 shrink-0 ${cfg.bar}`} />
        <div className="flex-1 min-w-0 px-3.5 py-3">
          <div className="flex items-center gap-2 mb-1.5">
            <span className="text-sm font-bold text-foreground tabular-nums mr-auto">{safeFormat(item.requestAmount)}</span>
            <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium border shrink-0 ${cfg.badge}`}>
              <StatusIcon className="h-3 w-3" />
              {getVietnameseAdvancePaymentStatus(item.status)}
            </span>
            {item.status === "PENDING" && !showConfirmCancel && (
              <button onClick={handleCancelClick} className="text-xs font-medium text-destructive border border-destructive/30 rounded px-1.5 py-0.5 hover:bg-destructive/10 transition-colors shrink-0">
                Hủy
              </button>
            )}
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground flex-wrap">
            <span className="shrink-0">{safeDate(item.createdAt)}</span>
            <span className="text-muted-foreground">·</span>
            <span className="shrink-0">Nhận <span className="font-semibold text-foreground">{safeFormat(item.netAmount)}</span></span>
            <span className="text-muted-foreground">·</span>
            <span className="shrink-0">Phí {safeFormat(item.fee)}</span>
          </div>
          {showConfirmCancel && item.status === "PENDING" && (
            <div className="flex items-center gap-3 mt-2 pt-2 border-t border-border">              <span className="text-xs text-muted-foreground flex-1">Xác nhận hủy?</span>
              <button onClick={() => setShowConfirmCancel(false)} className="text-xs font-medium text-muted-foreground hover:text-foreground transition-colors">Không</button>
              <button onClick={handleCancelClick} className="text-xs font-semibold text-destructive hover:text-destructive transition-colors">Xác nhận</button>
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
        title="Ứng lương"
        side="bottom"
        className="h-[92vh] rounded-t-lg p-0 flex flex-col overflow-hidden bg-card"
      >
        {/* Drag handle */}
        <div className="flex justify-center pt-2.5 pb-1 flex-shrink-0">
          <div className="h-1 w-9 rounded-full bg-muted-foreground/25" />
        </div>
        {/* Header */}
        <div className="console-panel-header px-4 py-3 border-b border-border/65 bg-muted/40">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <div className="p-2 bg-primary/10 rounded-lg">
                <Wallet className="h-4 w-4 text-primary" />
              </div>
              <div>
                <h2 className="text-base font-bold text-foreground">Ứng lương</h2>
                <p className="text-xs text-muted-foreground">{info ? formatPayrollMonthRange(info.forMonth) : "Đang tải..."}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
          {infoLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-24 w-full rounded-lg bg-card/60" />)}
            </div>
          ) : (
            <>
              {/* Limit card */}
              <div className="rounded-lg p-4 border border-border/65 bg-card">
                <div className="flex items-center gap-2 mb-3">
                  <TrendingUp className="h-4 w-4 text-primary" />
                  <span className="text-sm font-bold text-foreground">Hạn mức tháng này</span>
                </div>
                {info && (
                  <>
                    <div className="flex items-end justify-between mb-3">
                      <div>
                        <p className="text-xs text-muted-foreground mb-0.5">Còn lại</p>
                        <p className="text-2xl font-bold text-primary">{formatCurrency(info.remainingAmount)}</p>
                      </div>
                      <div className="text-right">
                        <p className="text-xs text-muted-foreground mb-0.5">Tối đa</p>
                        <p className="text-sm font-semibold text-muted-foreground">{formatCurrency(info.maxAdvanceAmount)}</p>
                      </div>
                    </div>
                    <div className="h-1.5 bg-primary/10 rounded-full overflow-hidden mb-3">
                      <div className="h-full rounded-full transition-all" style={{ width: `${remainingRatio}%`, background: "linear-gradient(90deg, hsl(var(--primary)), hsl(220 80% 35%))" }} />
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      <div className="bg-success/10 rounded-lg px-3 py-2 border border-success/30">
                        <p className="text-xs text-muted-foreground mb-0.5">Đã ứng</p>
                        <p className="text-xs font-bold text-success">{formatCurrency(info.completedAmount)}</p>
                      </div>
                      <div className="bg-warning/10 rounded-lg px-3 py-2 border border-warning/30">
                        <p className="text-xs text-muted-foreground mb-0.5">Đang chờ</p>
                        <p className="text-xs font-bold text-warning">{formatCurrency(info.pendingAmount)}</p>
                      </div>
                    </div>
                  </>
                )}
              </div>

              {/* Request form */}
              {info?.canRequest ? (
                <div className="rounded-lg p-4 border border-border/65 bg-card">
                  <div className="flex items-center gap-2 mb-3">
                    <DollarSign className="h-4 w-4 text-primary" />
                    <h3 className="text-sm font-bold text-foreground">Yêu cầu ứng lương</h3>
                  </div>
                  <div className="space-y-3">
                    <div className="relative">
                      <input
                        type="text" inputMode="numeric" placeholder="Nhập số tiền..."
                        value={formatAmountInput(amount)} onChange={handleAmountChange}
                        className={`w-full h-11 px-4 pr-14 text-sm font-medium rounded-lg border bg-card/80 focus:bg-card focus:outline-none focus:ring-2 transition-all ${
                          error ? "border-destructive/40 focus:ring-destructive/20" : "border-border focus:ring-primary/30 focus:border-border"                        }`}
                      />
                      <span className="absolute right-4 top-1/2 -translate-y-1/2 text-xs text-muted-foreground font-medium">₫</span>
                    </div>

                    {error && (
                      <p className="text-xs text-destructive flex items-center gap-1">
                        <AlertCircle className="h-3 w-3 shrink-0" />{error}
                      </p>
                    )}
                    {isTouched && numericAmount > 0 && numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT && (
                      <p className="text-xs text-muted-foreground">Tối thiểu: {formatCurrency(ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT)}</p>
                    )}

                    {feeDetails && numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT && (
                      <div className="bg-primary/5 rounded-lg p-3 space-y-1.5 text-xs border border-border">                        <div className="flex justify-between text-muted-foreground">
                          <span>Yêu cầu</span>
                          <span className="font-medium text-foreground">{formatCurrency(numericAmount)}</span>
                        </div>
                        <div className="flex justify-between text-muted-foreground">
                          <span>Phí giao dịch</span>
                          <span className="font-medium text-destructive">-{formatCurrency(feeDetails.fee)}</span>
                        </div>
                        <div className="flex justify-between pt-1.5 border-t border-border">                          <span className="font-semibold text-foreground">Thực nhận</span>
                          <span className="font-bold text-success">{formatCurrency(feeDetails.netAmount)}</span>
                        </div>
                      </div>
                    )}

                    <Button
                      onClick={handleSubmit}
                      disabled={!numericAmount || numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT || requestMutation.isPending}
                      className="w-full h-11 text-sm font-bold rounded-lg border-0 text-white"
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
                <div className="rounded-lg p-4 bg-warning/10 border border-warning/30">
                  <div className="flex items-center gap-2 text-warning">
                    <AlertCircle className="h-4 w-4 shrink-0" />
                    <span className="text-sm font-medium">Bạn đã hết hạn mức ứng lương cho tháng này</span>
                  </div>
                </div>
              )}

              {/* History */}
              <div>
                <div className="flex items-center gap-2 mb-3">
                  <History className="h-4 w-4 text-primary" />
                  <h3 className="text-sm font-bold text-foreground">Lịch sử yêu cầu</h3>
                </div>
                {historyLoading ? (
                  <div className="space-y-2.5">
                    {[1, 2, 3].map((i) => <Skeleton key={i} className="h-16 w-full rounded-lg bg-card/60" />)}
                  </div>
                ) : history.length === 0 ? (
                  <div className="text-center py-8 rounded-lg border border-dashed border-border" style={{ background: "rgba(255,255,255,0.50)" }}>
                    <div className="w-10 h-10 bg-primary/10 rounded-full flex items-center justify-center mx-auto mb-2">
                      <History className="h-5 w-5 text-primary" />
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
