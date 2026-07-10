import { useState, useCallback, useMemo, useEffect } from "react";
import { AlertCircle, ArrowRight, Clock3 } from "lucide-react";
import { formatCurrency, formatDate } from "@/utils/formatters";
import { formatAdvancePeriodDisplay, getAdvanceQuotaSummary } from "@/utils/advancePaymentHelpers";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import type { AdvancePaymentHistoryItem, AdvancePaymentInfo } from "@/types/api/advance-payment.types";

interface AdvancePaymentRequestFormProps {
  info: AdvancePaymentInfo;
  history?: AdvancePaymentHistoryItem[];
  feeDetails: { fee: number; netAmount: number } | null;
  hasBankDestination: boolean;
  onSubmit: (data: { amount: number; forMonth: string }) => void;
  onAmountChange?: (amount: number) => void;
  isPending: boolean;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentRequestForm({
  info,
  history,
  feeDetails,
  hasBankDestination,
  onSubmit,
  onAmountChange,
  isPending,
  className,
  style,
}: AdvancePaymentRequestFormProps) {
  const [amount, setAmount] = useState("");

  const quotaSummary = useMemo(
    () => getAdvanceQuotaSummary(info, history),
    [history, info]
  );
  const selectedMonth = quotaSummary.forMonth;
  const selectedQuotaRemaining = quotaSummary.remainingAmount;

  const latestPendingRequest = useMemo(
    () =>
      history?.reduce<AdvancePaymentHistoryItem | null>((latest, item) => {
        if (item.status !== "PENDING") return latest;
        if (!latest) return item;
        return Date.parse(item.createdAt) > Date.parse(latest.createdAt) ? item : latest;
      }, null) ?? null,
    [history]
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
    hasBankDestination &&
    !!numericAmount &&
    numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT &&
    !!feeDetails &&
    !validationError &&
    !isPending &&
    selectedQuotaRemaining > 0;

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

  return (
    <div
      className={className ?? "rounded-2xl border border-[#E4E7EC] bg-white p-4"}
      style={style}
    >
      <div className="flex items-start justify-between gap-4 border-b border-[#EAECF0] pb-4">
        <div className="min-w-0">
          <p className="employee-type-label text-[#667085]">Có thể ứng</p>
          <p className="employee-type-hero-amount mt-1 truncate text-[#067647] tabular-nums">
            {formatCurrency(selectedQuotaRemaining)}
          </p>
        </div>
        <div className="shrink-0 text-right">
          <p className="employee-type-label text-[#667085]">Kỳ lương</p>
          <p className="employee-type-row-amount mt-1 text-[#101828]">
            {formatAdvancePeriodDisplay(selectedMonth)}
          </p>
        </div>
      </div>

      {!info.canRequest || !hasBankDestination ? (
        <div className="mt-3 rounded-2xl border border-amber-200 bg-amber-50 px-3.5 py-3" role="status">
          <div className="flex items-start gap-2.5">
            <Clock3 className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
            <div className="min-w-0">
              <p className="employee-type-strong text-amber-800">
                {!hasBankDestination
                  ? "Chưa có tài khoản nhận tiền"
                  : info.canRequestTitle || "Chưa thể ứng lương"}
              </p>
              <p className="employee-type-body-sm mt-1 text-amber-700">
                {!hasBankDestination
                  ? "Liên hệ quản lý để cập nhật thông tin ngân hàng trước khi ứng lương."
                  : info.canRequestReason || "Vui lòng quay lại trong kỳ ứng lương tiếp theo."}
              </p>
            </div>
          </div>
        </div>
      ) : (
        <>
          <div className="mt-4">
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
                className={`employee-type-hero-amount h-14 w-full rounded-xl border bg-white px-3.5 pr-16 text-[#101828] placeholder:text-[#98A2B3] transition-all focus:outline-none focus:ring-2 ${
                  validationError
                    ? "border-[#FDA29B] focus:ring-[#FECDCA]"
                    : "border-[#D0D5DD] focus:border-[#07883F] focus:ring-[#D1FADF]"
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
                      ? "border-[#07883F] bg-[#07883F] text-white"
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
                <p className="employee-type-inline-amount mt-1 text-[#067647] tabular-nums">
                  {feeDetails ? formatCurrency(feeDetails.netAmount) : "Đang tính..."}
                </p>
              </div>
            </div>
          )}

          <button
            type="button"
            onClick={handleSubmit}
            disabled={!canSubmit}
            className={`employee-type-action mt-4 inline-flex min-h-12 w-full items-center justify-center gap-2 rounded-xl px-5 py-2 text-white transition-all active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-45 ${canSubmit ? "bg-[#07883F] shadow-[0_8px_16px_-10px_rgba(6,118,71,0.7)] hover:bg-[#067647]" : "bg-[#98A2B3]"}`}
          >
            {isPending ? (
              <>
                <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                Đang gửi...
              </>
            ) : (
              <>
                Tiếp tục
                <ArrowRight className="h-4 w-4" />
              </>
            )}
          </button>
        </>
      )}

      {latestPendingRequest && (
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
