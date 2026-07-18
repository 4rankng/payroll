export const BANK_TRANSFER_CYCLE_LABELS = {
  1: 'Kỳ 1 · ngày 1–7',
  2: 'Kỳ 2 · ngày 8–14',
  3: 'Kỳ 3 · ngày 15–21',
  4: 'Kỳ 4 · ngày 22–28',
} as const;

export function getCurrentMonthValue(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
}

export function formatBankTransferDate(value: string): string {
  const [year, month, day] = value.split('-');
  return `${day}/${month}/${year}`;
}

export function formatBankTransferPeriod(fromDate: string, toDate: string): string {
  const [fromYear, fromMonth, fromDay] = fromDate.split('-');
  const [toYear, toMonth, toDay] = toDate.split('-');

  if (fromYear === toYear && fromMonth === toMonth) {
    return `${fromDay}–${toDay}/${toMonth}/${toYear}`;
  }

  return `${formatBankTransferDate(fromDate)}–${formatBankTransferDate(toDate)}`;
}
