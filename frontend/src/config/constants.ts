// Application Constants

// Status Types
export const USER_STATUS = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
} as const;

export const USER_ROLES = {
  ADMIN: 'admin',
  PARTNER: 'partner',
} as const;

export const PROJECT_STATUS = {
  DRAFT: 'draft',
  ACTIVE: 'active',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
  INACTIVE: 'inactive',
} as const;

export const EMPLOYEE_STATUS = {
  ACTIVE: 'active',
  INACTIVE: 'inactive',
} as const;

export const TIMESHEET_STATUS = {
  DRAFT: 'draft',
  PENDING_APPROVAL: 'pending_approval',
  APPROVED: 'approved',
  REJECTED: 'rejected',
} as const;

export const PAY_TYPES = {
  NORMAL: 'normal',
  OVERTIME: 'overtime',
  WEEKEND: 'weekend',
  HOLIDAY: 'holiday',
} as const;

// Vietnamese Labels
export const USER_STATUS_LABELS = {
  [USER_STATUS.ACTIVE]: 'Đang dùng',
  [USER_STATUS.INACTIVE]: 'Ngừng hoạt động',
} as const;

export const PROJECT_STATUS_LABELS = {
  [PROJECT_STATUS.DRAFT]: 'Bản nháp',
  [PROJECT_STATUS.ACTIVE]: 'Đang dùng',
  [PROJECT_STATUS.COMPLETED]: 'Kết thúc',
  [PROJECT_STATUS.CANCELLED]: 'Đã hủy',
  [PROJECT_STATUS.INACTIVE]: 'X',
} as const;

export const EMPLOYEE_STATUS_LABELS = {
  [EMPLOYEE_STATUS.ACTIVE]: 'Đang làm việc',
  [EMPLOYEE_STATUS.INACTIVE]: 'Nghỉ việc',
} as const;

export const PAY_TYPE_LABELS = {
  [PAY_TYPES.NORMAL]: 'Giờ thường',
  [PAY_TYPES.OVERTIME]: 'Tăng ca',
  [PAY_TYPES.WEEKEND]: 'Cuối tuần',
  [PAY_TYPES.HOLIDAY]: 'Ngày lễ',
} as const;

// Date Formats
export const DATE_FORMATS = {
  API: 'yyyy-MM-dd', // API format
  DISPLAY: 'dd/MM/yyyy', // Display format
  DISPLAY_WITH_TIME: 'dd/MM/yyyy HH:mm',
  TIME: 'HH:mm:ss',
} as const;

// Currency
export const CURRENCY = {
  CODE: 'VND',
  SYMBOL: '₫',
  LOCALE: 'vi-VN',
} as const;

// Validation Rules
export const VALIDATION = {
  PASSWORD: {
    MIN_LENGTH: 8,
    REQUIRE_UPPERCASE: true,
    REQUIRE_LOWERCASE: true,
    REQUIRE_NUMBER: true,
    REQUIRE_SPECIAL: true,
    SPECIAL_CHARS: '!@#$%^&*()_+-=[]{}|;:,.<>?',
  },
  CCCD: {
    LENGTH: 12,
    PATTERN: /^\d{12}$/,
  },
  PHONE: {
    PATTERN: /^(\+84|84|0)[3-9]\d{8}$/,
  },
  EMAIL: {
    PATTERN: /^[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}$/i,
  },
  TIMESHEET: {
    MIN_HOURS: 0,
    MAX_HOURS: 24,
    MAX_DAYS_PER_EMPLOYEE: 30,
  },
} as const;

// Error Messages (Vietnamese)
export const ERROR_MESSAGES = {
  REQUIRED: 'Trường này là bắt buộc',
  INVALID_EMAIL: 'Email không hợp lệ',
  INVALID_PHONE: 'Số điện thoại không hợp lệ',
  INVALID_CCCD: 'CCCD phải có 12 chữ số',
  INVALID_DATE: 'Ngày không hợp lệ',
  DATE_RANGE: 'Ngày kết thúc phải sau ngày bắt đầu',
  PASSWORD_TOO_SHORT: `Mật khẩu phải có ít nhất ${VALIDATION.PASSWORD.MIN_LENGTH} ký tự`,
  PASSWORD_REQUIREMENTS: 'Mật khẩu phải có chữ hoa, chữ thường, số và ký tự đặc biệt',
  HOURS_INVALID: `Số giờ phải từ ${VALIDATION.TIMESHEET.MIN_HOURS} đến ${VALIDATION.TIMESHEET.MAX_HOURS}`,
  NETWORK_ERROR: 'Lỗi kết nối. Vui lòng thử lại',
  UNAUTHORIZED: 'Bạn không có quyền thực hiện thao tác này',
  SESSION_EXPIRED: 'Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại',
  SERVER_ERROR: 'Đã xảy ra lỗi. Vui lòng thử lại sau',
} as const;

// Success Messages (Vietnamese)
export const SUCCESS_MESSAGES = {
  LOGIN: 'Đăng nhập thành công',
  LOGOUT: 'Đăng xuất thành công',
  CREATED: 'Tạo mới thành công',
  UPDATED: 'Cập nhật thành công',
  DELETED: 'Xóa thành công',
  APPROVED: 'Phê duyệt thành công',
  REJECTED: 'Loại thành công',
  IMPORTED: 'Nhập dữ liệu thành công',
  EXPORTED: 'Xuất dữ liệu thành công',
  PASSWORD_CHANGED: 'Đổi mật khẩu thành công',
  BULK_CREATED: 'Tạo hàng loạt thành công',
  BULK_UPDATED: 'Cập nhật hàng loạt thành công',
  BULK_DELETED: 'Xóa hàng loạt thành công',
  BULK_APPROVED: 'Phê duyệt hàng loạt thành công',
  BULK_REJECTED: 'Loại hàng loạt thành công',
} as const;

// Sorting Options
export const SORT_OPTIONS = {
  CREATED_AT: 'created_at',
  UPDATED_AT: 'updated_at',
  NAME: 'name',
  DATE: 'date',
  AMOUNT: 'amount',
} as const;

export const SORT_ORDER = {
  ASC: 'asc',
  DESC: 'desc',
} as const;
