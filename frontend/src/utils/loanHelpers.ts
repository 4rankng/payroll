import type { LoanStatus, LoanType, ScheduleStatus, Loan, CustomScheduleItem } from '@/types/api/loan.types';

/**
 * Convert loan status to Vietnamese
 */
export function getVietnameseLoanStatus(status: LoanStatus): string {
  const statusMap: Record<LoanStatus, string> = {
    active: 'Đang hoạt động',
    closed: 'Đã đóng',
  };
  return statusMap[status] || status;
}

/**
 * Get color for loan status badge
 */
export function getLoanStatusColor(status: LoanStatus): string {
  const colorMap: Record<LoanStatus, string> = {
    active: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    closed: 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300',
  };
  return colorMap[status] || 'bg-gray-100 text-gray-800';
}

/**
 * Get Vietnamese label for loan type
 */
export function getLoanTypeLabel(type: LoanType): string {
  const typeMap: Record<LoanType, string> = {
    bullet_loan: 'Theo lãi suất',
    custom_schedule: 'Theo lịch trả',
  };
  return typeMap[type] || type;
}

/**
 * Get color for loan type badge
 */
export function getLoanTypeBadgeColor(type: LoanType): string {
  const colorMap: Record<LoanType, string> = {
    bullet_loan: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300',
    custom_schedule: 'bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-300',
  };
  return colorMap[type] || 'bg-gray-100 text-gray-800';
}

/**
 * Convert basis points to percentage
 * Example: 1200 bps = 12%
 */
export function bpsToPercent(bps: number): string {
  const percent = bps / 100;
  return `${percent.toFixed(2)}%`;
}

/**
 * Convert basis points to percentage number (for calculations)
 */
export function bpsToPercentNumber(bps: number): number {
  return bps / 100;
}

/**
 * Convert percentage to basis points
 * Example: 12% = 1200 bps
 */
export function percentToBps(percent: number): number {
  return Math.round(percent * 100);
}

/**
 * Pre-configured formatter for VND so we avoid recreating Intl instances.
 */
const VND_NUMBER_FORMATTER = new Intl.NumberFormat('vi-VN', {
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

/**
 * Format currency in Vietnamese Dong using locale-aware grouping and a suffix.
 */
export function formatVND(amount: number): string {
  return `${VND_NUMBER_FORMATTER.format(amount)} đ`;
}

/**
 * Format currency without currency symbol (just number with commas)
 */
export function formatNumber(amount: number): string {
  return new Intl.NumberFormat('vi-VN').format(amount);
}

/**
 * Calculate days until a date
 */
export function daysUntil(dateString: string): number {
  const targetDate = new Date(dateString);
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  targetDate.setHours(0, 0, 0, 0);

  const diffTime = targetDate.getTime() - today.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  return diffDays;
}

/**
 * Get color indicator for upcoming payment
 */
export function getPaymentUrgencyColor(daysUntil: number): string {
  if (daysUntil < 0) return 'text-red-600'; // Overdue
  if (daysUntil <= 7) return 'text-orange-600'; // Due soon
  if (daysUntil <= 30) return 'text-yellow-600'; // Coming up
  return 'text-gray-600'; // Future
}

/**
 * Format payment day of month with Vietnamese suffix
 */
export function formatPaymentDay(day: number): string {
  return `Ngày ${day} hàng tháng`;
}

/**
 * Calculate total interest for a loan over its term
 */
export function calculateTotalInterest(
  principalAmount: number,
  interestRateBps: number,
  termMonths: number
): number {
  const monthlyInterest = (principalAmount * interestRateBps / 10000) / 12;
  return monthlyInterest * termMonths;
}

/**
 * Calculate monthly interest amount
 */
export function calculateMonthlyInterest(
  outstandingPrincipal: number,
  interestRateBps: number
): number {
  const annualInterest = (outstandingPrincipal * interestRateBps) / 10000;
  return annualInterest / 12;
}

/**
 * Get Vietnamese label for schedule item type
 */
export function getScheduleItemTypeLabel(type: 'interest' | 'principal'): string {
  const labels = {
    interest: 'Lãi suất',
    principal: 'Nợ gốc',
  };
  return labels[type];
}

/**
 * Get Vietnamese label for schedule item status
 */
export function getScheduleItemStatusLabel(status: ScheduleStatus): string {
  const labels: Record<ScheduleStatus, string> = {
    paid: 'Đã thanh toán',
    pending: 'Chưa thanh toán',
    overdue: 'Quá hạn',
  };
  return labels[status];
}

/**
 * Get color for schedule item status
 */
export function getScheduleItemStatusColor(status: ScheduleStatus): string {
  const colors: Record<ScheduleStatus, string> = {
    paid: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
    overdue: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300',
  };
  return colors[status];
}

export type NextPaymentInfo = {
  dueDate: string | null;
  amount: number | null;
  status: ScheduleStatus | null;
  period: number | null;
};

/**
 * Get disbursement date (fall back to planned start date if not yet disbursed)
 */
export function getDisbursementDate(loan: Loan): string | null {
  if (!loan) return null;
  return loan.disbursement_date ?? (loan as { disbursed_at?: string }).disbursed_at ?? (loan as { start_date?: string }).start_date ?? null;
}

/**
 * Get the earliest upcoming payment (pending/overdue preferred, otherwise earliest schedule)
 */
export function getNextPayment(loan: Loan): NextPaymentInfo {
  if (!loan) {
    return { dueDate: null, amount: null, status: null, period: null };
  }

  // Prefer backend-provided next payment fields if present
  if (loan.next_payment_date) {
    return {
      dueDate: loan.next_payment_date,
      amount: loan.next_payment_amount ?? null,
      status: null,
      period: null,
    };
  }

  const schedules: CustomScheduleItem[] = Array.isArray(loan?.schedules) ? loan.schedules : [];

  if (!schedules.length) {
    return { dueDate: null, amount: null, status: null, period: null };
  }

  const sortedSchedules = [...schedules].sort(
    (a, b) => new Date(a.due_date).getTime() - new Date(b.due_date).getTime()
  );

  const prioritized = sortedSchedules.filter((item) => item.status === 'pending' || item.status === 'overdue');
  const target = prioritized[0] ?? sortedSchedules[0];
  return {
    dueDate: target?.due_date ?? null,
    amount: target?.amount ?? null,
    status: target?.status ?? null,
    period: target?.period ?? null,
  };
}
