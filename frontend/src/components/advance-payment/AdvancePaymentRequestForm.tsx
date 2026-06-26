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
  const [error, setError] = useState<string | null>(null);
  
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

  const validationError = useMemo(() => {
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
  }, [selectedQuotaRemaining, numericAmount, feeDetails, providerMin, info.providerMaxTransferAmount]);

  const canSubmit =
    !!numericAmount &&
    numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT &&
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
      setError(null);
    },
    []
  );

  const handleSliderChange = useCallback((values: number[]) => {
    const rounded = Math.round((values[0] || 0) / 10000) * 10000;
    setAmount(rounded.toString());
    setSliderValue(rounded);
    setError(null);
  }, []);

  const formatAmountInput = useCallback(
    (v: string) => (!v ? "" : parseInt(v, 10).toLocaleString("vi-VN")),
    []
  );

  const handleSubmit = useCallback(() => {
    if (numericAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return;
    if (validationError) return;
    onSubmit({ amount: numericAmount, forMonth: selectedMonth });
  }, [numericAmount, selectedMonth, onSubmit, validationError]);

  return (
    <div
      className={className ?? "bg-white rounded-2xl p-4"}
      style={style}
    >
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <p className="text-[12px] font-semibold uppercase leading-4 tracking-wide text-slate-500">
            Tạo yêu cầu
          </p>
          <h2 className="mt-1 text-[18px] font-bold leading-7 text-slate-950">
            Ứng lương
          </h2>
        </div>
        <EmployeeIconFrame icon={DollarSign} />
      </div>
      
      <div className="mb-3 flex flex-col gap-3 xs:flex-row xs:items-center">
        <div className="relative w-full shrink-0 xs:w-40">
          <input
            type="text"
            inputMode="numeric"
            aria-label="Số tiền muốn ứng"
            placeholder="0"
            value={formatAmountInput(amount)}
            onChange={handleAmountChange}
            className={`h-14 w-full rounded-xl border bg-slate-50 px-3 pr-12 text-[24px] font-bold leading-none text-slate-950 transition-all focus:bg-white focus:outline-none focus:ring-2 ${
              (error || validationError)
                ? "border-red-300 focus:ring-red-100"
                : "border-gray-200 focus:border-employee focus:ring-green-100"
            }`}
          />
          <span className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded bg-gray-100 px-2 py-1 text-sm font-semibold text-gray-500">
            VND
          </span>
        </div>

        {selectedQuotaRemaining > 0 && (
          <div className="flex-1 min-w-0">
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

      {(error || validationError) && (
        <p className="mb-2 flex items-center gap-1 text-[14px] leading-6 text-red-500">
          <AlertCircle className="h-3 w-3 shrink-0" />
          {error || validationError}
        </p>
      )}

      {feeDetails &&
        numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT &&
        !validationError && (
          <div className="mb-3 space-y-2 rounded-xl border border-slate-100 bg-slate-50 px-3.5 py-3 text-[15px]">
            <div className="flex justify-between gap-3 text-slate-500">
              <span>Phí chuyển tiền</span>
              <span className="text-[16px] font-semibold text-red-500 tabular-nums">
                −{formatCurrency(feeDetails.fee)}
              </span>
            </div>
            <div className="flex justify-between gap-3 border-t border-slate-200 pt-2">
              <span className="font-semibold text-slate-700">Bạn nhận được</span>
              <span
                className="text-[17px] font-bold text-employee tabular-nums"
              >
                {formatCurrency(feeDetails.netAmount)}
              </span>
            </div>
          </div>
        )}

      <div className="flex justify-end">
        <button
          onClick={handleSubmit}
          disabled={!canSubmit}
          className={`inline-flex min-h-14 w-full items-center justify-center gap-2 rounded-xl px-5 py-2 text-[17px] font-bold text-white transition-all active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-40 xs:w-auto ${canSubmit ? 'bg-employee' : 'bg-gray-400'}`}
        >
          {isPending ? (
            <>
              <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              Đang gửi...
            </>
          ) : (
            <>
              Gửi yêu cầu ứng
              <ArrowRight className="h-4 w-4" />
            </>
          )}
        </button>
      </div>
    </div>
  );
}
