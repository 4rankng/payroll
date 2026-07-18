// Settlement Simulation types — mirrors backend dto.SimulateSettlementRequest
// and dto.SimulationResult (see backend/internal/app/dto/payroll.go).
//
// The simulation projects N future "Xuất sao kê" exports starting from an
// admin-chosen date. It reuses the EXACT same PayrollReportByProjectService
// selection logic as production (GET /timesheets/payroll/report), so whatever
// the real export picks up, the sim picks up. Returns a coverage verdict:
// "will these N exports reconcile every paid-but-unsettled timesheet?"

export type SettlementVerdict =
  | 'AN_TOAN_DE_XUAT' // safe — full pool covered, reconciled
  | 'CAN_KIEM_TRA'; // needs review — remainders or reconciliation drift

export interface SettlementSimulationRequest {
  start_date: string; // YYYY-MM-DD — first export date (required)
  export_count?: number; // default 4, clamped [1,10]
  cadence_days?: number; // default 7; step between exports
  project_ids?: number[];
  employee_ids?: number[];
}

export interface SettlementSimulationResult {
  snapshot_epoch: string;
  start_date: string;
  export_dates: string[];
  verdict: SettlementVerdict;
  summary: SimulationSummary;
  reconciliation: ReconciliationResult;
  exports: ExportProjection[];
  remainders: RemainderRow[];
  warnings: SimWarning[];
}

export interface SimulationSummary {
  total_eligible_timesheets: number;
  total_eligible_groups: number;
  total_eligible_amount: number;
  total_included_timesheets: number;
  total_included_groups: number;
  total_included_amount: number;
  remaining_timesheets: number;
  remaining_groups: number;
  remaining_amount: number;
  all_settled: boolean;
}

export interface ReconciliationResult {
  exported_total: number;
  ledger_receivable: number;
  delta: number;
  reconciled: boolean;
}

export interface ExportProjection {
  sequence: number;
  export_date: string;
  from_date: string;
  to_date: string;
  included_count: number;
  included_amount: number;
  excluded_count: number;
  remaining_count: number;
  remaining_amount: number;
  included: SimulationRow[];
  excluded: SimulationExcludedRow[];
}

export interface SimulationRow {
  employee_id: number;
  employee_name: string;
  project_id: number;
  project_name: string;
  amount: number;
  timesheet_ids: number[];
  timesheet_dates: string[];
  bank_account_masked?: string;
}

export interface SimulationExcludedRow {
  employee_id: number;
  employee_name: string;
  project_id: number;
  project_name: string;
  amount: number;
  timesheet_ids: number[];
  timesheet_dates: string[];
  reason: string;
  in_production: boolean;
}

export interface RemainderRow {
  employee_id: number;
  employee_name: string;
  project_id: number;
  project_name: string;
  amount: number;
  timesheet_ids: number[];
  timesheet_dates: string[];
  reason: string;
  bank_account_masked?: string;
}

export interface SimWarning {
  code: string;
  message: string;
}
