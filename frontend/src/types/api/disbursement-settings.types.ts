// Types for the consolidated disbursement settings endpoint
// (TASK-034 §9 / TASK-038). Provides the active provider name,
// capabilities, fee info, and balances needed to drive provider-agnostic UI.

export interface ProviderCapabilities {
  balance_inquiry: boolean;
  account_verifier: boolean;
  report_export: boolean;
  status_poller: boolean;
  reconcile_via_inquiry: boolean;
}

export interface ActiveProvider {
  name: string; // e.g. '9pay' | '1pay'
  for_employee: boolean;
  for_bulk_transfer: boolean;
  capabilities: ProviderCapabilities;
}

export interface DisbursementFeeInfo {
  current_vnd: number;
  effective_date: string;
  upcoming: Array<{ effective_date: string; fee_vnd: number; notes?: string }>;
}

export interface InternalBalance {
  available: number;
  currency: string;
}

export interface ProviderBalance {
  amount: number;
  checked_at: string;
}

export interface DisbursementSettings {
  active_provider: ActiveProvider;
  fee: DisbursementFeeInfo;
  internal_balance: InternalBalance;
  provider_balance?: ProviderBalance; // only present when capabilities.balance_inquiry
  registered_providers?: Array<{ name: string; label: string }>;
}
