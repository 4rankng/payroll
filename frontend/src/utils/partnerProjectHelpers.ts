import { ProjectEmployee, ProjectStats } from "@/types/partnerProject";

export const getEmployeeStatusColor = (status: string): string => {
  switch (status) {
    case "Đang dùng":
      return "bg-success text-white";
    case "Nghỉ phép":
      return "bg-warning text-white";
    case "Ngừng hoạt động":
      return "bg-destructive text-white";
    default:
      return "bg-secondary text-foreground";
  }
};

export const getProjectStatusColor = (status: string): string => {
  switch (status) {
    case "Đang chạy":
      return "bg-success text-white";
    case "Hoàn thành":
      return "bg-primary text-white";
    case "Tạm dừng":
      return "bg-warning text-white";
    case "Chuẩn bị":
      return "bg-info text-white";
    default:
      return "bg-secondary text-foreground";
  }
};

export const calculateProjectStats = (employees: ProjectEmployee[]): ProjectStats => {
  const activeEmployees = employees.filter(emp => emp.status === "Đang dùng").length;
  const totalDepartments = new Set(employees.map(emp => emp.department)).size;

  return {
    totalEmployees: employees.length,
    activeEmployees,
    totalDepartments,
    uploadedTimesheets: 0, // This would come from API
  };
};

export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';
  
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

export const validateExcelFile = (file: File): { isValid: boolean; error?: string } => {
  const allowedTypes = [
    'application/vnd.ms-excel',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
  ];
  
  if (!allowedTypes.includes(file.type)) {
    return {
      isValid: false,
      error: "Định dạng file không hợp lệ. Vui lòng chọn file Excel (.xls hoặc .xlsx)"
    };
  }
  
  // Check file size (max 10MB)
  const maxSize = 10 * 1024 * 1024; // 10MB
  if (file.size > maxSize) {
    return {
      isValid: false,
      error: "File quá lớn. Kích thước tối đa cho phép là 10MB"
    };
  }
  
  return { isValid: true };
};