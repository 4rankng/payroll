// Advisory cash-prep forecast for the next timesheet bulk transfer.
// Mirrors backend dto.CashReadinessResponse. All monetary fields are VND.
export interface CashReadinessResponse {
  confirmed_payable: number;
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
  method: string; // "monte-carlo" | "gamma-fit" | "no-history" | "growth-adjusted"
  confidence: string; // "high" | "medium" | "low"
  basis_cycles: number;
  growth_rate: number; // EWMA growth factor (1.0 = stationary; >1 = upward trend)
  generated_at: string; // RFC3339
}

export type CashReadinessParams = {
  project_id?: number;
  employee_id?: number;
  fromDate?: string;
  toDate?: string;
};
