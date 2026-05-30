import { generateMonthOptions } from '@/utils/dateHelpers';

const ALL_VALUE = 'all';

export const monthOptions = [
  { value: ALL_VALUE, label: 'Tất cả' },
  ...generateMonthOptions(12).map((o) => ({
    value: o.value,
    label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
  })),
];

export function formatVND(value: number): string {
  const abs = Math.abs(value);
  const sign = value < 0 ? '-' : '';
  if (abs >= 1e9) return `${sign}${(abs / 1e9).toFixed(1)}tỷ đ`;
  if (abs >= 1e6) return `${sign}${(abs / 1e6).toFixed(1)}M đ`;
  if (abs >= 1e3) return `${sign}${(abs / 1e3).toFixed(0)}K đ`;
  return `${sign}${abs.toLocaleString('vi-VN')} đ`;
}
