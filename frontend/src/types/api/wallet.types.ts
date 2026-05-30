// Wallet types

export interface WalletBalance {
  available: number;
  pending_in: number;
  pending_out: number;
  currency: string;
  as_of: string;
}

// Wallet Topup types — matches backend wallet.WalletTopup / WalletTopupResponse
export interface WalletTopup {
  id: number;
  amount: number;
  bank_ref: string;
  occurred_at: string;
  note?: string;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface CreateWalletTopupRequest {
  amount: number;
  bank_ref: string;
  occurred_at: string;
  note?: string;
}

export interface WalletTopupListResponse {
  data: WalletTopup[];
  total: number;
  page: number;
  page_size: number;
}

// Wallet Payment types
export interface WalletPayment {
  id: number;
  txn_id: string;
  request_id: string;
  invoice_no?: string;
  requested_amount: number;
  charged_amount?: number;
  fee?: number;
  recipient_name: string;
  recipient_account_no: string;
  recipient_bank: string;
  metadata?: Record<string, unknown>;
  status: string;
  error_code?: string;
  error_message?: string;
  entity_id?: number;
  version: number;
  created_at: string;
  updated_at: string;
  settled_at?: string;
  created_by?: number;
}

export interface WalletPaymentListResponse {
  data: WalletPayment[];
  total: number;
  page: number;
  page_size: number;
}

export interface WalletPaymentFilter {
  txn_id?: string;
  request_id?: string;
  invoice_no?: string;
  status?: string;
  error_code?: string;
  recipient_name?: string;
  recipient_account?: string;
  recipient_bank?: string;
  start_date?: string;
  end_date?: string;
  entity_id?: number;
  page?: number;
  page_size?: number;
}

// Unified Transaction types
export interface UnifiedTransaction {
  id: number;
  type: 'topup' | 'payment';
  amount: number;
  status: string;
  occurred_at: string;
  reference: string;
  counterparty: string;
  note: string;
}

export interface UnifiedTransactionListResponse {
  data: UnifiedTransaction[];
  total: number;
  page: number;
  page_size: number;
}

export interface UnifiedTransactionFilter {
  type?: 'topup' | 'payment';
  status?: string;
  start_date?: string;
  end_date?: string;
  page?: number;
  page_size?: number;
}

// Reconciliation types
export interface ReconciliationJob {
  id: string;
  status: 'processing' | 'completed' | 'failed';
  total_rows: number;
  matched: number;
  unmatched: number;
  headers?: string[];
  raw_rows?: string[][];
}

export interface SyncBalanceResponse {
  provider_balance: number;
  local_balance: number;
  currency: string;
}

export interface AdjustBalanceRequest {
  amount: number;
  reason: string;
}
