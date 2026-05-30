import { isToday, isYesterday, isThisMonth, parseISO, format } from 'date-fns';
import { vi } from 'date-fns/locale';
import type { AdvancePaymentFileHistoryItem } from '@/types/api/advance-payment.types';

export interface DateGroup {
  label: string;
  key: string;
  files: AdvancePaymentFileHistoryItem[];
}

const GROUP_ORDER = ['today', 'yesterday'] as const;

function getGroupKey(date: Date): string {
  if (isToday(date)) return 'today';
  if (isYesterday(date)) return 'yesterday';
  if (isThisMonth(date)) return format(date, 'yyyy-MM');
  return format(date, 'yyyy-MM');
}

function getGroupLabel(key: string): string {
  if (key === 'today') return 'Hôm nay';
  if (key === 'yesterday') return 'Hôm qua';
  const date = new Date(key + '-01');
  if (isThisMonth(date)) return format(date, "'Tháng' M", { locale: vi });
  return format(date, "'Tháng' M/yyyy", { locale: vi });
}

function sortGroupKeys(a: string, b: string): number {
  if (a === 'today') return -1;
  if (b === 'today') return 1;
  if (a === 'yesterday') return -1;
  if (b === 'yesterday') return 1;
  return b.localeCompare(a);
}

export function groupFilesByDate(files: AdvancePaymentFileHistoryItem[]): DateGroup[] {
  const sorted = [...files].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  const buckets = new Map<string, AdvancePaymentFileHistoryItem[]>();

  for (const file of sorted) {
    const date = parseISO(file.createdAt);
    const key = getGroupKey(date);
    if (!buckets.has(key)) buckets.set(key, []);
    buckets.get(key)!.push(file);
  }

  const groups: DateGroup[] = [];
  const sortedKeys = [...buckets.keys()].sort(sortGroupKeys);

  for (const key of sortedKeys) {
    groups.push({
      label: getGroupLabel(key),
      key,
      files: buckets.get(key)!,
    });
  }

  return groups;
}
