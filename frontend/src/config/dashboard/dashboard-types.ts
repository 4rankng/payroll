export interface Employee {
  id: string;
  name: string;
  position: string;
  status: string;
  avatar?: string;
}

export interface Project {
  id: string;
  name: string;
  progress: number;
  color: string;
  employeeCount: number;
  status: string;
}

export interface Activity {
  id: string;
  description: string;
  user: string;
  timestamp: string;
  status: 'success' | 'warning' | 'error' | 'info';
}

export interface Notification {
  id: string;
  title: string;
  message: string;
  createdAt: string;
  read: boolean;
}

export interface DashboardStats {
  totalEmployees: number;
  activeProjects: number;
  pendingTimesheets: number;
  monthlyPayroll: number;
  revenueGrowth: number;
}