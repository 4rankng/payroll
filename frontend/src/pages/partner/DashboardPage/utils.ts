import { generateMonthOptions } from '@/utils/dateHelpers';
import { formatCurrency } from '@/utils/formatters';

const ALL_VALUE = 'all';

export const monthOptions = [
  { value: ALL_VALUE, label: 'Tất cả' },
  ...generateMonthOptions(12).map((o) => ({
    value: o.value,
    label: `T${o.value.split('-')[1]}/${o.value.split('-')[0]}`,
  })),
];

export const formatVND = formatCurrency;
