import { useCallback, useEffect, useRef, useState } from 'react';
import { ADVANCE_PAYMENT_CONSTANTS, type CalculateFeeRequest, type CalculateFeeResponse } from '@/types/api/advance-payment.types';

export function useAdvanceFeePreview(calculateFee: (request: CalculateFeeRequest) => Promise<CalculateFeeResponse>) {
  const [feeDetails, setFeeDetails] = useState<CalculateFeeResponse | null>(null);
  const [feeError, setFeeError] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const requestRef = useRef(0);
  const amountRef = useRef(0);

  const calculate = useCallback((amount: number) => {
    const request = ++requestRef.current;
    amountRef.current = amount;
    if (timerRef.current) clearTimeout(timerRef.current);
    // The previous amount's preview must never enable this amount's request.
    setFeeDetails(null);
    setFeeError(false);
    if (amount < ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT) return;

    timerRef.current = setTimeout(async () => {
      try {
        const result = await calculateFee({ amount });
        if (request === requestRef.current) setFeeDetails(result);
      } catch {
        if (request === requestRef.current) setFeeError(true);
      }
    }, 300);
  }, [calculateFee]);

  const retry = useCallback(() => calculate(amountRef.current), [calculate]);

  useEffect(() => () => {
    ++requestRef.current;
    if (timerRef.current) clearTimeout(timerRef.current);
  }, []);

  return { feeDetails, feeError, calculate, retry };
}
