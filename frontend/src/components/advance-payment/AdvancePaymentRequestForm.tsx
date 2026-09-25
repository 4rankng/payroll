import { useState, useCallback, useMemo, useEffect } from "react";
import {
  AlertCircle,
  ArrowRight,
  CheckCircle2,
  Clock3,
  Lock,
} from "lucide-react";
import { formatCurrency, formatDate } from "@/utils/formatters";
import {
  formatMonthShort,
  formatPayrollMonthRange,
  getAdvanceQuotaSummary,
  getAdvanceQuotaSummaryForMonth,
} from "@/utils/advancePaymentHelpers";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import type { AdvancePaymentHistoryItem, AdvancePaymentInfo } from "@/types/api/advance-payment.types";
import { AnimatedCurrency } from "@/components/employees/AnimatedCurrency";
import { cn } from "@/lib/utils";

type PeriodVariant =
  | "open"
  | "upcoming"
  | "closed"
  | "exhausted"
  | "no-bank"
  | "server-blocked";

interface PeriodStatus {
  variant: PeriodVariant;
  chipLabel: string;
  chipClassName: string;
  chipIcon: "check" | "clock" | "lock" | null;
  amount: number | null;
  amountClassName: string;
  amountLabel: string;
  helperLine: string | null;
  actionEnabled: boolean;
  actionLabel: string;
}

