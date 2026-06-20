/**
 * Utility functions for Advance Payment feature
 */

import type {
  AdvancePaymentStatus,
  AdvancePaymentRequestStatus,
  RequestMethod,
} from '@/types/api/advance-payment.types';
import {
  ADVANCE_PAYMENT_STATUS_LABELS,
  ADVANCE_PAYMENT_REQUEST_STATUS_LABELS,
  REQUEST_METHOD_LABELS,
  ADVANCE_PAYMENT_CONSTANTS,
} from '@/types/api/advance-payment.types';

/**
 * Convert advance payment status to Vietnamese
 */
export function getVietnameseAdvancePaymentStatus(status: AdvancePaymentStatus): string {
  return ADVANCE_PAYMENT_STATUS_LABELS[status] || status;
}

/**
 * Convert advance payment request status to Vietnamese
 */
export function getVietnameseAdvancePaymentRequestStatus(status: AdvancePaymentRequestStatus): string {
  return ADVANCE_PAYMENT_REQUEST_STATUS_LABELS[status] || status;
}

/**
 * Get color classes for advance payment status badge
 */
export function getAdvancePaymentStatusColor(status: AdvancePaymentStatus | AdvancePaymentRequestStatus): string {
  const normalizedStatus = status.toUpperCase() as AdvancePaymentStatus;
  const colorMap: Record<AdvancePaymentStatus, string> = {
    PENDING: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    APPROVED: 'bg-blue-100 text-blue-800 border-blue-200',
    COMPLETED: 'bg-green-100 text-green-800 border-green-200',
    FAILED: 'bg-red-100 text-red-800 border-red-200',
    CANCELLED: 'bg-gray-100 text-gray-800 border-gray-200',
  };
  return colorMap[normalizedStatus] || 'bg-gray-100 text-gray-800 border-gray-200';
}

/**
 * Convert request method to Vietnamese
 */
export function getVietnameseRequestMethod(method: RequestMethod): string {
  return REQUEST_METHOD_LABELS[method] || method;
}

/**
 * Calculate fee based on amount
 * Fee = MAX(10,000 VND, 2% * amount)
 */
export function calculateAdvancePaymentFee(amount: number): number {
  const percentageFee = Math.round(amount * (ADVANCE_PAYMENT_CONSTANTS.FEE_PERCENTAGE / 100));
  return Math.max(ADVANCE_PAYMENT_CONSTANTS.MIN_FEE, percentageFee);
}

/**
 * Calculate net amount (amount - fee)
 */
export function calculateNetAmount(amount: number): number {
  return amount - calculateAdvancePaymentFee(amount);
}

/**
 * Calculate fee with net amount in one call
 */
export function calculateAdvancePaymentDetails(amount: number): { fee: number; netAmount: number } {
  const fee = calculateAdvancePaymentFee(amount);
  return {
    fee,
    netAmount: amount - fee,
  };
}

/**
 * Format month string (YYYY-MM) to Vietnamese display format
 * Example: "2026-03" → "Tháng 3/2026"
 *
 * Use this for option labels in month-picker dropdowns where the value is
 * just an identifier. For headers/pills that describe the active advance
 * payment period, prefer `formatAdvancePeriodDisplay` — it shows the actual
 * cycle dates so users don't read the anchor month as a calendar month.
 */
export function formatMonthDisplay(monthString: string): string {
  if (!monthString || monthString.length !== 7) return monthString;

  const [year, month] = monthString.split('-');
  const monthNum = parseInt(month, 10);

  if (isNaN(monthNum) || monthNum < 1 || monthNum > 12) {
    return monthString;
  }

  return `Tháng ${monthNum}/${year}`;
}

/**
 * Day-of-month when the advance payment period rolls over.
 *
 * Mirrors `PeriodCycleStartDay` in
 * `backend/internal/app/services/advance_payment/month_resolver.go`. If the
 * backend constant changes, update this one too.
 */
export const ADVANCE_PERIOD_CYCLE_START_DAY = 20;

/**
 * Render an advance-payment period anchor (YYYY-MM) as the explicit cycle
 * range — i.e. "20/04 – 19/05/2026" instead of the bare anchor "Tháng
 * 4/2026". The backend's `forMonth` is the period START month, so on, e.g.,
 * May 9 it returns "2026-04" even though today is in May. Showing the
 * range removes that "but it's already May!" confusion.
 *
 * Example: "2026-04" → "20/04 – 19/05/2026"
 *          "2026-12" → "20/12/2026 – 19/01/2027" (period crosses year)
 */
