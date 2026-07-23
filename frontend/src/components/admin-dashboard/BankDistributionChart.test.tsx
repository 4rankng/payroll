import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { BankDistributionChart } from './BankDistributionChart';
import type { BankDistributionItem } from './bank-distribution';

const items: BankDistributionItem[] = [
  {
    name: 'Quân đội (MB)',
    shortName: 'MB',
    employeeCount: 58,
    percentage: 58,
    color: '#2563eb',
    transferCount: 120,
    totalPaidVnd: 3_000_000_000,
  },
  {
    name: 'Hàng hải (MSB)',
    shortName: 'MSB',
    employeeCount: 14,
    percentage: 14,
    color: '#0284c7',
    transferCount: 28,
    totalPaidVnd: 700_000_000,
  },
];

describe('BankDistributionChart', () => {
  it('renders a compact textual summary and exact distribution values', () => {
    render(<BankDistributionChart items={items} totalEmployees={100} />);

    expect(screen.getByText('100')).toBeInTheDocument();
    expect(screen.getByText('Quân đội (MB)')).toBeInTheDocument();
    expect(screen.getByText('Hàng hải (MSB)')).toBeInTheDocument();
    expect(screen.getAllByText('58%')).toHaveLength(2);
    expect(
      screen.getByRole('img', {
        name: 'Phân bổ ngân hàng nhận lương: MB 58%, MSB 14%',
      }),
    ).toBeInTheDocument();
  });
});
