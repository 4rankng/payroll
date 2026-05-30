export interface PartnerProject {
  id: string;
  name: string;
  startDate: string;
  status: 'Đang chạy' | 'Hoàn thành' | 'Tạm dừng' | 'Chuẩn bị';
  statusColor: string;
  endDate?: string;
  description?: string;
}

export interface ProjectEmployee {
  id: number;
  name: string;
  position: string;
  department: string;
  joinDate: string;
  status: 'Đang dùng' | 'Nghỉ phép' | 'Ngừng hoạt động';
  email?: string;
  phone?: string;
}

export interface ProjectStats {
  totalEmployees: number;
  activeEmployees: number;
  totalDepartments: number;
  uploadedTimesheets: number;
}

export interface TimesheetUpload {
  id: string;
  filename: string;
  size: number;
  uploadDate: string;
  status: 'processing' | 'completed' | 'failed';
  processedCount?: number;
  errorCount?: number;
}