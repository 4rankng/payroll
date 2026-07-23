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

const EMPLOYEE_PRIMARY_ACTION =
  "ct-btn employee-type-action h-auto min-h-12 w-full gap-2 rounded-xl border-0 bg-[var(--employee-accent)] px-5 py-2 text-white normal-case shadow-[var(--employee-cta-shadow)] hover:bg-[var(--employee-accent-strong)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--employee-accent-ring)] disabled:!border-[var(--employee-border)] disabled:!bg-[var(--employee-page)] disabled:!text-[var(--employee-text-muted)] disabled:!shadow-none disabled:!opacity-100";

const EMPLOYEE_DISABLED_ACTION =
  "ct-btn employee-type-action h-auto min-h-12 w-full gap-2 rounded-xl border border-[var(--employee-border)] bg-[var(--employee-page)] px-5 py-2 text-[var(--employee-text-muted)] normal-case shadow-none cursor-not-allowed";

const PERIOD_CHIP_BASE =
  "ct-badge employee-type-pill inline-flex h-auto shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1.5";

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
  /** Amount shown as the card hero; null → render an em-dash. */
  amount: number | null;
  amountClassName: string;
  amountLabel: string;
  /** Single short helper line rendered under the card. */
  helperLine: string | null;
  /** Whether the primary action is enabled. */
  actionEnabled: boolean;
  actionLabel: string;
}

interface AdvancePaymentRequestFormProps {
  info: AdvancePaymentInfo;
  /** Payroll month selected by the page-level month navigator (`YYYY-MM`). */
  viewMonth?: string;
  /** Whether the selected calendar month is earlier than the current month. */
  isPastMonth?: boolean;
  /** Self-check-in employees earn their quota from attendance, not timesheets. */
  isSelfCheckInFlow?: boolean;
  history?: AdvancePaymentHistoryItem[];
  feeDetails: { fee: number; netAmount: number } | null;
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
      return "Số tiền phải là bội số của 5.000 VND";
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

