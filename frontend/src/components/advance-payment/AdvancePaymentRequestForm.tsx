import { useState, useCallback, useMemo, useEffect } from "react";
import { ArrowRight, AlertCircle } from "lucide-react";
import { Slider } from "@/components/ui/slider";
import { formatCurrency } from "@/utils/formatters";
import { formatAdvancePeriodDisplay } from "@/utils/advancePaymentHelpers";
import { EMPLOYEE_BRAND_COLOR } from "@/constants/branding";
import { ADVANCE_PAYMENT_CONSTANTS } from "@/types/api/advance-payment.types";
import type { AdvancePaymentInfo } from "@/types/api/advance-payment.types";

/** Keep EMPLOYEE_BRAND_COLOR for the slider CSS custom property — cannot be expressed as a Tailwind class. */
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

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
  const [selectedMonth, setSelectedMonth] = useState<string>(defaultMonth);

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
      <div className="mb-3 flex items-center gap-2">
        <span
          className="text-lg font-bold text-employee"
        >
          $
        </span>
        <h2 className="text-lg font-bold leading-6 text-gray-900">
          Ứng lương
        </h2>
      </div>
      
      {availableQuotas.length > 0 && (
        <div className="mb-4">
          <label className="mb-1.5 block text-sm font-semibold text-gray-600">Chọn kỳ lương</label>
          <Select value={selectedMonth} onValueChange={setSelectedMonth}>
            <SelectTrigger className="h-12 w-full border-gray-200 bg-gray-50 text-base">
              <SelectValue placeholder="Chọn kỳ" />
            </SelectTrigger>
            <SelectContent>
              {sortedQuotas.map(q => (
                <SelectItem key={q.forMonth} value={q.forMonth} disabled={q.remainingAmount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT}>
                  Kỳ {formatAdvancePeriodDisplay(q.forMonth)} - Còn ứng được: {formatCurrency(q.remainingAmount)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      <div className="flex items-center gap-3 mb-3">
        <div className="relative shrink-0 w-36">
          <input
            type="text"
            inputMode="numeric"
            placeholder="0"
            value={formatAmountInput(amount)}
            onChange={handleAmountChange}
            className={`h-12 w-full rounded-xl border bg-gray-50 px-3 pr-12 text-lg font-bold transition-all focus:bg-white focus:outline-none focus:ring-2 ${
              (error || validationError)
                ? "border-red-300 focus:ring-red-100"
                : "border-gray-200 focus:border-[#00B14F] focus:ring-green-100"
            }`}
          />
          <span className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded bg-gray-100 px-1.5 py-0.5 text-xs font-semibold text-gray-500">
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
            <div className="mt-1 flex justify-between text-sm text-gray-500">
              <span>0</span>
              <span>{formatCurrency(selectedQuotaRemaining)}</span>
            </div>
          </div>
        )}
      </div>

      {(error || validationError) && (
        <p className="mb-2 flex items-center gap-1 text-sm leading-5 text-red-500">
          <AlertCircle className="h-3 w-3 shrink-0" />
          {error || validationError}
        </p>
      )}

      {feeDetails &&
        numericAmount >= ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT &&
        !validationError && (
          <div className="mb-3 space-y-1.5 rounded-xl border border-gray-100 bg-gray-50 px-3.5 py-3 text-sm">
            <div className="flex justify-between text-gray-500">
              <span>Số tiền muốn ứng</span>
              <span className="font-semibold text-gray-700">
                {formatCurrency(numericAmount)}
              </span>
            </div>
            <div className="flex justify-between text-gray-500">
              <span>Phí chuyển tiền</span>
              <span className="font-semibold text-red-500">
                −{formatCurrency(feeDetails.fee)}
              </span>
            </div>
            <div className="flex justify-between pt-1.5 border-t border-gray-200">
              <span className="font-semibold text-gray-700">Bạn nhận được</span>
              <span
                className="font-bold text-employee"
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
          className={`inline-flex min-h-12 items-center gap-1.5 rounded-xl px-5 py-2 text-base font-bold text-white transition-all active:scale-[0.97] disabled:cursor-not-allowed disabled:opacity-40 ${canSubmit ? 'bg-employee' : 'bg-gray-400'}`}
        >
          {isPending ? (
            <>
              <span className="animate-spin h-3.5 w-3.5 border-2 border-white/30 border-t-white rounded-full" />
              Đang gửi...
            </>
          ) : (
            <>
              Gửi yêu cầu ứng
              <ArrowRight className="h-3.5 w-3.5" />
            </>
          )}
        </button>
      </div>
    </div>
  );
}
