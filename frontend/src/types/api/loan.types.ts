// Loan domain types derived from backend API specs

export type LoanType = 'bullet_loan' | 'custom_schedule';
export type LoanStatus = 'active' | 'closed';
export type ScheduleStatus = 'pending' | 'paid' | 'overdue';
export type ScheduleType = 'interest' | 'principal';

export type ScheduleRow = {
  id: string;
  due_date: string;
  amount: string;
};

// ---------------------------------------------------------------------------
// Lender types
// ---------------------------------------------------------------------------

export interface Lender {
  id: number;
  name: string;
  cccd?: string | null;
  email?: string | null;
  mobile?: string | null;
  notes?: string | null;
  bank_id?: number | null;
  bank_account_number?: string | null;
  bank_account_name?: string | null;
  bank?: {
    id: number;
    branch_name: string;
  } | null;
  created_at: string;
  updated_at: string;
}

export interface LenderSummary {
  total_principal_borrowed: number;
  total_principal_repaid: number;
  outstanding_principal: number;
  active_loans_count: number;
}

export interface LenderWithSummary extends Lender {
  summary: LenderSummary;
}

export interface LenderFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  search?: string;
}

export interface CreateLenderRequest {
  name: string;
  cccd?: string;
  email?: string;
  mobile?: string;
  notes?: string;
  bank_id?: number;
  bank_account_number?: string;
  bank_account_name?: string;
}

export type UpdateLenderRequest = Partial<CreateLenderRequest>;

// ---------------------------------------------------------------------------
// Loan types
// ---------------------------------------------------------------------------

export interface LoanLender {
  id: number;
  name: string;
  email?: string | null;
  mobile?: string | null;
}

export interface CustomScheduleItem {
  id: number;
  period: number;
  due_date: string;
  amount: number;
  status: ScheduleStatus;
  paid_at: string | null;
}

export interface Loan {
  id: number;
  loan_code: string;
  loan_type: LoanType;
  lender: LoanLender;
  principal_amount: number;
  outstanding_principal: number;
  interest_rate_bps: number | null;
  monthly_interest: number | null;
  total_interest_paid: number;
  term_months: number | null;
  disbursement_date: string;
  payment_day_of_month: number | null;
  next_payment_date?: string | null;
  next_payment_amount?: number | null;
  status: LoanStatus;
  description?: string | null;
  disbursement_reference?: string | null;
  created_at: string;
  updated_at: string;
  schedules?: CustomScheduleItem[];
}

export interface LoanScheduleItem {
  period: number;
  due_date: string;
  type: ScheduleType;
  amount: number;
  status: ScheduleStatus;
}

export type ScheduleItem = LoanScheduleItem;

export interface LoanFilters {
  page?: number;
  pageSize?: number;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
  lender_id?: number;
  status?: LoanStatus;
}

export interface CreateLoanRequest {
  lender_id: number;
  principal_amount: number;
  start_date: string;
  description?: string;
  disburse_now?: boolean;
  disbursement_reference?: string;
  interest_rate_bps?: number;
  term_months?: number;
  payment_day_of_month?: number;
  schedules?: {
    due_date: string;
    amount: number;
  }[];
}

export interface DisburseLoanRequest {
  disbursement_date: string;
  disbursement_reference?: string;
}

export interface DisburseLoanResponse {
  loan_id: number;
  disbursed_amount: number;
  ledger_entry_ids: number[];
}

export interface RepayLoanRequest {
  amount: number;
  payment_date: string;
  payment_reference?: string;
  notes?: string;
}

export interface RepayLoanResponse {
  loan_id: number;
  repaid_amount: number;
  outstanding_principal: number;
  status: LoanStatus;
  ledger_entry_ids: number[];
}

export interface RepayScheduleRequest {
  schedule_id: number;
  payment_date: string;
  payment_reference?: string;
  notes?: string;
}

export interface RepayScheduleResponse {
  loan_id: number;
  schedule_id: number;
  amount: number;
  status: ScheduleStatus;
  transaction_id?: number;
}

export interface UpdateLoanRequest {
  description?: string;
  payment_day_of_month?: number;
  disbursement_reference?: string;
}

// ---------------------------------------------------------------------------
// Form state helpers
// ---------------------------------------------------------------------------

export type FormLoanType = 'bullet_loan' | 'amortization' | 'custom_schedule';

export interface FormState {
  loan_type: FormLoanType;
  lender_id: string;
  principal_amount: string;
  start_date: string;
  description: string;
  disburse_now: boolean;
  disbursement_reference: string;
  interest_rate_percent: string;
  term_months: string;
  payment_day_of_month: string;
  custom_term_months: string;
  custom_payment_day_of_month: string;
  custom_monthly_amount: string;
  custom_last_month_amount: string;
  schedules: ScheduleRow[];
}
