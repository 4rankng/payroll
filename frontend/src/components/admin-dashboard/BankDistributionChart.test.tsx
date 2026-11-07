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
  it('renders the donut contract: centre total, legend rows and whole-percent shares', () => {
    render(<BankDistributionChart items={items} totalEmployees={100} />);

    expect(screen.getByText('100')).toBeInTheDocument();

    // Centre figure: compact total paid across all banks.
    expect(
      screen.getByRole('img', { name: 'TỔNG CHI TRẢ 3,7 tỷ' }),
    ).toBeInTheDocument();

    // Legend rows — short label + "transactions · compact amount".
    const legend = screen.getByText('120 gd · 3 tỷ').closest('ul');
    expect(legend).toHaveTextContent('MB');
    expect(legend).toHaveTextContent('MSB');
    expect(screen.getByText('120 gd · 3 tỷ')).toBeInTheDocument();
    expect(screen.getByText('28 gd · 700 tr')).toBeInTheDocument();

    // Shares are recomputed over the donut's own value set (headcount).
    expect(screen.getByText('81%')).toBeInTheDocument();
    expect(screen.getByText('19%')).toBeInTheDocument();

    // The old stacked bar visual is gone.
    expect(
      screen.queryByRole('img', { name: /Phân bổ ngân hàng nhận lương/ }),
    ).not.toBeInTheDocument();
  });
});
