// 3-Level Payrate API Types
// Structure: Position → Day Type → Hour Type → Rate

// Core payrate structure based on API documentation
export type DayType = 'weekday' | 'weekend' | 'holiday' | 'ngày thường' | 'ngày nghỉ' | 'ngày lễ';
export type PayrateStatus = 'active' | 'inactive';
export type HourRange = string; // e.g., "08:00-17:00", "17:00-22:00"

export interface DayRateConfiguration {
  [hourRange: string]: number; // Hour range -> rate in VND
}

export interface PositionRateConfiguration {
  weekday: DayRateConfiguration;
  weekend: DayRateConfiguration;
  holiday: DayRateConfiguration;
}

// Flexible rate category type for Vietnamese structure
export type RateCategory = {
  [key: string]: number | RateCategory;
};

export interface PayrateStructure {
  [position: string]: PositionRateConfiguration | RateCategory; // Support both formats
}

// Main payrate entity
export interface PayRate {
  id: number;
  project_id: number;
  rates: PayrateStructure;
  fromDate: string;
  toDate: string | null;
  /** Day after the project's most recent paid timesheet work date — earliest allowed start. */
  earliest_effective_from?: string;
  /** True when the start date has no legal move (paid floor below, linked timesheets above) — render read-only. */
  from_date_locked?: boolean;
  created_by: number;
  created_at: string;
  updated_at: string;
}

// Request types
export interface CreatePayRateRequest {
  rates: PayrateStructure;
  effective_from: string;
  effective_to?: string;
}

export interface UpdatePayRateRequest {
  rates: PayrateStructure;
  effective_from: string;
  effective_to?: string;
}

export interface PayRateListParams {
  project_id?: number;
  page?: number;
  pageSize?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

// Response wrapper types
export interface PayRateListResponse {
  status: string;
  message: string;
  data: PayRate[];
  pagination?: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalRecords: number;
  };
}

export interface PayRateResponse {
  status: string;
  message: string;
  data: PayRate;
}

export interface PayRateActionResponse {
  status: string;
  message: string;
  data: {
    id: number;
    approved_by?: number;
    approved_at?: string;
    rejection_reason?: string;
  };
}

export interface DeletePayRateResponse {
  status: string;
  message: string;
  data: null;
}

// Error types specific to pay rates
export interface PayRateError {
  code: 'DATE_OVERLAP' | 'INVALID_RATE_CONFIG' | 'INSUFFICIENT_PERMISSIONS' |
        'RATE_IN_USE' | 'HISTORICAL_RATE_MODIFICATION' | 'FUTURE_DATE_REQUIRED' |
        'APPROVAL_ALREADY_PROCESSED' | 'VALIDATION_ERROR' | 'RATE_NOT_FOUND';
  message: string;
  details?: {
    conflicting_rate_id?: number;
    affected_timesheets?: number;
    minimum_rate?: number;
    maximum_rate?: number;
    overlap_period?: {
      start: string;
      end: string;
    };
    missing_position?: string;
    missing_day_type?: DayType;
    missing_hour_type?: string;
  };
}

// Status calculation helper
export function getPayrateStatus(payrate: PayRate): PayrateStatus {
  const today = new Date().toISOString().split('T')[0];
  const fromDate = payrate.fromDate;
  const toDate = payrate.toDate;

  // If current date is within range, it's active
  if (fromDate <= today && (!toDate || toDate >= today)) {
    return 'active';
  }

  // Otherwise, it's inactive (future or past)
  return 'inactive';
}

// Vietnamese translations for API status values
export const VIETNAMESE_PAYRATE_LABELS = {
  statuses: {
    active: 'Đang dùng',
    inactive: 'Không hoạt động'
  },
  fields: {
    effective_from: 'Có hiệu lực từ',
    effective_to: 'Có hiệu lực đến',
    rates: 'Cấu hình mức lương'
  },
  dayTypes: {
    'ngày thường': 'Ngày thường',
    'ngày nghỉ': 'Ngày nghỉ',
    'ngày lễ': 'Ngày lễ'
  }
} as const;

export const DEFAULT_POSITIONS = [
  'phổ thông',
  'kinh nghiệm',
  'học việc',
  'thợ điện',
  'thợ cơ khí',
  'thợ hàn',
  'công nhân',
  'kỹ thuật',
  'thủ kho',
  'tổ trưởng',
  'quản đốc',
  'giám sát'
] as const;

export const DEFAULT_POSITION = 'phổ thông';

