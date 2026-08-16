/**
 * Utility functions for Advance Payment feature
 */

import type {
  AdvancePaymentHistoryItem,
  AdvancePaymentInfo,
  AdvancePaymentQuota,
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
 * just an identifier. Use `formatPayrollMonthRange` for labels that describe
 * the employee's salary month.
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
 * Render a payroll month as its full calendar-month range.
 *
 * This is deliberately separate from `formatAdvancePeriodDisplay`: payroll
 * for July is 01/07–31/07, while the window in which that payroll can be
 * advanced is controlled independently by the backend (currently 20/07–08/08).
 */
export function formatPayrollMonthRange(monthString: string): string {
  if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(monthString)) return monthString;

  const [yearString, monthStringValue] = monthString.split("-");
  const year = Number(yearString);
  const month = Number(monthStringValue);
  const lastDay = new Date(year, month, 0).getDate();

  return `01/${monthStringValue} – ${String(lastDay).padStart(2, "0")}/${monthStringValue}/${yearString}`;
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

export interface AdvanceQuotaSummary {
  forMonth: string;
  maxAdvanceAmount: number;
  completedAmount: number;
  pendingAmount: number;
  remainingAmount: number;
  usedAmount: number;
  usedPercentage: number;
}

function sortQuotasByMonth(quotas: AdvancePaymentQuota[]): AdvancePaymentQuota[] {
  return [...quotas].sort((a, b) => a.forMonth.localeCompare(b.forMonth));
}

function hasQuotaValue(quota: AdvancePaymentQuota): boolean {
  return (
    quota.maxAdvanceAmount > 0 ||
    quota.completedAmount > 0 ||
    quota.pendingAmount > 0 ||
    quota.remainingAmount > 0
  );
}

function getHistoryUsedAmount(
  history: AdvancePaymentHistoryItem[] | undefined,
  forMonth: string,
): number {
  if (!history || history.length === 0) return 0;

  const matchingHistory = history.filter((item) => !item.forMonth || item.forMonth === forMonth);

  return matchingHistory.reduce((total, item) => {
    if (item.status !== "PENDING" && item.status !== "COMPLETED") return total;
    return total + item.requestAmount;
  }, 0);
}

export function getAdvanceQuotaSummary(
  info: AdvancePaymentInfo,
  history?: AdvancePaymentHistoryItem[],
): AdvanceQuotaSummary {
  const quotas = sortQuotasByMonth(info.quotas ?? []);
  const activeQuota =
    quotas.find((quota) => quota.forMonth === info.forMonth && hasQuotaValue(quota)) ??
    quotas.find((quota) => quota.remainingAmount > 0) ??
    [...quotas].reverse().find(hasQuotaValue) ??
    quotas.find((quota) => quota.forMonth === info.forMonth) ??
    quotas[0];

  const forMonth = activeQuota?.forMonth ?? info.forMonth;
  const rawMaxAdvanceAmount = activeQuota?.maxAdvanceAmount ?? info.maxAdvanceAmount;
  const quotaCompleted = activeQuota?.completedAmount ?? info.completedAmount;
  const quotaPending = activeQuota?.pendingAmount ?? info.pendingAmount;
  // When the API provides an explicit monthly quota, trust its usage fields
  // over legacy all-time history rows that have no payroll-month tag.
  const scopedHistory = activeQuota
    ? history?.filter((item) => item.forMonth === forMonth)
    : history;
  const historyUsed = getHistoryUsedAmount(scopedHistory, forMonth);
  const usedAmount = Math.max(quotaCompleted + quotaPending, historyUsed);
  const remainingAmount =
    activeQuota?.remainingAmount ??
    Math.max(0, info.remainingAmount || rawMaxAdvanceAmount - usedAmount);
  const maxAdvanceAmount = Math.max(rawMaxAdvanceAmount, usedAmount + remainingAmount);
  const usedPercentage =
    maxAdvanceAmount > 0
      ? Math.min(100, Math.round((usedAmount / maxAdvanceAmount) * 100))
      : 0;

  return {
    forMonth,
    maxAdvanceAmount,
    completedAmount: Math.max(quotaCompleted, historyUsed - quotaPending),
    pendingAmount: quotaPending,
    remainingAmount,
    usedAmount,
    usedPercentage,
  };
}

/**
 * Return quota values for one explicitly viewed payroll month without falling
 * back to another month. Month navigation uses this path so an empty July
 * payroll cannot accidentally display June's remaining or exhausted quota.
 */
export function getAdvanceQuotaSummaryForMonth(
  info: AdvancePaymentInfo,
  forMonth: string,
  history?: AdvancePaymentHistoryItem[],
): AdvanceQuotaSummary {
  const hasMonthSpecificQuotas = Array.isArray(info.quotas);
  const quotas = hasMonthSpecificQuotas ? info.quotas : [];
  const exactQuota = quotas.find((quota) => quota.forMonth === forMonth);
  // The top-level values are a backward-compatible fallback only. Once the
  // API supplies the quota list, even when it is empty, that list is
  // authoritative: an empty July must not inherit totals from a closed cycle.
  const usesTopLevel = !hasMonthSpecificQuotas && info.forMonth === forMonth;
  const rawMaxAdvanceAmount = exactQuota?.maxAdvanceAmount ?? (usesTopLevel ? info.maxAdvanceAmount : 0);
  const quotaCompleted = exactQuota?.completedAmount ?? (usesTopLevel ? info.completedAmount : 0);
  const quotaPending = exactQuota?.pendingAmount ?? (usesTopLevel ? info.pendingAmount : 0);
  // Older history responses have no `forMonth`. They may supplement the
  // backward-compatible top-level response, but an explicit monthly quota is
  // authoritative and must never be enlarged by all-time untagged history.
  const acceptsLegacyHistory = usesTopLevel;
  const matchingHistory = history?.filter(
    (item) => item.forMonth === forMonth || (!item.forMonth && acceptsLegacyHistory),
  );
  const historyUsed = getHistoryUsedAmount(matchingHistory, forMonth);
  const usedAmount = Math.max(quotaCompleted + quotaPending, historyUsed);
  const remainingAmount =
    exactQuota?.remainingAmount ??
    (usesTopLevel ? Math.max(0, info.remainingAmount || rawMaxAdvanceAmount - usedAmount) : 0);
  const maxAdvanceAmount = Math.max(rawMaxAdvanceAmount, usedAmount + remainingAmount);

  return {
    forMonth,
    maxAdvanceAmount,
    completedAmount: Math.max(quotaCompleted, historyUsed - quotaPending),
    pendingAmount: quotaPending,
    remainingAmount,
    usedAmount,
    usedPercentage:
      maxAdvanceAmount > 0
        ? Math.min(100, Math.round((usedAmount / maxAdvanceAmount) * 100))
        : 0,
  };
}

/**
 * Format month string to short Vietnamese format
 * Example: "2026-03" → "03/2026"
 */
export function formatMonthShort(monthString: string): string {
  if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(monthString)) return monthString;
  const [year, month] = monthString.split("-");
  return `${month}/${year}`;
}

