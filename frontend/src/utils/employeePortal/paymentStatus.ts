export type DayPaymentStatus = 'full' | 'partial' | 'none';

/**
 * Determine payment status for a day based on payable vs paid amounts.
 * Fully paid when payable ~= paid (within small epsilon to handle floats).
 */
export function getDayPaymentStatus(
  totalAmount: number,
  totalPaidAmount: number,
  bulkTransferPercentage: number
): DayPaymentStatus {
  const payable = totalAmount * bulkTransferPercentage;
  const epsilon = 1; // treat values within 1 VND as equal
  if (Math.abs(payable - totalPaidAmount) <= epsilon) return 'full';
  if (totalPaidAmount > 0 && totalPaidAmount < payable) return 'partial';
  return 'none';
}