// Common day and hour types
export const COMMON_DAY_TYPES = [
  'ngày thường',
  'ngày nghỉ',
  'ngày lễ'
] as const;

export const COMMON_HOUR_TYPES = {
  CATEGORIES: [
    'ca ngày',
    'ca đêm',
    'tăng ca'
  ],
TIME_RANGES: [
  '08:00-20:00',   // common 12-hour two-shift pattern reported at Samsung plants. :contentReference[oaicite:0]{index=0}
  '20:00-08:00',   // night / second 12-hour shift (continuous production). :contentReference[oaicite:1]{index=1}
  '08:00-17:00',   // standard day shift (8h + typical 1h break) used for admin/short-line operations. :contentReference[oaicite:2]{index=2}
  '06:00-14:00',   // early morning / first 8-hour manufacturing shift (used in 3-shift setups). :contentReference[oaicite:3]{index=3}
  '14:00-22:00',   // afternoon/evening 8-hour shift (rotating 3-shift systems). :contentReference[oaicite:4]{index=4}
  '22:00-06:00'    // legally defined night window in Vietnam (night premium applies). :contentReference[oaicite:5]{index=5}
]

} as const;

// Utility functions
export function getPositionsFromRates(rates: PayrateStructure): string[] {
  return Object.keys(rates);
}

export function getHourTypesFromPosition(rates: PayrateStructure, position: string, dayType: DayType): string[] {
  const positionRates = rates[position];
  if (!positionRates) return [];

  const dayRates = positionRates[dayType];
  if (!dayRates) return [];

  return Object.keys(dayRates);
}

export function extractPositionsFromPayrates(payrates: PayRate[] | PayRate): string[] {
  if (!Array.isArray(payrates)) {
    if (!payrates || !payrates.rates) {
      return [];
    }
    return Object.keys(payrates.rates);
  }

  if (!payrates || payrates.length === 0) {
    return [];
  }

  const activePayrate = payrates.find(rate => !rate.toDate || new Date(rate.toDate) > new Date());

  if (!activePayrate || !activePayrate.rates) {
    return [];
  }

  return Object.keys(activePayrate.rates);
}

export function hasPosition(payrates: PayRate[] | PayRate, position: string): boolean {
  const positions = extractPositionsFromPayrates(payrates);
  return positions.includes(position);
}

export function isValidVietnamesePayrateStructure(payrate: PayRate): boolean {
  if (!payrate || !payrate.rates) {
    return false;
  }

  const positions = Object.keys(payrate.rates);
  if (positions.length === 0) {
    return false;
  }

  for (const position of positions) {
    const positionRates = payrate.rates[position];
    if (typeof positionRates === 'object' && positionRates !== null) {
      const dayTypes = Object.keys(positionRates);
      const hasVietnameseDayTypes = dayTypes.some(dayType =>
        COMMON_DAY_TYPES.some(dt => dt === dayType)
      );
      if (hasVietnameseDayTypes) {
        return true;
      }
    }
  }

  return false;
}

export function validatePayrateStructure(rates: PayrateStructure): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (!rates || typeof rates !== 'object') {
    errors.push('Cấu hình payrate phải là object');
    return { valid: false, errors };
  }

  const positions = Object.keys(rates);
  if (positions.length === 0) {
    errors.push('Phải có ít nhất một vị trí');
    return { valid: false, errors };
  }

  for (const position of positions) {
    const dayConfig = rates[position];
    if (!dayConfig || typeof dayConfig !== 'object') {
      errors.push(`Cấu hình cho vị trí "${position}" không hợp lệ`);
      continue;
    }

    const dayTypes: DayType[] = ['weekday', 'weekend', 'holiday'];
    for (const dayType of dayTypes) {
      const hourConfig = dayConfig[dayType];
      if (!hourConfig || typeof hourConfig !== 'object') {
        errors.push(`Cấu hình cho "${dayType}" của vị trí "${position}" không hợp lệ`);
        continue;
      }

      const hourTypes = Object.keys(hourConfig);
      if (hourTypes.length === 0) {
        errors.push(`"${dayType}" của vị trí "${position}" phải có ít nhất một khung giờ`);
      }

      // Validate rate values
      for (const hourType of hourTypes) {
        const rate = hourConfig[hourType];
        if (typeof rate !== 'number' || rate < 0) {
          errors.push(`Mức lương cho "${hourType}" của "${dayType}" - vị trí "${position}" phải là số không âm`);
        }
      }
    }
  }

  return {
    valid: errors.length === 0,
    errors
  };
}