interface AdvancePaymentRequestFormProps {
  info: AdvancePaymentInfo;
  viewMonth?: string;
  isPastMonth?: boolean;
  isSelfCheckInFlow?: boolean;
  history?: AdvancePaymentHistoryItem[];
  feeDetails: { fee: number; netAmount: number } | null;
  feeError?: boolean;
  onRetryFee?: () => void;
  hasBankDestination: boolean;
  onSubmit: (data: { amount: number; forMonth: string }) => void;
  onAmountChange?: (amount: number) => void;
  onBankAction?: () => void;
  isPending: boolean;
  requestConfirmation?: {
    amount: number;
    forMonth: string;
    submittedAt: string;
  } | null;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentRequestForm({
  info,
  viewMonth,
  isPastMonth = false,
  isSelfCheckInFlow = false,
  history,
  feeDetails,
  feeError = false,
  onRetryFee,
  hasBankDestination,
  onSubmit,
  onAmountChange,
  onBankAction,
  isPending,
  requestConfirmation,
  className,
  style,
}: AdvancePaymentRequestFormProps) {
  const [amount, setAmount] = useState("");

  const actionableQuotaSummary = useMemo(
    () => getAdvanceQuotaSummary(info, history),
    [history, info],
  );
  const quotaSummary = useMemo(
    () =>
      viewMonth
        ? getAdvanceQuotaSummaryForMonth(info, viewMonth, history)
        : actionableQuotaSummary,
    [actionableQuotaSummary, history, info, viewMonth],
  );
  const selectedMonth = quotaSummary.forMonth;
  const selectedQuotaRemaining = quotaSummary.remainingAmount;
  const isViewingActionableMonth = selectedMonth === actionableQuotaSummary.forMonth;
  const viewedMonthLabel = formatMonthShort(selectedMonth);

  const latestPendingRequest = useMemo(
    () =>
      history?.reduce<AdvancePaymentHistoryItem | null>((latest, item) => {
        if (item.status !== "PENDING") return latest;
        if (item.forMonth && item.forMonth !== selectedMonth) return latest;
        if (!latest) return item;
        return Date.parse(item.createdAt) > Date.parse(latest.createdAt) ? item : latest;
      }, null) ?? null,
    [history, selectedMonth],
  );
  const latestPendingDate = useMemo(() => {
    if (!latestPendingRequest?.createdAt) return null;
    return Number.isNaN(Date.parse(latestPendingRequest.createdAt))
      ? null
      : formatDate(latestPendingRequest.createdAt);
  }, [latestPendingRequest]);

  const numericAmount = useMemo(() => {
    const parsed = parseInt(amount.replace(/\D/g, ""), 10);
    return isNaN(parsed) ? 0 : parsed;
  }, [amount]);

  const providerMin = info.providerMinTransferAmount ?? 0;

  const amountValidationError = useMemo(() => {
    if (!numericAmount) return null;
    if (numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) {
      return `Số tiền tối thiểu là ${formatCurrency(ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT)}`;
    }
    if (numericAmount % 5000 !== 0) {
      return "Số tiền phải là bội số của 5.000 ₫";
    }
    if (numericAmount > selectedQuotaRemaining) {
      return `Số tiền không được vượt quá ${formatCurrency(selectedQuotaRemaining)}`;
    }
    return null;
  }, [selectedQuotaRemaining, numericAmount]);

  const transferValidationError = useMemo(() => {
    if (!numericAmount || amountValidationError) return null;
    if (providerMin > 0 && feeDetails && feeDetails.netAmount < providerMin) {
      return `Số tiền thực nhận sau phí phải tối thiểu ${formatCurrency(providerMin)}`;
    }
    const providerMax = info.providerMaxTransferAmount ?? 0;
    if (providerMax > 0 && feeDetails && feeDetails.netAmount > providerMax) {
      return `Số tiền thực nhận không được vượt quá ${formatCurrency(providerMax)}`;
    }
    return null;
  }, [amountValidationError, numericAmount, feeDetails, providerMin, info.providerMaxTransferAmount]);

  const validationError = amountValidationError ?? transferValidationError;
  const canShowFeePreview =
    numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT && !amountValidationError;

  const awaitingPayroll =
    !isSelfCheckInFlow && !isPastMonth && quotaSummary.maxAdvanceAmount <= 0;
  const quotaExhausted = quotaSummary.maxAdvanceAmount > 0 && selectedQuotaRemaining <= 0;
  const showQuotaProgress = quotaSummary.maxAdvanceAmount > 0;

  const status = useMemo<PeriodStatus>(() => {
    if (isPastMonth) {
      return {
        variant: "closed",
        chipLabel: "Đã đóng",
        chipClassName: "bg-slate-100 text-slate-600 border border-slate-200",
        chipIcon: "lock",
        amount: null,
        amountClassName: "text-slate-500",
        amountLabel: `Kỳ ứng lương ${viewedMonthLabel} kết thúc`,
        helperLine: "Kỳ ứng lương này đã đóng, chọn kỳ hiện tại để tiếp tục.",
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    if (!hasBankDestination) {
      return {
        variant: "no-bank",
        chipLabel: "Chưa mở",
        chipClassName: "bg-amber-50 text-amber-700 border border-amber-200",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-slate-500",
        amountLabel: "Chưa có tài khoản nhận tiền",
        helperLine: "Liên hệ quản lý cập nhật thông tin ngân hàng để ứng lương.",
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    if (awaitingPayroll) {
      return {
        variant: "upcoming",
        chipLabel: "Chưa mở",
        chipClassName: "bg-amber-50 text-amber-700 border border-amber-200",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-slate-500",
        amountLabel: "Chưa có hạn mức",
        helperLine: `Chờ bảng lương tháng ${viewedMonthLabel}`,
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    if (quotaExhausted) {
      return {
        variant: "exhausted",
        chipLabel: "Đã dùng hết",
        chipClassName: "bg-amber-50 text-amber-700 border border-amber-200",
        chipIcon: "clock",
        amount: 0,
        amountClassName: "text-slate-500",
        amountLabel: "Đã dùng hết hạn mức",
        helperLine: null,
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    if (!isViewingActionableMonth || !info.canRequest) {
      return {
        variant: "server-blocked",
        chipLabel: "Chưa mở",
        chipClassName: "bg-amber-50 text-amber-700 border border-amber-200",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-slate-500",
        amountLabel: info.canRequestTitle || "Chưa thể ứng lương",
        helperLine: info.canRequestReason || "Vui lòng quay lại trong kỳ ứng lương tiếp theo.",
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    return {
      variant: "open",
      chipLabel: "Đang mở",
      chipClassName: "bg-emerald-50 text-emerald-700 border border-emerald-200",
      chipIcon: "check",
      amount: selectedQuotaRemaining,
      amountClassName: "text-slate-900",
      amountLabel: "Có thể ứng",
      helperLine: null,
      actionEnabled: true,
      actionLabel: "Yêu cầu ứng lương",
    };
  }, [
    awaitingPayroll,
    hasBankDestination,
    info.canRequest,
    info.canRequestReason,
    info.canRequestTitle,
    isPastMonth,
    isViewingActionableMonth,
    quotaExhausted,
    selectedQuotaRemaining,
    viewedMonthLabel,
  ]);

  const canSubmit =
    status.actionEnabled &&
    isViewingActionableMonth &&
    info.canRequest &&
    hasBankDestination &&
    !!numericAmount &&
    numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT &&
    !!feeDetails &&
    !feeError &&
    !validationError &&
    !isPending &&
    selectedQuotaRemaining > 0;

  const allowanceUsedAmount = Math.min(
    quotaSummary.maxAdvanceAmount,
    Math.max(
      quotaSummary.usedAmount,
      quotaSummary.maxAdvanceAmount - quotaSummary.remainingAmount,
    ),
  );
  const progressValue =
    quotaSummary.maxAdvanceAmount > 0
      ? Math.min(
          100,
          Math.max(0, Math.round((allowanceUsedAmount / quotaSummary.maxAdvanceAmount) * 100)),
        )
      : 0;

  useEffect(() => {
    onAmountChange?.(numericAmount);
  }, [numericAmount, onAmountChange]);

  const handleAmountChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value.replace(/\D/g, "");
      setAmount(value);
    },
    [],
  );

  const setAmountFromNumber = useCallback((value: number) => {
    const rounded = Math.max(0, Math.round(value / 5000) * 5000);
    setAmount(rounded ? rounded.toString() : "");
  }, []);

  const formatAmountInput = useCallback(
    (v: string) => (!v ? "" : parseInt(v, 10).toLocaleString("vi-VN")),
    [],
  );

  const quickAmounts = useMemo(() => {
    const maxValidAmount = Math.floor(selectedQuotaRemaining / 5000) * 5000;
    if (maxValidAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return [];
    const candidates = [
      { label: "25%", ratio: 0.25 },
      { label: "50%", ratio: 0.5 },
      { label: "Tối đa", ratio: 1 },
    ].map(({ label, ratio }) => {
      const rawAmount = maxValidAmount * ratio;
      const roundedAmount = Math.max(
        ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT,
        Math.round(rawAmount / 5000) * 5000,
      );
      return {
        label,
        amount: Math.min(roundedAmount, maxValidAmount),
      };
    });

    return candidates.filter(
      (candidate, index) =>
        candidates.findLastIndex((item) => item.amount === candidate.amount) === index,
    );
  }, [selectedQuotaRemaining]);

  const handleSubmit = useCallback(() => {
    if (numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return;
    if (validationError) return;
    if (!feeDetails) return;
    onSubmit({ amount: numericAmount, forMonth: selectedMonth });
  }, [feeDetails, numericAmount, selectedMonth, onSubmit, validationError]);

  const visibleConfirmation =
    requestConfirmation?.forMonth === selectedMonth ? requestConfirmation : null;

  const isOpen = status.variant === "open";
  const showFormControls = isOpen && !visibleConfirmation;

  const StatusIcon = status.chipIcon === "check"
    ? CheckCircle2
    : status.chipIcon === "lock"
    ? Lock
    : Clock3;

  return (
    <div
      className={cn(
        "overflow-hidden rounded-2xl border border-slate-200/60 bg-white shadow-[0_1px_3px_rgba(0,0,0,0.04),0_4px_16px_rgba(0,0,0,0.03)]",
        className,
      )}
      style={style}
      role="region"
      aria-labelledby="employee-advance-title"
    >
      {/* Header section */}
      <div className="px-4 pb-4 pt-4">
        {/* Grid, not flex: minmax(0,1fr) lets the title truncate while the chip
            keeps its intrinsic width, so the label can never wrap. */}
        <div className="grid grid-cols-[minmax(0,1fr)_auto] items-end gap-2">
          <h2
            id="employee-advance-title"
            className="employee-type-label-caps min-w-0 break-words text-[var(--employee-text-secondary)]"
          >
            Ứng lương tháng {viewedMonthLabel}
          </h2>
          {/* shrink-0 + nowrap: the label must never wrap inside its own pill. */}
          <span
            className={cn(
              "inline-flex shrink-0 items-center justify-self-end gap-1.5 whitespace-nowrap rounded-full px-2.5 py-1 text-[0.6875rem] font-semibold",
              status.chipClassName,
            )}
            aria-label={status.chipLabel}
          >
            <StatusIcon className="h-3 w-3 shrink-0" aria-hidden="true" />
            {status.chipLabel}
          </span>
        </div>

        {/* Period date */}
        <p className="employee-type-label mt-2.5 text-[var(--employee-text-secondary)] tabular-nums">
          {formatPayrollMonthRange(selectedMonth)}
        </p>

        {/* Amount display — single focal point. */}
        <div className="mt-2 min-w-0">
          <p
            className={cn(
              "text-[1.75rem] font-bold leading-none tracking-tight tabular-nums break-words",
              status.amountClassName,
            )}
            aria-live="polite"
          >
            {status.amount === null ? (
              "—"
            ) : (
              <AnimatedCurrency amount={status.amount} />
            )}
          </p>
          <p className="mt-1.5 text-[0.75rem] font-medium text-slate-500">
            {status.amountLabel}
          </p>
        </div>

        {/* Quota progress — flat, not a card inside a card. The percentage sits
            on the bar it describes instead of competing with the amount. */}
        {showQuotaProgress && (
          <div className="mt-4">
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-[0.75rem] font-medium text-slate-500">Tiến độ hạn mức</span>
              <span className="text-[0.75rem] font-semibold tabular-nums text-slate-600">
                {progressValue}%
              </span>
            </div>
            <div
              className="relative mt-1.5 h-1.5 overflow-hidden rounded-full bg-slate-200/70"
              role="progressbar"
              aria-label="Hạn mức ứng lương đã sử dụng"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(progressValue)}
            >
              <div
                className="absolute inset-y-0 left-0 rounded-full bg-emerald-500 transition-[width] duration-500 ease-out"
                style={{ width: `${progressValue}%` }}
              />
            </div>
            <div className="mt-2 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 text-[0.75rem]">
              <span className="text-slate-500">Đã dùng {formatCurrency(allowanceUsedAmount)}</span>
              <span className="font-semibold tabular-nums text-slate-700">
                Hạn mức {formatCurrency(quotaSummary.maxAdvanceAmount)}
              </span>
            </div>
          </div>
        )}

        {/* Confirmation acknowledgement */}
        {visibleConfirmation && (
          <div
            className="mt-4 flex items-start gap-3.5 rounded-xl border border-emerald-200 bg-emerald-50/50 px-4 py-4"
            role="status"
            aria-live="polite"
          >
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-700">
              <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <p className="text-[0.9375rem] font-semibold text-slate-800">
                Yêu cầu đã gửi
              </p>
              <p className="mt-0.5 text-[0.8125rem] text-slate-500">
                {formatCurrency(visibleConfirmation.amount)} · Gửi ngày{" "}
                {formatDate(visibleConfirmation.submittedAt)}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Form controls — only in the open state */}
      {showFormControls && (
        // No divider: the amount field is part of the same summary block, so
        // spacing carries the separation instead of another rule line.
        <div className="px-4 pb-4 pt-4">
          <label
            htmlFor="advance-payment-amount"
            className="block text-[0.8125rem] font-semibold text-slate-600"
          >
            Số tiền muốn ứng
          </label>
          <div className="relative mt-2">
            <input
              id="advance-payment-amount"
              type="text"
              inputMode="numeric"
              autoComplete="off"
              aria-label="Số tiền muốn ứng"
              placeholder="0"
              value={formatAmountInput(amount)}
              onChange={handleAmountChange}
              className={cn(
                "h-14 w-full rounded-xl border bg-white px-4 pr-16 text-[1.125rem] font-semibold tabular-nums text-slate-900 placeholder:text-slate-300 transition-all duration-150",
                validationError
                  ? "border-red-300 focus:border-red-400 focus:ring-2 focus:ring-red-500/20"
                  : "border-slate-200 focus:border-emerald-400 focus:ring-2 focus:ring-emerald-500/20 focus:outline-none",
              )}
            />
            <span className="absolute right-4 top-1/2 -translate-y-1/2 text-[0.875rem] font-medium text-slate-500">
              ₫
            </span>
          </div>

          {/* Quick amount buttons */}
          {quickAmounts.length > 0 && (
            <div className="mt-3 grid grid-cols-3 gap-2" aria-label="Chọn nhanh số tiền">
              {quickAmounts.map((quickAmount) => (
                <button
                  key={quickAmount.label}
                  type="button"
                  aria-pressed={numericAmount === quickAmount.amount}
                  onClick={() => setAmountFromNumber(quickAmount.amount)}
                  className={cn(
                    "h-11 rounded-xl text-[0.8125rem] font-semibold transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 focus-visible:ring-offset-2 active:scale-95",
                    numericAmount === quickAmount.amount
                      ? "bg-gradient-to-r from-emerald-500 to-emerald-600 text-white shadow-md shadow-emerald-500/20"
                      : "border border-slate-200 bg-white text-slate-600 hover:border-emerald-300 hover:bg-emerald-50/50 hover:text-emerald-700",
                  )}
                >
                  {quickAmount.label}
                </button>
              ))}
            </div>
          )}

          {/* Validation error */}
          {validationError && (
            <div className="mt-3 flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 px-3.5 py-2.5 text-[0.8125rem] text-red-600">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              {validationError}
            </div>
          )}

          {/* Fee preview */}
          {canShowFeePreview && feeError ? (
            <div role="alert" className="mt-4 rounded-xl border border-red-200 bg-red-50 px-3.5 py-2.5 text-sm text-red-700">
              <p>Không thể tính phí chuyển tiền. Vui lòng thử lại.</p>
              {onRetryFee && (
                <button type="button" onClick={onRetryFee} className="mt-2 min-h-11 rounded-lg border border-red-300 bg-white px-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500">
                  Tính lại phí
                </button>
              )}
            </div>
          ) : canShowFeePreview && (
            <div className="mt-4 grid grid-cols-2 divide-x divide-slate-200 rounded-xl border border-slate-200 bg-slate-50/50">
              <div className="px-4 py-3">
                <p className="text-[0.75rem] font-medium text-slate-600">Phí chuyển tiền</p>
                <p className="mt-1 text-[1rem] font-semibold tabular-nums text-slate-600">
                  {feeDetails ? formatCurrency(feeDetails.fee) : "Đang tính..."}
                </p>
              </div>
              <div className="px-4 py-3 text-right">
                <p className="text-[0.75rem] font-medium text-slate-600">Bạn thực nhận</p>
                <p className="mt-1 text-[1rem] font-semibold tabular-nums text-emerald-700">
                  {feeDetails ? formatCurrency(feeDetails.netAmount) : "Đang tính..."}
                </p>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Primary action button */}
      {!visibleConfirmation &&
        (status.variant === "no-bank" && onBankAction ? (
          <div className="px-4 pb-4">
            <button
              type="button"
              onClick={onBankAction}
              className="flex h-14 w-full items-center justify-center gap-2.5 rounded-xl bg-gradient-to-r from-emerald-500 to-emerald-600 text-[0.9375rem] font-semibold text-white shadow-lg shadow-emerald-500/25 transition-all duration-200 hover:from-emerald-600 hover:to-emerald-700 hover:shadow-xl hover:shadow-emerald-500/30 active:scale-[0.98]"
            >
              Xem tài khoản nhận tiền
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        ) : showFormControls ? (
          <div className="px-4 pb-4">
            <button
              type="button"
              onClick={handleSubmit}
              disabled={!canSubmit}
              className="flex h-14 w-full items-center justify-center gap-2.5 rounded-xl bg-gradient-to-r from-emerald-500 to-emerald-600 text-[0.9375rem] font-semibold text-white shadow-lg shadow-emerald-500/25 transition-all duration-200 hover:from-emerald-600 hover:to-emerald-700 hover:shadow-xl hover:shadow-emerald-500/30 active:scale-[0.98] disabled:from-slate-200 disabled:to-slate-200 disabled:text-slate-500 disabled:shadow-none disabled:cursor-not-allowed disabled:hover:from-slate-200 disabled:hover:to-slate-200"
            >
              {isPending ? (
                <>
                  <span className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                  Đang gửi...
                </>
              ) : (
                <>
                  Yêu cầu ứng lương
                  <ArrowRight className="h-4 w-4" aria-hidden="true" />
                </>
              )}
            </button>
          </div>
        ) : (
          <div className="px-4 pb-4">
            <button
              type="button"
              disabled
              className="flex h-14 w-full items-center justify-center gap-2.5 rounded-xl border border-slate-200 bg-slate-50 text-[0.9375rem] font-semibold text-slate-500 cursor-not-allowed"
            >
              {status.actionLabel}
              <Lock className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        ))}

      {/* Pending request indicator */}
      {!visibleConfirmation && latestPendingRequest && (
        <div className="border-t border-slate-100 px-4 py-3.5" role="status">
          <div className="flex items-center gap-3">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-amber-100 text-amber-700">
              <Clock3 className="h-4 w-4" aria-hidden="true" />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-[0.8125rem] font-semibold text-amber-700">Yêu cầu đang chờ xử lý</p>
              {latestPendingDate && (
                <p className="mt-0.5 text-[0.75rem] text-amber-700">
                  Gửi ngày {latestPendingDate}
                </p>
              )}
            </div>
            <span className="shrink-0 text-[0.9375rem] font-bold tabular-nums text-amber-700">
              {formatCurrency(latestPendingRequest.requestAmount)}
            </span>
          </div>
        </div>
      )}

      {/* Helper line */}
      {status.helperLine && !visibleConfirmation && (
        <div className="border-t border-slate-100 px-4 py-3.5">
          <div className="flex items-start gap-2.5 rounded-lg bg-slate-50 px-3.5 py-3 text-[0.8125rem] text-slate-500">
            {status.chipIcon === "lock" ? (
              <Lock className="mt-0.5 h-4 w-4 shrink-0 text-slate-500" aria-hidden="true" />
            ) : (
              <Clock3 className="mt-0.5 h-4 w-4 shrink-0 text-slate-500" aria-hidden="true" />
            )}
            <p className="leading-relaxed">{status.helperLine}</p>
          </div>
        </div>
      )}
    </div>
  );
}