  // Single source of truth for the card's state — one switch, consumed by the
  // render. Eliminates the previous scattered requestUnavailable / awaitingPayroll
  // / showUnavailableContext booleans and guarantees the amount, chip, and label
  // never disagree (the original misleading-state bug).
  const status = useMemo<PeriodStatus>(() => {
    if (isPastMonth) {
      return {
        variant: "closed",
        chipLabel: "Đã đóng",
        chipClassName: "ct-badge-ghost bg-[var(--employee-page)] text-[var(--employee-text-secondary)] border border-[var(--employee-border)]",
        chipIcon: "lock",
        amount: null,
        amountClassName: "text-[var(--employee-text-muted)]",
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
        chipClassName: "ct-badge-warning ct-badge-outline",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-[var(--employee-text-muted)]",
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
        chipClassName: "ct-badge-warning ct-badge-outline",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-[var(--employee-text-muted)]",
        amountLabel: "Chưa có hạn mức",
        helperLine: `Chờ bảng lương tháng ${viewedMonthLabel}`,
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    if (quotaExhausted) {
      return {
        variant: "exhausted",
        chipLabel: "Đã dùng hết hạn mức",
        chipClassName: "ct-badge-warning ct-badge-outline",
        chipIcon: "clock",
        amount: 0,
        amountClassName: "text-[var(--employee-text-muted)]",
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
        chipClassName: "ct-badge-warning ct-badge-outline",
        chipIcon: "clock",
        amount: null,
        amountClassName: "text-[var(--employee-text-muted)]",
        amountLabel: info.canRequestTitle || "Chưa thể ứng lương",
        helperLine: info.canRequestReason || "Vui lòng quay lại trong kỳ ứng lương tiếp theo.",
        actionEnabled: false,
        actionLabel: "Ứng lương",
      };
    }
    return {
      variant: "open",
      chipLabel: "Đang mở",
      chipClassName: "ct-badge-success ct-badge-outline bg-white/90",
      chipIcon: "check",
      amount: selectedQuotaRemaining,
      amountClassName: "text-[var(--employee-accent)]",
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

  return (
    <div
      className={cn(
        "ct-card employee-surface-card overflow-hidden bg-[var(--employee-surface)] p-4",
        className,
      )}
      style={style}
    >
      {/* ============== Section A — primary card (selected month) ============== */}
      <div
        className={cn(
          "-mx-4 -mt-4 rounded-t-2xl px-4 pb-4 pt-4",
          isOpen
            ? "bg-[var(--employee-summary-wash)]"
            : "bg-[var(--employee-surface)]",
        )}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="employee-type-label-caps text-[var(--employee-text-secondary)]">
              Ứng lương tháng {viewedMonthLabel}
            </p>
            <p className="employee-type-payroll-value mt-1 text-[var(--employee-text)] tabular-nums">
              {formatPayrollMonthRange(selectedMonth)}
            </p>
          </div>
          <span className={cn(PERIOD_CHIP_BASE, status.chipClassName)} aria-label={status.chipLabel}>
            {status.chipIcon === "check" && (
              <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            {status.chipIcon === "clock" && (
              <Clock3 className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            {status.chipIcon === "lock" && (
              <Lock className="h-3.5 w-3.5" aria-hidden="true" />
            )}
            {status.chipLabel}
          </span>
        </div>

        <p
          className={cn(
            "employee-type-hero-amount mt-4 break-words tabular-nums",
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
        <p
          className={cn(
            "employee-type-label mt-2",
            isOpen
              ? "text-[var(--employee-text-secondary)]"
              : "text-[var(--employee-text-muted)]",
          )}
        >
          {status.amountLabel}
        </p>

        {/* Historical quota details stay visible only for the selected month. */}
        {showQuotaProgress && (
          <div className="mt-4">
            <progress
              className="ct-progress ct-progress-primary h-2 w-full bg-white/90"
              role="progressbar"
              aria-label="Hạn mức ứng lương đã sử dụng"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(progressValue)}
              value={progressValue}
              max={100}
            />
            <div className="employee-type-body-sm mt-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-[var(--employee-text-secondary)]">
              <span>Đã dùng {formatCurrency(allowanceUsedAmount)}</span>
              <span className="font-medium text-[var(--employee-text)]">
                Hạn mức {formatCurrency(quotaSummary.maxAdvanceAmount)}
              </span>
            </div>
          </div>
        )}

        {/* Submitted-request acknowledgement replaces the controls. */}
        {visibleConfirmation && (
          <div
            className="mt-4 flex items-start gap-3 rounded-xl border border-[var(--employee-accent-border)] bg-white px-3.5 py-3.5"
            role="status"
            aria-live="polite"
          >
            <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]">
              <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <p className="employee-type-warning-title text-[var(--employee-text)]">
                Yêu cầu đã gửi
              </p>
              <p className="employee-type-body-sm mt-0.5 text-[var(--employee-text-secondary)]">
                {formatCurrency(visibleConfirmation.amount)} · Gửi ngày{" "}
                {formatDate(visibleConfirmation.submittedAt)}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* ============== Form controls — only in the open state ============== */}
      {showFormControls && (
        <>
          <div className="mt-4 border-t border-[#EAECF0] pt-4">
            <label
              htmlFor="advance-payment-amount"
              className="employee-type-label block text-[#475467]"
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
                  "ct-input ct-input-bordered employee-type-card-amount h-14 w-full rounded-xl bg-white px-3.5 pr-16 text-[var(--employee-text)] placeholder:text-[var(--employee-text-muted)]",
                  validationError ? "ct-input-error" : "focus:border-primary",
                )}
              />
              <span className="employee-type-label absolute right-3 top-1/2 -translate-y-1/2 text-[#667085]">
                VND
              </span>
            </div>
          </div>

          {quickAmounts.length > 0 && (
            <div className="mt-2.5 grid grid-cols-3 gap-2" aria-label="Chọn nhanh số tiền">
              {quickAmounts.map((quickAmount) => (
                <button
                  key={quickAmount.label}
                  type="button"
                  aria-pressed={numericAmount === quickAmount.amount}
                  onClick={() => setAmountFromNumber(quickAmount.amount)}
                  className={cn(
                    "ct-btn ct-btn-sm employee-type-action h-auto min-h-11 rounded-lg px-2 py-2 normal-case",
                    numericAmount === quickAmount.amount
                      ? "border-0 bg-[var(--employee-accent)] text-white hover:bg-[var(--employee-accent-strong)]"
                      : "ct-btn-outline",
                  )}
                >
                  {quickAmount.label}
                </button>
              ))}
            </div>
          )}

          {validationError && (
            <p className="ct-alert ct-alert-error employee-type-body mt-3 items-start gap-1.5 rounded-lg px-3 py-2 shadow-none">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              {validationError}
            </p>
          )}

          {canShowFeePreview && (
            <div className="mt-4 grid grid-cols-2 divide-x divide-[#EAECF0] border-y border-[#EAECF0] py-3">
              <div className="px-3">
                <p className="employee-type-label text-[#667085]">Phí chuyển tiền</p>
                <p className="employee-type-inline-amount mt-1 text-[#475467] tabular-nums">
                  {feeDetails ? formatCurrency(feeDetails.fee) : "Đang tính..."}
                </p>
              </div>
              <div className="px-3 text-right">
                <p className="employee-type-label text-[#667085]">Bạn thực nhận</p>
                <p className="employee-type-inline-amount mt-1 text-[var(--employee-accent)] tabular-nums">
                  {feeDetails ? formatCurrency(feeDetails.netAmount) : "Đang tính..."}
                </p>
              </div>
            </div>
          )}
        </>
      )}

      {/* ============== Primary action ============== */}
      {!visibleConfirmation &&
        (status.variant === "no-bank" && onBankAction ? (
          <button
            type="button"
            onClick={onBankAction}
            className={cn(EMPLOYEE_PRIMARY_ACTION, "mt-4")}
          >
            Xem tài khoản nhận tiền
            <ArrowRight className="h-4 w-4" aria-hidden="true" />
          </button>
        ) : showFormControls ? (
          <button
            type="button"
            onClick={handleSubmit}
            disabled={!canSubmit}
            className={cn(EMPLOYEE_PRIMARY_ACTION, "mt-4")}
          >
            {isPending ? (
              <>
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                Đang gửi...
              </>
            ) : (
              <>
                Yêu cầu ứng lương
                <ArrowRight className="h-4 w-4" aria-hidden="true" />
              </>
            )}
          </button>
        ) : (
          <button type="button" disabled className={cn(EMPLOYEE_DISABLED_ACTION, "mt-4")}>
            {status.actionLabel}
            <Lock className="h-4 w-4" aria-hidden="true" />
          </button>
        ))}

      {/* ============== Pending-request strip (preserved) ============== */}
      {!visibleConfirmation && latestPendingRequest && (
        <div
          className="mt-4 flex items-center gap-2.5 border-t border-[#EAECF0] pt-3"
          role="status"
        >
          <Clock3 className="h-4 w-4 shrink-0 text-[#B54708]" />
          <div className="min-w-0 flex-1">
            <p className="employee-type-label text-[#B54708]">Yêu cầu đang chờ xử lý</p>
            {latestPendingDate && (
              <p className="employee-type-body-sm mt-0.5 text-[#B54708]">
                Gửi ngày {latestPendingDate}
              </p>
            )}
          </div>
          <span className="employee-type-inline-amount shrink-0 text-[#B54708] tabular-nums">
            {formatCurrency(latestPendingRequest.requestAmount)}
          </span>
        </div>
      )}

      {/* ============== One short helper line ============== */}
      {status.helperLine && !visibleConfirmation && (
        <p className="employee-type-body-sm mt-3 text-center text-[var(--employee-text-secondary)]">
          {status.helperLine}
        </p>
      )}
    </div>
  );
}
