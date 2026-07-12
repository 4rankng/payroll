import { useState, useCallback, useMemo, useEffect } from "react";
import { AlertCircle, ArrowRight, CheckCircle2, Clock3 } from "lucide-react";
import { formatCurrency, formatDate } from "@/utils/formatters";
import {
  formatMonthShort,
  formatPayrollMonthRange,
  getAdvanceQuotaSummary,
  getAdvanceQuotaSummaryForMonth,
} from "@/utils/advancePaymentHelpers";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import type { AdvancePaymentHistoryItem, AdvancePaymentInfo } from "@/types/api/advance-payment.types";
import { EmployeePayIllustration } from "@/components/employees/EmployeePayIllustration";

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
    [history, info]
  );
  const quotaSummary = useMemo(
    () => viewMonth
      ? getAdvanceQuotaSummaryForMonth(info, viewMonth, history)
      : actionableQuotaSummary,
    [actionableQuotaSummary, history, info, viewMonth]
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
    [history, selectedMonth]
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
    // Provider minimum: net amount (after fee) must meet the disbursement provider's floor.
    if (providerMin > 0 && feeDetails && feeDetails.netAmount < providerMin) {
      return `Số tiền thực nhận sau phí phải tối thiểu ${formatCurrency(providerMin)}`;
    }
    // Provider maximum: net amount (after fee) must not exceed the disbursement provider's ceiling.
    const providerMax = info.providerMaxTransferAmount ?? 0;
    if (providerMax > 0 && feeDetails && feeDetails.netAmount > providerMax) {
      return `Số tiền thực nhận không được vượt quá ${formatCurrency(providerMax)}`;
    }
    return null;
  }, [amountValidationError, numericAmount, feeDetails, providerMin, info.providerMaxTransferAmount]);

  const validationError = amountValidationError ?? transferValidationError;
  const canShowFeePreview =
    numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT && !amountValidationError;

  const canSubmit =
    info.canRequest &&
    isViewingActionableMonth &&
    hasBankDestination &&
    !!numericAmount &&
    numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT &&
    !!feeDetails &&
    !validationError &&
    !isPending &&
    selectedQuotaRemaining > 0;
  const awaitingPayroll = !isSelfCheckInFlow && !isPastMonth && quotaSummary.maxAdvanceAmount <= 0;
  const quotaExhausted = quotaSummary.maxAdvanceAmount > 0 && selectedQuotaRemaining <= 0;
  const requestUnavailable =
    isPastMonth ||
    awaitingPayroll ||
    !isViewingActionableMonth ||
    !info.canRequest ||
    !hasBankDestination ||
    quotaExhausted;
  const showUnavailableContext = !isPastMonth && !quotaExhausted;
  const allowanceUsedAmount = Math.min(
    quotaSummary.maxAdvanceAmount,
    Math.max(quotaSummary.usedAmount, quotaSummary.maxAdvanceAmount - quotaSummary.remainingAmount)
  );
  const progressValue = quotaSummary.maxAdvanceAmount > 0
    ? Math.min(100, Math.max(0, Math.round((allowanceUsedAmount / quotaSummary.maxAdvanceAmount) * 100)))
    : 0;
  const status = useMemo(() => {
    if (isPastMonth) return { label: "Kỳ lương trước", tone: "neutral" as const };
    if (awaitingPayroll) return { label: "Chờ cập nhật bảng công", tone: "warning" as const };
    if (!hasBankDestination) return { label: "Thiếu tài khoản nhận tiền", tone: "warning" as const };
    if (quotaExhausted) return { label: "Đã dùng hết hạn mức", tone: "warning" as const };
    if (!isViewingActionableMonth) return { label: "Kỳ lương trước", tone: "neutral" as const };
    return { label: "Còn hạn mức", tone: "success" as const };
  }, [awaitingPayroll, hasBankDestination, isPastMonth, isViewingActionableMonth, quotaExhausted]);

  useEffect(() => {
    onAmountChange?.(numericAmount);
  }, [numericAmount, onAmountChange]);

  const handleAmountChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value.replace(/\D/g, "");
      setAmount(value);
    },
    []
  );

  const setAmountFromNumber = useCallback((value: number) => {
    const rounded = Math.max(0, Math.round(value / 5000) * 5000);
    setAmount(rounded ? rounded.toString() : "");
  }, []);

  const formatAmountInput = useCallback(
    (v: string) => (!v ? "" : parseInt(v, 10).toLocaleString("vi-VN")),
    []
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
        Math.round(rawAmount / 5000) * 5000
      );
      return {
        label,
        amount: Math.min(roundedAmount, maxValidAmount),
      };
    });

    return candidates.filter(
      (candidate, index) =>
        candidates.findLastIndex((item) => item.amount === candidate.amount) === index
    );
  }, [selectedQuotaRemaining]);

  const handleSubmit = useCallback(() => {
    if (numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return;
    if (validationError) return;
    if (!feeDetails) return;
    onSubmit({ amount: numericAmount, forMonth: selectedMonth });
  }, [feeDetails, numericAmount, selectedMonth, onSubmit, validationError]);

  const visibleConfirmation = requestConfirmation?.forMonth === selectedMonth
    ? requestConfirmation
    : null;

  return (
    <div
      className={className ?? "overflow-hidden rounded-2xl border border-[var(--employee-border)] bg-white p-4 shadow-[var(--employee-shadow)]"}
      style={style}
    >
      <div className="-mx-4 -mt-4 rounded-t-2xl border-b border-[var(--employee-accent-border)] bg-[var(--employee-summary-wash)] px-4 pb-4 pt-4">
        <div className="flex items-start justify-between gap-3">
          <p className="employee-type-label-caps pt-1 text-[var(--employee-accent)]">Số tiền có thể ứng</p>
          {!awaitingPayroll && (
            <span
              className={`employee-type-pill inline-flex shrink-0 items-center gap-1.5 rounded-full px-2.5 py-1.5 ${
                status.tone === "success"
                  ? "bg-white/90 text-[var(--employee-accent)] ring-1 ring-[var(--employee-accent-border)]"
                  : status.tone === "warning"
                    ? "bg-[var(--employee-warning-soft)] text-[var(--employee-warning-strong)] ring-1 ring-[var(--employee-warning-border)]"
                    : "bg-white/90 text-[var(--employee-text-secondary)] ring-1 ring-[var(--employee-border)]"
              }`}
            >
              {status.tone === "success" && <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />}
              {status.label}
            </span>
          )}
        </div>

        <div className="mt-1 flex items-center gap-2.5">
          <EmployeePayIllustration />
          <p className="employee-type-hero-amount min-w-0 break-words text-[var(--employee-accent)] tabular-nums">
            {formatCurrency(selectedQuotaRemaining)}
          </p>
        </div>

        <div className="mt-4 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
          <span className="employee-type-payroll-label text-[var(--employee-text-secondary)]">Kỳ lương</span>
          <span className="employee-type-payroll-value text-[var(--employee-text)] tabular-nums">
            {formatPayrollMonthRange(selectedMonth)}
          </span>
        </div>

        <div className="mt-3">
          <div
            className="h-2 overflow-hidden rounded-full bg-white/90 ring-1 ring-inset ring-[var(--employee-accent-border)]"
            role="progressbar"
            aria-label="Hạn mức ứng lương đã sử dụng"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={Math.round(progressValue)}
          >
            <span
              className="block h-full origin-left rounded-full bg-[var(--employee-accent)] transition-transform duration-300"
              style={{ transform: `scaleX(${progressValue / 100})` }}
            />
          </div>
          <div className="employee-type-body-sm mt-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-[var(--employee-text-secondary)]">
            <span>Đã dùng {formatCurrency(allowanceUsedAmount)}</span>
            <span className="font-medium text-[var(--employee-text)]">Hạn mức {formatCurrency(quotaSummary.maxAdvanceAmount)}</span>
          </div>
        </div>
      </div>

      {visibleConfirmation ? (
        <div
          className="employee-advance-request-confirm mt-4 flex items-start gap-3 rounded-xl border border-[var(--employee-accent-border)] bg-[var(--employee-summary-wash)] px-3.5 py-3.5"
          role="status"
          aria-live="polite"
        >
          <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-white text-[var(--employee-accent)] shadow-sm">
            <CheckCircle2 className="h-5 w-5" aria-hidden="true" />
          </span>
          <div className="min-w-0">
            <p className="employee-type-warning-title text-[var(--employee-text)]">Yêu cầu đã gửi</p>
            <p className="employee-type-body-sm mt-0.5 text-[var(--employee-text-secondary)]">
              {formatCurrency(visibleConfirmation.amount)} · Gửi ngày {formatDate(visibleConfirmation.submittedAt)}
            </p>
          </div>
        </div>
      ) : requestUnavailable ? showUnavailableContext ? (
        <div className="mt-4 border-t border-[#EAECF0] pt-4" role="status">
          <div className="flex items-start gap-2.5 rounded-[10px] border border-[var(--employee-warning-border)] bg-[var(--employee-warning-soft)] px-3 py-2.5">
            <Clock3 className="mt-0.5 h-4 w-4 shrink-0 text-[var(--employee-warning)]" aria-hidden="true" />
            <div className="min-w-0">
              <p className="employee-type-warning-title text-[var(--employee-warning-strong)]">
                {awaitingPayroll
                  ? `Đang chờ bảng công tháng ${viewedMonthLabel}`
                  : !hasBankDestination
                  ? "Chưa có tài khoản nhận tiền"
                  : info.canRequestTitle || "Chưa thể ứng lương"}
              </p>
              <p className="employee-type-body-sm mt-0.5 text-[var(--employee-warning)]">
                {awaitingPayroll
                  ? `Hạn mức ứng lương sẽ hiển thị sau khi bảng công tháng ${viewedMonthLabel} được cập nhật.`
                  : !hasBankDestination
                  ? "Liên hệ quản lý để cập nhật thông tin ngân hàng trước khi ứng lương."
                  : info.canRequestReason || "Vui lòng quay lại trong kỳ ứng lương tiếp theo."}
              </p>
            </div>
          </div>
          {!hasBankDestination && (
            <button
              type="button"
              onClick={onBankAction}
              disabled={!onBankAction}
              className="employee-type-action mt-3 inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-[var(--employee-accent)] px-5 py-2 text-white shadow-[var(--employee-cta-shadow)] transition-transform duration-200 active:scale-[0.98] disabled:cursor-not-allowed disabled:bg-[#D0D5DD] disabled:shadow-none"
            >
              Xem tài khoản nhận tiền
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </button>
          )}
        </div>
      ) : null : (
        <>
          <div className="mt-4 border-t border-[#EAECF0] pt-4">
            <label htmlFor="advance-payment-amount" className="employee-type-label block text-[#475467]">
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
                className={`employee-type-card-amount h-14 w-full rounded-xl border bg-white px-3.5 pr-16 text-[var(--employee-text)] placeholder:text-[var(--employee-text-muted)] transition-all duration-200 focus:outline-none focus:ring-2 ${
                  validationError
                    ? "border-[#FDA29B] focus:ring-[#FECDCA]"
                    : "border-[#D0D5DD] focus:border-[var(--employee-accent)] focus:ring-[var(--employee-accent-ring)]"
                }`}
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
                  className={`employee-type-action min-h-11 rounded-lg border px-2 py-2 transition-all active:scale-[0.97] ${
                    numericAmount === quickAmount.amount
                      ? "border-[var(--employee-accent)] bg-[var(--employee-accent)] text-white"
                      : "border-[#D0D5DD] bg-white text-[#475467]"
                  }`}
                >
                  {quickAmount.label}
                </button>
              ))}
            </div>
          )}

          {validationError && (
            <p className="employee-type-body mt-3 flex items-start gap-1.5 rounded-lg bg-[#FEF3F2] px-3 py-2 text-[#B42318]">
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

          <button
            type="button"
            onClick={handleSubmit}
            disabled={!canSubmit}
            className={`employee-type-action mt-4 inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl px-5 py-2 text-white transition-transform duration-200 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-55 ${canSubmit ? "bg-[var(--employee-accent)] shadow-[var(--employee-cta-shadow)]" : "bg-[var(--employee-text-muted)]"}`}
          >
            {isPending ? (
              <>
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                Đang gửi...
              </>
            ) : (
              <>
                Yêu cầu ứng lương
                <ArrowRight className="h-4 w-4" />
              </>
            )}
          </button>
        </>
      )}

      {!visibleConfirmation && latestPendingRequest && (
        <div className="mt-4 flex items-center gap-2.5 border-t border-[#EAECF0] pt-3" role="status">
          <Clock3 className="h-4 w-4 shrink-0 text-[#B54708]" />
          <div className="min-w-0 flex-1">
            <p className="employee-type-label text-[#B54708]">Yêu cầu đang chờ xử lý</p>
            {latestPendingDate && (
              <p className="employee-type-body-sm mt-0.5 text-[#B54708]">Gửi ngày {latestPendingDate}</p>
            )}
          </div>
          <span className="employee-type-inline-amount shrink-0 text-[#B54708] tabular-nums">
            {formatCurrency(latestPendingRequest.requestAmount)}
          </span>
        </div>
      )}
    </div>
  );
}
