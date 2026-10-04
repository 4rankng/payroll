import { useState, useCallback, useMemo, useEffect } from "react";
import {
  AlertCircle,
  ArrowRight,
  CheckCircle,
  Clock,
  Lock01,
} from "@untitledui/icons";
import { formatCurrency, formatDate } from "@/utils/formatters";
import {
  ADVANCE_REQUEST_WINDOW_OPEN_DAY,
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
  // Before window-open day the next check-in period is not requestable yet,
  // so an exhausted prior month must point at the opening date instead of a
  // dead-end "choose the current period".
  const exhaustedNextWindowLine = useMemo(() => {
    if (!isSelfCheckInFlow || !quotaExhausted) return null;
    const now = new Date();
    const currentMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
    if (now.getDate() < ADVANCE_REQUEST_WINDOW_OPEN_DAY) {
      return `Kỳ ứng lương tháng ${formatMonthShort(currentMonth)} sẽ mở vào ngày ${ADVANCE_REQUEST_WINDOW_OPEN_DAY}/${now.getMonth() + 1}.`;
    }
    return `Kỳ tháng ${formatMonthShort(currentMonth)} đang mở — chọn kỳ hiện tại để ứng tiếp.`;
  }, [isSelfCheckInFlow, quotaExhausted]);
  const showQuotaProgress = quotaSummary.maxAdvanceAmount > 0;

  const status = useMemo<PeriodStatus>(() => {
    if (isPastMonth) {
      return {
        variant: "closed",
        chipLabel: "Đã đóng",
        chipClassName: "bg-[var(--employee-page)] text-fg-secondary border border-[var(--employee-border-strong)]",
        chipIcon: "lock",
        amount: null,
        amountClassName: "text-fg-tertiary",
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
        chipClassName: "bg-[var(--employee-warning-soft)] text-[var(--employee-warning)] border border-[var(--employee-warning-border)]",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-fg-tertiary",
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
        chipClassName: "bg-[var(--employee-warning-soft)] text-[var(--employee-warning)] border border-[var(--employee-warning-border)]",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-fg-tertiary",
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
        chipClassName: "bg-[var(--employee-warning-soft)] text-[var(--employee-warning)] border border-[var(--employee-warning-border)]",
        chipIcon: "clock",
        amount: 0,
        amountClassName: "text-fg-tertiary",
        amountLabel: "Đã dùng hết hạn mức",
        helperLine: exhaustedNextWindowLine,
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    if (!isViewingActionableMonth || !info.canRequest) {
      return {
        variant: "server-blocked",
        chipLabel: "Chưa mở",
        chipClassName: "bg-[var(--employee-warning-soft)] text-[var(--employee-warning)] border border-[var(--employee-warning-border)]",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-fg-tertiary",
        amountLabel: info.canRequestTitle || "Chưa thể ứng lương",
        helperLine: info.canRequestReason || "Vui lòng quay lại trong kỳ ứng lương tiếp theo.",
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    return {
      variant: "open",
      chipLabel: "Đang mở",
      chipClassName: "bg-[var(--employee-accent-soft)] text-[var(--employee-accent)] border border-[var(--employee-accent-border)]",
      chipIcon: "check",
      amount: selectedQuotaRemaining,
      amountClassName: "text-[var(--employee-text)]",
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
    exhaustedNextWindowLine,
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
    ? CheckCircle
    : status.chipIcon === "lock"
    ? Lock01
    : Clock;

  return (
    <div
      className={cn(
        "overflow-hidden rounded-2xl border border-[var(--employee-border)] bg-[var(--employee-surface)] shadow-[var(--employee-shadow)]",
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
          <p className="mt-1.5 text-[0.75rem] font-medium text-fg-tertiary">
            {status.amountLabel}
          </p>
        </div>

        {/* Quota progress — flat, not a card inside a card. The percentage sits
            on the bar it describes instead of competing with the amount. */}
        {showQuotaProgress && (
          <div className="mt-4">
            <div className="flex items-baseline justify-between gap-3">
              <span className="text-[0.75rem] font-medium text-fg-tertiary">Tiến độ hạn mức</span>
              <span className="text-[0.75rem] font-semibold tabular-nums text-fg-secondary">
                {progressValue}%
              </span>
            </div>
            <div
              className="relative mt-1.5 h-1.5 overflow-hidden rounded-full bg-utility-gray-200"
              role="progressbar"
              aria-label="Hạn mức ứng lương đã sử dụng"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(progressValue)}
            >
              <div
                className="absolute inset-y-0 left-0 rounded-full bg-[var(--employee-accent)] transition-[width] duration-500 ease-out"
                style={{ width: `${progressValue}%` }}
              />
            </div>
            <div className="mt-2 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 text-[0.75rem]">
              <span className="text-fg-tertiary">Đã dùng {formatCurrency(allowanceUsedAmount)}</span>
              <span className="font-semibold tabular-nums text-fg-secondary">
                Hạn mức {formatCurrency(quotaSummary.maxAdvanceAmount)}
              </span>
            </div>
          </div>
        )}

        {/* Confirmation acknowledgement */}
        {visibleConfirmation && (
          <div
            className="mt-4 flex items-start gap-3.5 rounded-xl border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)]/50 px-4 py-4"
            role="status"
            aria-live="polite"
          >
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--employee-accent-ring)] text-[var(--employee-accent)]">
              <CheckCircle className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <p className="text-[0.9375rem] font-semibold text-[var(--employee-text)]">
                Yêu cầu đã gửi
              </p>
              <p className="mt-0.5 text-[0.8125rem] text-fg-tertiary">
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
            className="block text-[0.8125rem] font-semibold text-fg-secondary"
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
                "h-14 w-full rounded-xl border bg-[var(--employee-surface)] px-4 pr-16 text-[1.125rem] font-semibold tabular-nums text-[var(--employee-text)] placeholder:text-fg-disabled transition-all duration-150",
                validationError
                  ? "border-utility-error-300 focus:border-utility-error-400 focus:ring-2 focus:ring-utility-error-500/20"
                  : "border-[var(--employee-border-strong)] focus:border-[var(--employee-accent)] focus:ring-2 focus:ring-[var(--employee-accent-ring)] focus:outline-none",
              )}
            />
            <span className="absolute right-4 top-1/2 -translate-y-1/2 text-[0.875rem] font-medium text-fg-tertiary">
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
                    "h-11 rounded-xl text-[0.8125rem] font-semibold transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)] focus-visible:ring-offset-2 active:scale-95",
                    numericAmount === quickAmount.amount
                      ? "bg-[var(--employee-accent)] text-white shadow-sm"
                      : "border border-[var(--employee-border-strong)] bg-[var(--employee-surface)] text-fg-secondary hover:border-[var(--employee-accent-border)] hover:bg-[var(--employee-accent-soft)]/50 hover:text-[var(--employee-accent)]",
                  )}
                >
                  {quickAmount.label}
                </button>
              ))}
            </div>
          )}

          {/* Validation error */}
          {validationError && (
            <div className="mt-3 flex items-start gap-2 rounded-xl border border-utility-error-200 bg-[var(--employee-error-soft)] px-3.5 py-2.5 text-[0.8125rem] text-[var(--employee-error)]">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
              {validationError}
            </div>
          )}

          {/* Fee preview */}
          {canShowFeePreview && feeError ? (
            <div role="alert" className="mt-4 rounded-xl border border-utility-error-200 bg-[var(--employee-error-soft)] px-3.5 py-2.5 text-sm text-[var(--employee-error)]">
              <p>Không thể tính phí chuyển tiền. Vui lòng thử lại.</p>
              {onRetryFee && (
                <button type="button" onClick={onRetryFee} className="mt-2 min-h-11 rounded-lg border border-utility-error-300 bg-[var(--employee-surface)] px-3 font-semibold focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-focus-ring)]">
                  Tính lại phí
                </button>
              )}
            </div>
          ) : canShowFeePreview && (
            <div className="mt-4 grid grid-cols-2 divide-x divide-[var(--employee-border-strong)] rounded-xl border border-[var(--employee-border-strong)] bg-[var(--employee-surface-muted)]">
              <div className="px-4 py-3">
                <p className="text-[0.75rem] font-medium text-fg-secondary">Phí chuyển tiền</p>
                <p className="mt-1 text-[1rem] font-semibold tabular-nums text-fg-secondary">
                  {feeDetails ? formatCurrency(feeDetails.fee) : "Đang tính..."}
                </p>
              </div>
              <div className="px-4 py-3 text-right">
                <p className="text-[0.75rem] font-medium text-fg-secondary">Bạn thực nhận</p>
                <p className="mt-1 text-[1rem] font-semibold tabular-nums text-[var(--employee-accent)]">
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
              className="flex h-14 w-full items-center justify-center gap-2.5 rounded-xl bg-[var(--employee-accent)] text-[0.9375rem] font-semibold text-white shadow-[var(--employee-cta-shadow)] transition-all duration-200 hover:bg-[var(--employee-accent-strong)] active:scale-[0.98] focus-visible:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--employee-focus-ring)]"
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
              className="flex h-14 w-full items-center justify-center gap-2.5 rounded-xl bg-[var(--employee-accent)] text-[0.9375rem] font-semibold text-white shadow-[var(--employee-cta-shadow)] transition-all duration-200 hover:bg-[var(--employee-accent-strong)] active:scale-[0.98] focus-visible:outline-none focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-[var(--employee-focus-ring)] disabled:cursor-not-allowed disabled:bg-[var(--employee-border)] disabled:text-fg-tertiary disabled:shadow-none"
            >
              {isPending ? (
                <>
                  <span className="h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white motion-reduce:animate-none" />
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
              className="flex h-14 w-full items-center justify-center gap-2.5 rounded-xl border border-[var(--employee-border-strong)] bg-[var(--employee-page)] text-[0.9375rem] font-semibold text-fg-tertiary cursor-not-allowed"
            >
              {status.actionLabel}
              <Lock01 className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        ))}

      {/* Pending request indicator */}
      {!visibleConfirmation && latestPendingRequest && (
        <div className="border-t border-[var(--employee-border)] px-4 py-3.5" role="status">
          <div className="flex items-center gap-3">
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-[var(--employee-warning-soft)] text-[var(--employee-warning)]">
              <Clock className="h-4 w-4" aria-hidden="true" />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-[0.8125rem] font-semibold text-[var(--employee-warning)]">Yêu cầu đang chờ xử lý</p>
              {latestPendingDate && (
                <p className="mt-0.5 text-[0.75rem] text-[var(--employee-warning)]">
                  Gửi ngày {latestPendingDate}
                </p>
              )}
            </div>
            <span className="shrink-0 text-[0.9375rem] font-bold tabular-nums text-[var(--employee-warning)]">
              {formatCurrency(latestPendingRequest.requestAmount)}
            </span>
          </div>
        </div>
      )}

      {/* Helper line */}
      {status.helperLine && !visibleConfirmation && (
        <div className="border-t border-[var(--employee-border)] px-4 py-3.5">
          <div className="flex items-start gap-2.5 rounded-lg bg-[var(--employee-surface-muted)] px-3.5 py-3 text-[0.8125rem] text-fg-tertiary">
            {status.chipIcon === "lock" ? (
              <Lock01 className="mt-0.5 h-4 w-4 shrink-0 text-fg-tertiary" aria-hidden="true" />
            ) : (
              <Clock className="mt-0.5 h-4 w-4 shrink-0 text-fg-tertiary" aria-hidden="true" />
            )}
            <p className="leading-relaxed">{status.helperLine}</p>
          </div>
        </div>
      )}
    </div>
  );
}
