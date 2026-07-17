export type CashReadinessReliabilityState = 'uncalibrated' | 'learning' | 'measured';

// Advisory forecast for the next timesheet Ky only.
// Mirrors backend dto.CashReadinessResponse. All monetary fields are integer VND.
// Forecast-v2 fields are optional so the UI remains compatible during rollout.
export interface CashReadinessResponse {
  observed_approved: number;
  projected_p50: number;
  projected_expected: number;
  projected_p95: number;
  band_lower: number;
  band_upper: number;
  expected_total: number;
  wallet_available: number;
  wallet_available_ok: boolean;
  cash_to_prepare: number;
  gap: number;
  prepare_by_date: string; // RFC3339
  next_pay_date: string; // RFC3339
  lead_days: number;
  ky: number;
  cycle_day_today: number;
  method: string; // e.g. "bootstrap-v2" | "completed-cycle-bootstrap" | "no-history"
  confidence: string; // "high" | "medium" | "low"
  basis_cycles: number;
  growth_rate: number; // EWMA growth factor (1.0 = stationary; >1 = upward trend)
  generated_at: string; // RFC3339

  // Additive forecast-v2 decomposition and reserve contract.
  pending_target_amount?: number;
  expected_pending_amount?: number;
  expected_future_amount?: number;
  expected_payout?: number;
  recommended_reserve?: number;
  interval_lower?: number;
  interval_upper?: number;

  // Accuracy is based on resolved forecasts for the same decision horizon.
  model_version?: string;
  calibration_samples?: number;
  reliability_state?: CashReadinessReliabilityState;
  accuracy_wape?: number;
  accuracy_bias?: number;
  interval_coverage?: number;
  reserve_shortfall_rate?: number;
}

export type CashReadinessParams = {
  project_id?: number;
  employee_id?: number;
  fromDate?: string;
  toDate?: string;
};
