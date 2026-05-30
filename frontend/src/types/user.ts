export interface User {
  id: number;
  email: string;
  username: string;
  fullname: string;
  role: 'admin' | 'partner' | 'employee' | 'adv_partner';
  last_login?: string;
  created_at: string;
  updated_at: string;
}

export type UserFormData = {
  email: string;
  username: string;
  fullname: string;
  password: string;
  role: User["role"] | "";
};

export interface CreateUserData {
  email?: string;
  username: string;
  fullname: string;
  password: string;
  role: 'admin' | 'partner' | 'employee' | 'adv_partner';
}

export interface UpdateUserData {
  email?: string;
  username?: string;
  fullname?: string;
  role?: 'admin' | 'partner' | 'employee' | 'adv_partner';
}

export interface UserSummary {
  total_users: number;
  total_admins: number;
  total_partners: number;
  total_employees: number;
  recent_logins_today: number;
}

export interface ResetPasswordData {
  password: string;
}

export interface UserActivitySummary {
  user_id: number;
  period_days: number;
  authentication: {
    total_logins: number;
    last_login: string | null;
  };
  payroll_operations: {
    timesheets_managed: number;
  };
  recent_activities: UserActivity[];
}

export interface UserActivity {
  id: number;
  action: string;
  entity_type: string;
  entity_id: number;
  business_context: string;
  impact_level: 'low' | 'medium' | 'high';
  ip_address: string;
  created_at: string;
}