// Import types from payrate structure
import type { DayType } from '@/types/api/payrate.types';

export interface TimesheetItem {
  id: number;
  employee_id: number;
  project_id: number;
  date: string;
  hours_worked: number;

  // New 3-level payrate fields
  position: string; // Position from payrate config (e.g., "phổ thông", "kinh nghiệm")
  day_type: DayType; // Auto-calculated by backend: "ngày thường" | "ngày nghỉ" | "ngày lễ"
  hour_type: string; // User input: "ca ngày", "ca đêm", "tăng ca", "08:00-17:00", etc.

  payrate: number; // VND per hour - calculated by backend from 3-level structure
  amount: number; // calculated by backend: hours_worked × payrate

  status: 'draft' | 'pending_approval' | 'approved' | 'rejected';
  notes?: string;
  approved_by?: number;
  approved_at?: string;
  rejection_reason?: string;
  created_at: string;
  updated_at: string;

  // Related objects (populated via joins)
  employee?: {
    id: number;
    fullname: string;
    employee_code?: string;
  };
  project?: {
    id: number;
    name: string;
    code?: string;
  };

  // Derived fields for UI
  employeeName?: string;

  // Additional calculated fields for display
  workDays?: number;
  actualDays?: number;
  overtime?: number;
  totalHours?: number;
  late?: number;
  absent?: number;
}

export interface TimesheetFormData {
  employee_id: number;
  project_id: number;
  date: string;
  hours_worked: number;

  // New 3-level fields
  position: string; // Required - position from payrate config
  hour_type: string; // Required - hour type for rate calculation
  // day_type will be auto-calculated by backend from date

  notes?: string;
}

export interface TimesheetStats {
  totalActualDays: number;
  totalOvertime: number;
  pendingApproval: number;
  efficiency: number;
  totalEmployees: number;
}

// Additional types for timesheet entry validation
export interface TimesheetValidationResult {
  valid: boolean;
  errors: string[];
  warnings?: string[];
  calculatedRate?: number; // Preview rate from backend
  calculatedAmount?: number; // Preview amount
  calculatedDayType?: DayType; // What day type will be used
}

// For position and hour type selection in forms
export interface TimesheetFormOptions {
  positions: string[]; // Available positions from current payrate config
  hourTypes: string[]; // Available hour types from current payrate config
  dayTypePreview?: DayType; // Preview of day type for selected date
  ratePreview?: number; // Preview of rate for selected position/day_type/hour_type
}


// Utility functions
export function getDisplayPosition(item: TimesheetItem): string {
  return item.position || 'Chưa xác định';
}

export function getDisplayHourType(item: TimesheetItem): string {
  return item.hour_type || 'Chưa xác định';
}

export function getDisplayDayType(item: TimesheetItem): string {
  const dayTypeLabels = {
    'ngày thường': 'Ngày thường',
    'ngày nghỉ': 'Ngày nghỉ',
    'ngày lễ': 'Ngày lễ'
  };
  return dayTypeLabels[item.day_type] || item.day_type;
}


// Validation for timesheet form data
export function validateTimesheetFormData(data: TimesheetFormData): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (!data.employee_id) {
    errors.push('Nhân viên là bắt buộc');
  }

  if (!data.project_id) {
    errors.push('Dự án là bắt buộc');
  }

  if (!data.date) {
    errors.push('Ngày làm việc là bắt buộc');
  }

  if (data.hours_worked < 0) {
    errors.push('Số giờ làm việc không thể âm');
  }

  if (data.hours_worked && data.hours_worked > 24) {
    errors.push('Số giờ làm việc không thể vượt quá 24 giờ');
  }

  if (!data.position || !data.position.trim()) {
    errors.push('Vị trí là bắt buộc');
  }

  if (!data.hour_type || !data.hour_type.trim()) {
    errors.push('Khung giờ là bắt buộc');
  }

  return {
    valid: errors.length === 0,
    errors
  };
}

// Enhanced validation that includes payrate checking
export function validateTimesheetFormDataWithPayrate(
  data: TimesheetFormData,
  formOptions: TimesheetFormOptions
): { valid: boolean; errors: string[] } {
  const baseValidation = validateTimesheetFormData(data);
  const errors = [...baseValidation.errors];

  // Check if the payrate is 0 (slot is disabled)
  if (formOptions.ratePreview === 0) {
    errors.push('Không thể tạo bản ghi cho khung giờ cấm (mức lương = 0)');
  }

  return {
    valid: errors.length === 0,
    errors
  };
}
