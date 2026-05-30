// Test data factory for generating consistent test data without faker
export class TestDataFactory {
  static createEmployee() {
    return {
      firstName: 'Nguyễn',
      lastName: 'Văn An',
      email: 'an.nguyen@company.com',
      phone: '0901234567',
      address: '123 Đường Trần Hưng Đạo',
      city: 'Hồ Chí Minh',
      state: 'Hồ Chí Minh',
      zipCode: '700000',
      bankAccount: '1234567890',
      bankName: 'Ngân hàng BIDV',
      salary: 50000000,
      position: 'Lập trình viên',
      department: 'Công nghệ thông tin',
      startDate: '2024-01-15',
    };
  }

  static createProject() {
    return {
      name: 'Dự án Hệ thống TingTing',
      description: 'Dự án phát triển hệ thống quản lý lương cho công ty',
      startDate: '2024-01-01',
      endDate: '2024-12-31',
      budget: 500000000,
      status: 'active' as const,
    };
  }

  static createTimesheet() {
    return {
      employeeId: 'emp-001',
      projectId: 'proj-001',
      date: '2024-09-17',
      hours: 8.0,
      description: 'Phát triển tính năng Nhân viên',
      status: 'pending' as const,
    };
  }

  static createUser() {
    return {
      firstName: 'Admin',
      lastName: 'System',
      email: 'admin@company.com',
      password: 'TestPassword123!',
      role: 'admin' as const,
      isActive: true,
    };
  }
}

// Vietnamese text helpers for UI testing
export const VietnameseText = {
  login: {
    title: 'Đăng nhập',
    email: 'Email',
    password: 'Mật khẩu',
    submit: 'Đăng nhập',
    forgotPassword: 'Quên mật khẩu?',
  },
  dashboard: {
    title: 'Bảng điều khiển',
    welcome: 'Chào mừng',
    overview: 'Tổng quan',
  },
  employees: {
    title: 'Nhân viên',
    addEmployee: 'Thêm nhân viên',
    editEmployee: 'Chỉnh sửa nhân viên',
    deleteEmployee: 'Xóa nhân viên',
    search: 'Tìm kiếm',
  },
  timesheet: {
    title: 'Bảng chấm công',
    submit: 'Gửi',
    approve: 'Phê duyệt',
    reject: 'Loại',
    pending: 'Chờ duyệt',
    approved: 'Đã duyệt',
    rejected: 'Đã loại',
  },
  projects: {
    title: 'Dự án',
    addProject: 'Thêm dự án',
    editProject: 'Chỉnh sửa dự án',
    deleteProject: 'Xóa dự án',
  },
  common: {
    save: 'Lưu',
    cancel: 'Đóng',
    confirm: 'Xác nhận',
    delete: 'Xóa',
    edit: 'Chỉnh sửa',
    add: 'Thêm',
    search: 'Tìm kiếm',
    filter: 'Lọc',
    export: 'Xuất',
    import: 'Nhập',
    loading: 'Đang tải...',
    error: 'Lỗi',
    success: 'Thành công',
    warning: 'Cảnh báo',
    info: 'Thông tin',
  },
};

// API endpoints for testing
export const API_ENDPOINTS = {
  auth: {
    login: '/api/auth/login',
    logout: '/api/auth/logout',
    refresh: '/api/auth/refresh',
  },
  employees: {
    list: '/api/employees',
    create: '/api/employees',
    update: '/api/employees/:id',
    delete: '/api/employees/:id',
  },
  projects: {
    list: '/api/projects',
    create: '/api/projects',
    update: '/api/projects/:id',
    delete: '/api/projects/:id',
  },
  timesheet: {
    list: '/api/timesheet',
    submit: '/api/timesheet',
    approve: '/api/timesheet/:id/approve',
    reject: '/api/timesheet/:id/reject',
  },
};