export function formatAdvancePeriodDisplay(monthString: string): string {
  if (!monthString || monthString.length !== 7) return monthString;

  const [yearStr, monthStr] = monthString.split('-');
  const year = parseInt(yearStr, 10);
  const month = parseInt(monthStr, 10);

  if (isNaN(year) || isNaN(month) || month < 1 || month > 12) {
    return monthString;
  }

  const startDay = ADVANCE_PERIOD_CYCLE_START_DAY;
  const endDay = startDay - 1;
  const endMonth = month === 12 ? 1 : month + 1;
  const endYear = month === 12 ? year + 1 : year;

  const pad = (n: number) => n.toString().padStart(2, '0');

  if (year === endYear) {
    // Period stays within one calendar year — show the year once at the end.
    return `${pad(startDay)}/${pad(month)} – ${pad(endDay)}/${pad(endMonth)}/${year}`;
  }
  // Period crosses a year boundary (Dec → Jan) — annotate both ends.
  return `${pad(startDay)}/${pad(month)}/${year} – ${pad(endDay)}/${pad(endMonth)}/${endYear}`;
}

/**
 * Format month string to short Vietnamese format
 * Example: "2026-03" → "03/2026"
 */
export function formatMonthShort(monthString: string): string {
  if (!monthString || monthString.length !== 7) return monthString;
  return monthString.replace('-', '/');
}

/**
 * Get current "for month" based on business rule
 * Day >= 9 → current month; Day < 9 → last month
 * Returns YYYY-MM format
 */
export function getCurrentForMonth(): string {
  const now = new Date();
  const day = now.getDate();

  if (day >= 9) {
    // Current month
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  } else {
    // Last month
    const lastMonth = new Date(now.getFullYear(), now.getMonth() - 1, 1);
    return `${lastMonth.getFullYear()}-${String(lastMonth.getMonth() + 1).padStart(2, '0')}`;
  }
}

/**
 * Get previous month in YYYY-MM format
 */
export function getPreviousMonth(forMonth: string): string {
  const [year, month] = forMonth.split('-').map(Number);
  const date = new Date(year, month - 2, 1); // month - 2 because months are 0-indexed
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Get next month in YYYY-MM format
 */
export function getNextMonth(forMonth: string): string {
  const [year, month] = forMonth.split('-').map(Number);
  const date = new Date(year, month, 1); // month is 1-indexed in input, so use as-is
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}

/**
 * Validate advance payment amount
 * Returns error message or null if valid
 */
export function validateAdvancePaymentAmount(
  amount: number,
  remainingAmount: number,
  minAmount: number = ADVANCE_PAYMENT_CONSTANTS.MIN_AMOUNT
): string | null {
  if (amount < minAmount) {
    return `Số tiền tối thiểu là ${minAmount.toLocaleString('vi-VN')} đ`;
  }
  if (amount % 5000 !== 0) {
    return `Số tiền phải là bội số của 5.000 đ`;
  }
  if (amount > remainingAmount) {
    return `Số tiền không được vượt quá ${remainingAmount.toLocaleString('vi-VN')} đ`;
  }
  return null;
}

/**
 * Generate month options for dropdown (past N months)
 */
export function generateMonthOptions(count: number = 12): Array<{ value: string; label: string }> {
  const options = [];
  const now = new Date();

  for (let i = 0; i < count; i++) {
    const date = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const value = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
    options.push({
      value,
      label: formatMonthDisplay(value),
    });
  }

  return options;
}

/**
 * Check if amount qualifies for percentage-based fee (vs minimum fee)
 */
export function isPercentageFee(amount: number): boolean {
  const percentageFee = amount * (ADVANCE_PAYMENT_CONSTANTS.FEE_PERCENTAGE / 100);
  return percentageFee > ADVANCE_PAYMENT_CONSTANTS.MIN_FEE;
}

/**
 * Get fee breakdown for display
 */
export function getFeeBreakdown(amount: number): {
  percentageFee: number;
  minFee: number;
  appliedFee: number;
  isUsingMinFee: boolean;
} {
  const percentageFee = Math.round(amount * (ADVANCE_PAYMENT_CONSTANTS.FEE_PERCENTAGE / 100));
  const minFee = ADVANCE_PAYMENT_CONSTANTS.MIN_FEE;
  const isUsingMinFee = percentageFee < minFee;

  return {
    percentageFee,
    minFee,
    appliedFee: Math.max(minFee, percentageFee),
    isUsingMinFee,
  };
}
