import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { WalletDemandForecastResponse } from '@/types/api/wallet.types';
import { WalletDemandCard } from './WalletDemandCard';

const forecast: WalletDemandForecastResponse = {
  current_for_month: '2026-07',
  current_cycle_day: 1,
  max_cycle_day: 20,
  periods: [],
  generated_at: '2026-07-20T09:00:00+07:00',
  prediction: {
    actual_so_far: 0,
    projected_total: 100_000_000,
    projected_paid: 100_000_000,
    already_paid: 0,
    remaining_to_pay: 100_000_000,
    recommended_balance: 100_000_000,
    current_available: 52_358_330,
    shortfall: 47_641_670,
    surplus: 0,
    completion_rate: 1,
    method: 'monte-carlo',
    confidence: 'high',
    basis_periods: 3,
    lead_days: 2,
    horizon_cycle_day: 3,
  },
};

describe('WalletDemandCard', () => {
  it('describes the recommendation as the amount needed for the remaining period', () => {
    render(<WalletDemandCard data={forecast} />);

    expect(screen.getByText('Phần còn lại của kỳ')).toBeInTheDocument();
    expect(screen.queryByText(/Nạp thêm để đạt mức/)).not.toBeInTheDocument();
    expect(screen.queryByText(/2 ngày tới/)).not.toBeInTheDocument();
  });

  it('does not repeat the top-up guidance when the wallet is sufficiently funded', () => {
    render(
      <WalletDemandCard
        data={{
          ...forecast,
          prediction: {
            ...forecast.prediction,
            current_available: 120_000_000,
            shortfall: 0,
            surplus: 20_000_000,
          },
        }}
      />,
    );

    expect(screen.getByText('Phần còn lại của kỳ')).toBeInTheDocument();
    expect(screen.queryByText(/Đủ chi trả cho phần còn lại/)).not.toBeInTheDocument();
  });

  it('keeps current-demand wording when there is no historical forecast', () => {
    render(
      <WalletDemandCard
        data={{
          ...forecast,
          prediction: {
            ...forecast.prediction,
            method: 'no-history',
            recommended_balance: 0,
            current_available: 52_358_330,
            shortfall: 0,
            surplus: 52_358_330,
          },
        }}
      />,
    );

    expect(
      screen.getByText('Chưa đủ dữ liệu các kỳ trước. Hiển thị nhu cầu hiện tại chờ chi trả.'),
    ).toBeInTheDocument();
    expect(screen.getByText('Đủ chi trả theo nhu cầu hiện tại.')).toBeInTheDocument();
  });
});
