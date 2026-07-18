// Settlement Simulation types — mirrors backend dto.SimulateSettlementRequest
// and dto.SimulationResult (see backend/internal/app/dto/payroll.go).
//
// The simulation projects the current payroll cycle + next N−1 cycles
// (Kỳ 1–4 monthly cadence) by reusing the production ExportPlanner. It
// returns a full-pool coverage verdict: "will these N exports cover every
// outstanding approved timesheet?" Payroll-only scope — every remainder is
// unpaid wages (the eligible pool is payment_status IN (pending, failed) by
// construction).

export type SettlementVerdict =
  | 'AN_TOAN_DE_XUAT' // safe to export — full pool covered, no blocking, reconciled
  | 'CAN_KIEM_TRA' // needs review — remainders / blocking / drift
  | 'KHONG_THE_TAT_TOAN'; // reserved — cannot fire in payroll-only scope

export interface SettlementSimulationRequest {
  project_ids?: number[];
  employee_ids?: number[];
  projected_cycle_count?: number; // default 4, clamped [1,6] server-side
  for_month?: string; // optional YYYY-MM override
}

export interface SettlementSimulationResult {
  snapshot_epoch: string; // ISO 8601 — max(updated_at) across observed rows
  starting_cycle: CycleMeta;
  projected_cycle_count: number;
  verdict: SettlementVerdict;
  summary: SimulationSummary;
  reconciliation: ReconciliationResult;
  cycles: CycleProjection[];
  remainders: RemainderRow[];
  warnings: SimWarning[];
}

export interface CycleMeta {
  index: number; // 1..4
  month_ref: string; // YYYY-MM-DD (first of month)
  from_date: string;
  to_date: string;
  pay_date: string;
}

export interface SimulationSummary {
  total_eligible_count: number;
  total_eligible_amount: number; // int64 VND
  total_included_count: number;
  total_included_amount: number;
  remaining_after_all_count: number;
  remaining_after_all_amount: number;
  all_settled: boolean;
}

export interface ReconciliationResult {
  exported_total: number;
  ledger_receivable: number;
  delta: number; // receivable − exported; 0 = reconciled
  reconciled: boolean;
}

export interface CycleProjection {
  sequence: number; // 1..N
  label: string; // "Kỳ 2 (hiện tại)"
  from_date: string;
  to_date: string;
  pay_date: string;
  included_count: number;
  included_amount: number;
  excluded_count: number;
  remaining_after_count: number;
  remaining_after_amount: number;
  included: SimulationRow[];
  excluded: SimulationExcludedRow[];
  findings: SimFinding[];
}

export interface SimulationRow {
  employee_id: number;
  employee_name: string;
  project_id: number;
  project_name: string;
  amount: number;
  timesheet_ids: number[];
  bank_account_masked: string; // last 4 only — PII never leaves backend
}

export interface SimulationExcludedRow {
  employee_id: number;
  employee_name: string;
  project_id: number;
  project_name: string;
  amount: number;
  timesheet_ids: number[];
  reason: string;
  in_production: boolean; // true = production also excludes; false = sim-only check
}

// RemainderRow — every remainder is unpaid wages in payroll-only scope.
export interface RemainderRow {
  employee_id: number;
  employee_name: string;
  project_id: number;
  project_name: string;
  amount: number;
  timesheet_ids: number[];
  reason: string;
  bank_account_masked: string;
}

export interface SimFinding {
  severity: 'blocking' | 'warning' | 'info';
  code: string;
  message: string;
  in_production: boolean;
}

export interface SimWarning {
  code: string;
  message: string;
}
