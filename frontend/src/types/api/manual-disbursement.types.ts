// Types for the admin "Chuyển tiền" (Manual Disbursement) feature.
// Mirrors the backend manualDisbursementResponse / request shapes.

export type ManualDisbursementStatus =
  | "pending"
  | "authorised"
  | "completed"
  | "failed"

export interface ManualDisbursementResponse {
  txn_id: string;
  request_id: string;
  invoice_no: string;
  status: ManualDisbursementStatus;
  requested_amount: number;
  fee: number;
  recipient_name: string;
  recipient_account_no: string;
  recipient_bank: string;
  description?: string | null;
  error_code?: string | null;
  error_message?: string | null;
  created_at: string;
  settled_at?: string | null;
  created_by?: number | null;
}

export interface InitiateManualDisbursementRequest {
  amount: number;
  description: string;
  bank_code: string;
  account_no: string;
  account_name: string;
  account_type: "0" | "1";
}

export interface VerifyAccountRequest {
  bank_code: string;
  account_no: string;
  account_type: "0" | "1";
  account_name?: string;
}

export interface VerifyAccountResponse {
  Valid: boolean;
  BankCode: string;
  AccountNo: string;
  AccountName: string;
  AccountType: string;
  RawErrorCode: string;
  RawMessage: string;
}

export type EmployeeAccountLookupOutcome =
  | 'valid'
  | 'invalid'
  | 'name_mismatch'
  | 'unverified';

/** The server resolves every bank field from the selected employee's record. */
export interface EmployeeAccountLookupRequest {
  employee_id: number;
}

export interface EmployeeAccountLookupResponse {
  employee: {
    id: number;
    fullname: string;
  };
  stored_bank: {
    bank_id: number;
    bank_name: string;
    bank_code: string;
    swift_code: string;
    account_number: string;
    account_name: string;
  };
  outcome: EmployeeAccountLookupOutcome;
  provider_result: VerifyAccountResponse;
}

export const TERMINAL_STATUSES: ManualDisbursementStatus[] = [
  "completed",
  "failed",
];

export const isTerminalStatus = (s: ManualDisbursementStatus): boolean =>
  TERMINAL_STATUSES.includes(s);

export interface BankInfoResponse {
  bank_code: string;
  swift_code: string;
  bank_name: string;
  logo_url?: string;
}

export interface WalletBalanceInfo {
  wallet_balance: number;
  utilised_amount: number;
  available_balance: number;
  last_synced_at: string;
}
