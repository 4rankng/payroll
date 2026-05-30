import type { TimesheetEntry } from '@/components/sheets/timesheet-entry/types/multi-timesheet.types';

export interface AmountDisplayOptions {
  calculatedAmount?: number | null;
  hoursWorked?: number | null;
  position?: string | null;
  hourType?: string | null;
  dayType?: string | null;
  calculatedRate?: number | null;
  employeeId?: number | null;
}

export interface AmountDisplayResult {
  text: string;
  formattedAmount?: string;
  formattedRate?: string;
  shouldDisplay: boolean;
}

export function calculateAmountDisplay(
  options: AmountDisplayOptions,
  locale: string = 'vi-VN'
): AmountDisplayResult {
  const { calculatedAmount, hoursWorked, position, hourType, dayType, calculatedRate, employeeId } = options;

  // Don't display if no employee is selected
  if (!employeeId) {
    return { text: '', shouldDisplay: false };
  }

  // If we have a calculated amount, format and return it
  if (calculatedAmount && calculatedAmount > 0) {
    const formattedAmount = calculatedAmount.toLocaleString(locale) + ' đ';
    const result: AmountDisplayResult = { text: formattedAmount, formattedAmount, shouldDisplay: true };

    if (calculatedRate && calculatedRate > 0) {
      result.formattedRate = `@${calculatedRate.toLocaleString(locale)} đ/giờ`;
    }

    return result;
  }

  // If we have hours worked, check if we have all the necessary information to calculate
  if (hoursWorked && hoursWorked > 0) {
    // If position, hourType, and dayType are all available, we're calculating
    if (position && hourType && dayType) {
      return { text: 'Đang tính toán...', shouldDisplay: true };
    }
    
    // Otherwise, we have hours but are missing some information
    return { text: 'Thiếu thông tin', shouldDisplay: true };
  }

  // Default to zero if no hours worked but employee is selected
  return { text: '0 đ', shouldDisplay: true };
}

export function getAmountDisplayText(
  entry: TimesheetEntry,
  locale: string = 'vi-VN'
): AmountDisplayResult {
  return calculateAmountDisplay({
    calculatedAmount: entry.calculatedAmount,
    hoursWorked: entry.hoursWorked,
    position: entry.position,
    hourType: entry.hourType,
    dayType: entry.dayType,
    calculatedRate: entry.calculatedRate,
    employeeId: entry.employeeId
  }, locale);
}