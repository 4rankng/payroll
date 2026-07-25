import { describe, expect, it } from 'vitest';
import type { WalletDemandForecastResponse, WalletDemandPeriod } from '@/types/api/wallet.types';
import { buildWalletDemandChartData } from './WalletDemandChart';

const period = (
  forMonth: string,
  label: string,
  isCurrent: boolean,
  dailyAmounts: number[],
): WalletDemandPeriod => {
  let cumulative = 0;
  return {
    for_month: forMonth,
    label,
    is_current: isCurrent,
    series: dailyAmounts.map((dailyAmount, index) => {
      cumulative += dailyAmount;
      return {
        cycle_day: index + 1,
        day_label: `${20 + index}/7`,
        amount: cumulative,
        daily_amount: dailyAmount,
      };
    }),
  };
};

const forecast: WalletDemandForecastResponse = {
  current_for_month: '2026-07',
  current_cycle_day: 2,
  max_cycle_day: 4,
  periods: [
    period('2026-07', 'Kỳ 07/2026', true, [0, 45_000_000, 0, 0]),
    period('2026-06', 'Kỳ 06/2026', false, [20_000_000, 20_000_000, 30_000_000, 30_000_000]),
    period('2026-05', 'Kỳ 05/2026', false, [10_000_000, 30_000_000, 20_000_000, 40_000_000]),
  ],
  prediction: {
    actual_so_far: 45_000_000,
    projected_total: 280_000_000,
    projected_paid: 270_000_000,
    already_paid: 45_000_000,
    remaining_to_pay: 225_000_000,
    recommended_balance: 250_000_000,
    current_available: 50_000_000,
    shortfall: 200_000_000,
    surplus: 0,
    completion_rate: 0.95,
    method: 'monte-carlo',
    confidence: 'high',
    basis_periods: 2,
    p50_reference: 225_000_000,
  },
  generated_at: '2026-07-25T12:00:00+07:00',
};

describe('buildWalletDemandChartData', () => {
  it('anchors the active-cycle line to the live current-state forecast', () => {
    const { periods, rows } = buildWalletDemandChartData(forecast);
    const currentKey = periods.find((item) => item.is_current)?.displayLabel;

    expect(currentKey).toBe('Dự báo 07/2026');
    expect(rows[1][currentKey!]).toBe(225_000_000);
    expect(rows[1][currentKey!]).not.toBe(85_000_000);
  });

  it('preserves the historical remaining-demand shape after today', () => {
    const { periods, rows } = buildWalletDemandChartData(forecast);
    const currentKey = periods.find((item) => item.is_current)?.displayLabel;

    expect(rows[2][currentKey!]).toBe(158_823_529);
    expect(rows[3][currentKey!]).toBe(92_647_059);
  });
});
