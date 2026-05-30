// Payrates configuration types

export type PayType = 'normal' | 'overtime' | 'weekend' | 'holiday';

export interface PayrateConfig {
  id: number;
  project_id: number;
  employee_id?: number; // NULL means applies to all employees in project
  paytype: PayType;
  rate_vnd: number; // VND per hour
  from_date: string;
  to_date?: string; // NULL means indefinite
  last_updated_by: number;
  last_approved_by?: number;
  approved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PayrateFormData {
  project_id: number;
  employee_id?: number;
  paytype: PayType;
  rate_vnd: number;
  from_date: string;
  to_date?: string;
}

export interface PayrateStats {
  total_payrates: number;
  active_payrates: number;
  pending_approval: number;
  avg_normal_rate: number;
  avg_overtime_rate: number;
}