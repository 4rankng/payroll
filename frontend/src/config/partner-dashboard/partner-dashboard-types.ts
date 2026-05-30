// Partner Dashboard Types

export interface Project {
  id: number;
  name: string;
  startDate: string;
  employeeCount: number;
  status: ProjectStatus;
  statusColor: string;
}

export type ProjectStatus = 'Đang chạy' | 'Sắp hoàn thành' | 'Hoàn thành' | 'Tạm dừng';

export interface ProjectStats {
  totalProjects: number;
  totalEmployees: number;
  activeProjects: number;
}

export interface PartnerDashboardData {
  projects: Project[];
  stats: ProjectStats;
  loading: boolean;
}