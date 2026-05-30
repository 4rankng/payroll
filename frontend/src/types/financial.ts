// Financial Management Types

export type AccountType = 'capital' | 'revenue' | 'expense' | 'salary' | 'partner_payment';

export interface LedgerEntry {
  id: number;
  account_type: AccountType;
  debit: number;
  credit: number;
  balance: number;
  description: string;
  reference_id?: number; // Links to PayrollItem, etc
  reference_type?: string;
  party?: string; // Who paid/received from
  transaction_date: string;
  created_by: number;
  created_at: string;
  updated_at: string;
}

export interface FinancialSummary {
  total_capital: number;
  total_revenue: number;
  total_expenses: number;
  total_salary_costs: number;
  total_partner_payments: number;
  net_profit: number;
  cash_flow: number;
}

export interface BlacklistedToken {
  id: number;
  token_hash: string;
  user_id: number;
  expires_at: string;
  created_at: string;
}

export interface AuditLog {
  id: number;
  user_id: number;
  action: string; // CREATE, UPDATE, DELETE
  table_name: string;
  record_id: number;
  old_values?: Record<string, unknown>;
  new_values?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}
