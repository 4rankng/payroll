import { describe, expect, it } from 'vitest';

import type { CashReadinessResponse } from '@/types/api/cash-readiness.types';

import { getCashReadinessDisplay } from './cashReadinessDisplay';

const legacyResponse = (): CashReadinessResponse => ({
  observed_approved: 100,
  projected_p50: 50,
  projected_expected: 70,
  projected_p95: 90,
  band_lower: 150,
  band_upper: 190,
  expected_total: 170,
  wallet_available: 0,
  wallet_available_ok: true,
  cash_to_prepare: 150,
  gap: 150,
  prepare_by_date: '2026-07-22T00:00:00+07:00',
  next_pay_date: '2026-07-24T00:00:00+07:00',
  lead_days: 2,
  ky: 3,
  cycle_day_today: 3,
  method: 'normalized-bootstrap',
  confidence: 'low',
  basis_cycles: 5,
  growth_rate: 1,
  generated_at: '2026-07-17T12:00:00+07:00',
});

describe('getCashReadinessDisplay', () => {
  it('keeps legacy responses useful during an additive rollout', () => {
    const display = getCashReadinessDisplay(legacyResponse());

    expect(display.expectedPayout).toBe(170);
    expect(display.recommendedReserve).toBe(170);
    expect(display.showExpectedPayout).toBe(false);
    expect(display.intervalLower).toBe(150);
    expect(display.intervalUpper).toBe(190);
    expect(display.drivers.map((driver) => driver.label)).toEqual(['Đã duyệt', 'Phần còn lại']);
  });

  it('does not show an empty-cycle warning and hides accuracy before measurement', () => {
    const display = getCashReadinessDisplay({
      ...legacyResponse(),
      method: 'completed-cycle-bootstrap',
      expected_payout: 400,
      recommended_reserve: 450,
      interval_lower: 300,
      interval_upper: 500,
      reliability_state: 'uncalibrated',
      calibration_samples: 0,
      accuracy_wape: 0,
    });

    expect(display.dataWarning).toBeUndefined();
    expect(display.showExpectedPayout).toBe(true);
    expect(display.accuracyMetrics).toEqual([]);
  });

  it('shows accuracy only after the backend marks the forecast measured', () => {
    const display = getCashReadinessDisplay({
      ...legacyResponse(),
      reliability_state: 'measured',
      calibration_samples: 24,
      accuracy_wape: 0.08,
      accuracy_bias: -0.02,
      interval_coverage: 0.9,
      reserve_shortfall_rate: 0.05,
    });

    expect(display.reliabilityLabel).toBe('Đã đo bằng kết quả thực tế');
    expect(display.accuracyMetrics).toHaveLength(4);
  });

  it('hides an empty breakdown that only repeats the headline amount', () => {
    const display = getCashReadinessDisplay({
      ...legacyResponse(),
      observed_approved: 0,
      pending_target_amount: 0,
      expected_pending_amount: 0,
      expected_future_amount: 350_557_938,
      expected_payout: 350_557_938,
      recommended_reserve: 350_557_938,
    });

    expect(display.drivers).toEqual([]);
  });

  it('keeps a real multi-part breakdown but removes zero-value rows', () => {
    const display = getCashReadinessDisplay({
      ...legacyResponse(),
      observed_approved: 100,
      pending_target_amount: 0,
      expected_future_amount: 70,
      expected_payout: 170,
      recommended_reserve: 170,
    });

    expect(display.drivers.map((driver) => driver.label)).toEqual(['Đã duyệt', 'Chưa phát sinh']);
  });
});