/**
 * Whether a selected payroll month is older than the active advance period.
 *
 * The active advance period can be the previous calendar month through the
 * request cutoff. Calendar-month comparisons would incorrectly close that
 * still-requestable payroll month in the employee portal.
 */
export function isPastAdvancePaymentPeriod(viewMonth: string, activeMonth: string): boolean {
  return viewMonth < activeMonth;
}

/**
 * Select the API-provided active payroll period when a non-check-in employee
 * first opens FlexiblePay without choosing a month. Before the cutoff that
 * period can be the prior calendar month, so the calendar default is wrong.
 */
export function getInitialEmployeeAdvanceMonth(
  activeMonth: string | undefined,
  isCheckIn: boolean,
  hasExplicitMonth: boolean,
): string | undefined {
  if (!activeMonth || isCheckIn || hasExplicitMonth) return undefined;
  return activeMonth;
}

/**
 * Day-of-month from which a freshly-created advance request lands in the
 * CURRENT calendar month — also the day the self-check-in advance window opens.
 *
 * Mirrors `SelfCheckInAdvanceWindowOpenDay` in
 * `backend/internal/app/services/advance_payment/checkin_advance.go`. If the
 * backend constant changes, update this one too.
 */
export const ADVANCE_REQUEST_WINDOW_OPEN_DAY = 10;

/**
 * Default month (YYYY-MM) to open the admin advance-payments list on.
 *
 * From `ADVANCE_REQUEST_WINDOW_OPEN_DAY` onward, a new self-check-in request
 * lands in the current calendar month (the request window opens on day 10);
 * before that, new non-check-in requests land in the previous month (tail of
 * the previous advance period). Defaulting the admin list to that same month
 * keeps freshly-created requests visible without manually switching the
 * selector.
 *
 * `now` is injectable for testing.
 */
export function getDefaultAdvanceMonth(now: Date = new Date()): string {
  const monthDate =
    now.getDate() < ADVANCE_REQUEST_WINDOW_OPEN_DAY
      ? new Date(now.getFullYear(), now.getMonth() - 1, 1)
      : now;
  return `${monthDate.getFullYear()}-${String(monthDate.getMonth() + 1).padStart(2, '0')}`;
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
    return `Số tiền tối thiểu là ${minAmount.toLocaleString('vi-VN')} ₫`;
  }
  if (amount % 5000 !== 0) {
    return `Số tiền phải là bội số của 5.000 ₫`;
  }
  if (amount > remainingAmount) {
    return `Số tiền không được vượt quá ${remainingAmount.toLocaleString('vi-VN')} ₫`;
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
