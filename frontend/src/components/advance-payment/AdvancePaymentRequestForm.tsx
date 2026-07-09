import { useState, useCallback, useMemo, useEffect } from "react";
import { ArrowRight, AlertCircle, DollarSign } from "lucide-react";
import { Slider } from "@/components/ui/slider";
import { formatCurrency } from "@/utils/formatters";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import { EmployeeIconFrame } from "@/components/employees/EmployeeIconFrame";
import type { AdvancePaymentInfo } from "@/types/api/advance-payment.types";

interface AdvancePaymentRequestFormProps {
  info: AdvancePaymentInfo;
  feeDetails: { fee: number; netAmount: number } | null;
  onSubmit: (data: { amount: number; forMonth: string }) => void;
  onAmountChange?: (amount: number) => void;
  isPending: boolean;
  className?: string;
  style?: React.CSSProperties;
}

export function AdvancePaymentRequestForm({
  info,
  feeDetails,
  onSubmit,
  onAmountChange,
  isPending,
  className,
  style,
}: AdvancePaymentRequestFormProps) {
  const [amount, setAmount] = useState("");
  const [sliderValue, setSliderValue] = useState(0);

  // Default to the oldest available month with quota
  const availableQuotas = useMemo(() => info.quotas || [], [info.quotas]);
  const sortedQuotas = useMemo(
    () => [...availableQuotas].sort((a, b) => a.forMonth.localeCompare(b.forMonth)),
    [availableQuotas]
  );
  const defaultMonth = sortedQuotas.length > 0 ? sortedQuotas[0].forMonth : info.forMonth;
  const selectedMonth = defaultMonth;

  const selectedQuotaRemaining = useMemo(() => {
    if (availableQuotas.length > 0) {
      const q = availableQuotas.find(q => q.forMonth === selectedMonth);
      return q ? q.remainingAmount : 0;
    }
    return info.remainingAmount;
  }, [availableQuotas, selectedMonth, info.remainingAmount]);

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
      const num = value ? parseInt(value, 10) : 0;
      setAmount(value);
      setSliderValue(num);
    },
    []
  );

  const setAmountFromNumber = useCallback((value: number) => {
    const rounded = Math.max(0, Math.round(value / 5000) * 5000);
    setAmount(rounded ? rounded.toString() : "");
    setSliderValue(rounded);
  }, []);

  const handleSliderChange = useCallback((values: number[]) => {
    const rounded = Math.round((values[0] || 0) / 10000) * 10000;
    setAmountFromNumber(rounded);
  }, [setAmountFromNumber]);

  const formatAmountInput = useCallback(
    (v: string) => (!v ? "" : parseInt(v, 10).toLocaleString("vi-VN")),
    []
  );

  const quickAmounts = useMemo(() => {
    const maxValidAmount = Math.floor(selectedQuotaRemaining / 5000) * 5000;
    if (maxValidAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return [];
    return [0.25, 0.5, 0.75, 1].map((ratio) => {
      const rawAmount = maxValidAmount * ratio;
      const roundedAmount = Math.max(
        ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT,
        Math.round(rawAmount / 5000) * 5000
      );
      return {
        label: `${Math.round(ratio * 100)}%`,
        amount: Math.min(roundedAmount, maxValidAmount),
      };
    }).filter((item, index, arr) => arr.findIndex((candidate) => candidate.amount === item.amount) === index);
  }, [selectedQuotaRemaining]);

  const handleSubmit = useCallback(() => {
    if (numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return;
    if (validationError) return;
    if (!feeDetails) return;
    onSubmit({ amount: numericAmount, forMonth: selectedMonth });
  }, [feeDetails, numericAmount, selectedMonth, onSubmit, validationError]);

  return (
    <div
      className={className ?? "bg-white rounded-[24px] p-4"}
      style={style}
    >
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <p className="text-[12px] font-semibold uppercase leading-4 tracking-wide text-slate-500">
            Tạo yêu cầu
          </p>
          <h2 className="mt-1 text-[22px] font-extrabold leading-7 tracking-normal text-slate-950">
            Nhận lương sớm
          </h2>
        </div>
        <EmployeeIconFrame icon={DollarSign} />
      </div>

      <div className="border-t border-slate-100 pt-4">
        <label htmlFor="advance-payment-amount" className="block text-[13px] font-semibold leading-5 text-slate-500">
          Số tiền muốn ứng
        </label>
        <div className="relative mt-2">
          <input
            id="advance-payment-amount"
            type="text"
            inputMode="numeric"
            aria-label="Số tiền muốn ứng"
            placeholder="0"
            value={formatAmountInput(amount)}
            onChange={handleAmountChange}
            className={`h-16 w-full rounded-2xl border bg-white px-3 pr-14 text-[32px] font-extrabold leading-none tracking-normal text-slate-950 transition-all focus:outline-none focus:ring-2 ${
              validationError
                ? "border-red-300 focus:ring-red-100"
                : "border-slate-200 focus:border-employee focus:ring-green-100"
            }`}
          />
          <span className="absolute right-3 top-1/2 -translate-y-1/2 rounded-lg bg-slate-100 px-2 py-1 text-[13px] font-bold text-slate-500">
            VND
          </span>
        </div>

        {quickAmounts.length > 0 && (
          <div className="mt-3 grid grid-cols-2 gap-2">
            {quickAmounts.map((quickAmount) => (
              <button
                key={`${quickAmount.label}-${quickAmount.amount}`}
                type="button"
                onClick={() => setAmountFromNumber(quickAmount.amount)}
                className={`min-h-14 rounded-full border px-4 py-2 text-left transition-all active:scale-[0.98] ${
                  numericAmount === quickAmount.amount
                    ? "border-employee bg-employee text-white"
                    : "border-slate-200 bg-white text-slate-700"
                }`}
              >
                <span className="block text-[12px] font-bold leading-4 opacity-80">
                  {quickAmount.label} hạn mức
                </span>
                <span className="mt-1 block truncate text-[15px] font-extrabold leading-5 tabular-nums">
                  {formatCurrency(quickAmount.amount)}
                </span>
              </button>
            ))}
          </div>
        )}

        {selectedQuotaRemaining > 0 && (
          <div className="mt-4">
            <div className="mb-2 flex items-center justify-between gap-3">
              <span className="text-[13px] font-semibold leading-5 text-slate-500">Kéo để chọn nhanh</span>
              <span className="text-[13px] font-bold leading-5 text-employee tabular-nums">
                Tối đa {formatCurrency(selectedQuotaRemaining)}
              </span>
            </div>
            <Slider
              value={[sliderValue]}
              onValueChange={handleSliderChange}
              max={selectedQuotaRemaining}
              step={10000}
              className="w-full [&_[role=slider]]:h-4 [&_[role=slider]]:w-4 [&_[role=slider]]:border-2 [&_[role=slider]]:border-white [&_[role=slider]]:shadow-md [&>span:first-child]:h-1.5 [&>span:first-child]:rounded-full [&>span:first-child]:bg-gray-200"
              style={
                { "--slider-thumb-color": EMPLOYEE_BRAND_COLOR } as React.CSSProperties
              }
            />
            <div className="mt-2 flex justify-between text-[14px] text-slate-500">
              <span>0</span>
              <span>Tối đa</span>
            </div>
          </div>
        )}
      </div>

      {validationError && (
        <p className="mt-3 flex items-center gap-1.5 rounded-2xl bg-red-50 px-3 py-2 text-[14px] font-semibold leading-6 text-red-600">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {validationError}
        </p>
      )}

      {canShowFeePreview && (
        <div className="mt-4 space-y-2 border-t border-slate-100 pt-4 text-[15px]">
          <div className="flex justify-between gap-3 text-slate-500">
            <span>Số tiền yêu cầu</span>
            <span className="font-bold text-slate-900 tabular-nums">
              {formatCurrency(numericAmount)}
            </span>
          </div>
          <div className="flex justify-between gap-3 text-slate-500">
            <span>Phí chuyển tiền</span>
            <span className="font-bold text-red-500 tabular-nums">
              {feeDetails ? `−${formatCurrency(feeDetails.fee)}` : "Đang tính..."}
            </span>
          </div>
          <div className="flex justify-between gap-3 border-t border-slate-200 pt-2">
            <span className="font-extrabold text-slate-700">Bạn nhận được</span>
            <span className="text-[22px] font-extrabold leading-7 text-employee tabular-nums">
              {feeDetails ? formatCurrency(feeDetails.netAmount) : "Đang tính..."}
            </span>
          </div>
        </div>
      )}

      <div className="sticky z-20 mt-4 pt-2" style={{ bottom: "calc(env(safe-area-inset-bottom, 0px) + 0.75rem)" }}>
        <button
          onClick={handleSubmit}
          disabled={!canSubmit}
          className={`inline-flex min-h-14 w-full items-center justify-center gap-2 rounded-2xl px-5 py-2 text-[17px] font-extrabold text-white transition-all active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-45 ${canSubmit ? 'bg-employee shadow-[0_14px_30px_-18px_rgba(0,177,79,0.9)]' : 'bg-gray-400'}`}
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
      </div>
    </div>
  );
}
