export const BANK_TRANSFER_CYCLE_LABELS = {
  1: 'Kỳ 1 · ngày 1–7',
  2: 'Kỳ 2 · ngày 8–14',
  3: 'Kỳ 3 · ngày 15–21',
  4: 'Kỳ 4 · ngày 22–28',
} as const;

const bankTransferDateFormatter = new Intl.DateTimeFormat('vi-VN', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  timeZone: 'Asia/Ho_Chi_Minh',
});

const bankTransferTimeFormatter = new Intl.DateTimeFormat('vi-VN', {
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
  timeZone: 'Asia/Ho_Chi_Minh',
});

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

export function formatBankTransferDateTime(value?: string): string {
  if (!value) return '—';

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';

  return `${bankTransferDateFormatter.format(date)} · ${bankTransferTimeFormatter.format(date)}`;
}
