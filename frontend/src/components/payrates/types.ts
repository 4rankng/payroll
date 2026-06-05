// 3-Level Payrate Component Types
// Structure: Position → Day Type → Hour Type → Rate

// Import core types from API
import type {
  PayrateStructure,
  DayType,
  PayrateStatus,
  RateCategory
} from '@/types/api/payrate.types';
import { DEFAULT_POSITIONS } from '@/types/api/payrate.types';

// Re-export core types for component use
export type {
  PayrateStructure,
  DayType,
  PayrateStatus,
  RateCategory
} from '@/types/api/payrate.types';

// Component-level structure-type discriminator
export type PayrateStructureType = 'hourly' | 'category';

// Hour/day configuration shapes used by components
export interface HourConfiguration {
  [hourType: string]: number;
}

export interface DayConfiguration {
  [dayType: string]: HourConfiguration;
}

// Position -> DayConfiguration map (used by templateGenerator)
export type PayrateConfiguration = Record<string, DayConfiguration>;

// Configuration for a single position selected in the UI
export interface PositionConfig {
  positions: string[];
  rateType: PayrateStructureType;
  rates: PayrateConfiguration;
}

export function getCategoryTree(rates: PayrateStructure) {
  return Object.keys(rates);
}

// Additional missing exports
export interface CategoryTreeNode {
  id: string;
  label: string;
  children?: CategoryTreeNode[];
}

export interface FlexiblePayrateConfig extends PayrateConfig {
  flexible?: boolean;
}

export type EditorMode = 'create' | 'edit';

// Main payrate config interface for components
export interface PayrateConfig {
  id?: number;
  project_id: number;
  rates: PayrateStructure;
  fromDate: string;
  toDate?: string | null;
  status: PayrateStatus;
  created_by?: number;
  created_at?: string;
  updated_at?: string;
}

// Request/Response types for component use
export interface CreatePayrateRequest {
  project_id: number;
  rates: PayrateStructure;
  effective_from: string;
  effective_to?: string;
}

export interface UpdatePayrateRequest {
  project_id?: number;
  rates?: PayrateStructure;
  effective_from?: string;
  effective_to?: string;
}

export interface PayrateResponse {
  success: boolean;
  message: string;
  data: PayrateConfig;
}

export interface PayrateListResponse {
  success: boolean;
  message: string;
  data: PayrateConfig[];
  pagination?: {
    page: number;
    page_size: number;
    total_pages: number;
    total_records: number;
  };
}

// UI Helper types
export interface MatrixCell {
  position: string;
  dayType: DayType;
  hourType: string;
  rate: number;
  path: string; // "position.dayType.hourType"
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  warnings?: string[];
}

// Component props types
export interface PayrateMatrixEditorProps {
  rates: PayrateStructure;
  onChange: (rates: PayrateStructure) => void;
  readOnly?: boolean;
  validation?: ValidationResult;
}

export interface PositionManagerProps {
  positions: string[];
  onPositionsChange: (positions: string[]) => void;
  rates: PayrateStructure;
  onRatesChange: (rates: PayrateStructure) => void;
}

// Status labels for UI
export const STATUS_LABELS: Record<PayrateStatus, string> = {
  active: 'Đang dùng',
  inactive: 'Không hoạt động'
} as const;

// Utility functions for components
export const formatCurrency = (amount: number): string => {
  return new Intl.NumberFormat('vi-VN').format(amount) + ' VND';
};

export const parseCurrency = (value: string): number => {
  return parseInt(value.replace(/[^\d]/g, '')) || 0;
};

// Get all positions from rates structure
export function getPositionsFromRates(rates: PayrateStructure): string[] {
  return Object.keys(rates);
}

// Get all hour types from a position and day type
export function getHourTypesFromPosition(
  rates: PayrateStructure,
  position: string,
  dayType: DayType
): string[] {
  const positionRates = rates[position];
  if (!positionRates) return [];

  const dayRates = positionRates[dayType];
  if (!dayRates) return [];

  return Object.keys(dayRates);
}

// Get all unique hour types across all positions and day types
export function getAllHourTypes(rates: PayrateStructure): string[] {
  const hourTypes = new Set<string>();

  Object.values(rates).forEach(dayConfig => {
    Object.values(dayConfig).forEach(hourConfig => {
      Object.keys(hourConfig).forEach(hourType => {
        hourTypes.add(hourType);
      });
    });
  });

  return Array.from(hourTypes).sort();
}

// Strict HH:MM-HH:MM validation (24-hour clock)
export const TIME_RANGE_RE = /^([01]\d|2[0-3]):[0-5]\d-([01]\d|2[0-3]):[0-5]\d$/;

export function isTimeRange(key: string): boolean {
  return TIME_RANGE_RE.test(key);
}

// Try to suggest a corrected value from a partial/invalid time-range input.
export function suggestTimeRange(raw: string): string | null {
  const parts = raw.split('-');
  if (parts.length !== 2) return null;
  const fixed = parts.map(p => {
    const [h, m] = p.split(':');
    if (h === undefined || m === undefined) return null;
    let hh = parseInt(h, 10);
    let mm = parseInt(m, 10);
    if (isNaN(hh) || isNaN(mm)) return null;
    hh = Math.min(23, Math.max(0, hh));
    mm = Math.min(59, Math.max(0, mm));
    return `${String(hh).padStart(2, '0')}:${String(mm).padStart(2, '0')}`;
  });
  if (fixed.some(f => !f)) return null;
  const result = `${fixed[0]}-${fixed[1]}`;
  return TIME_RANGE_RE.test(result) ? result : null;
}

// Centralized day-type constants
export const ALL_DAY_TYPES: DayType[] = ['ngày thường', 'ngày nghỉ', 'ngày lễ'];
export const FLEXIBLE_DAY_TYPES: DayType[] = ['ngày thường'];

// Common positions (re-export of API default positions)
export const COMMON_POSITIONS: readonly string[] = DEFAULT_POSITIONS;

// Position rate templates
export interface PositionRateTemplate {
  type: PayrateStructureType;
  name: string;
  description: string;
  baseStructure: DayConfiguration;
}

const HOURLY_BASE_STRUCTURE: DayConfiguration = {
  'ngày thường': {
    'ca ngày': 30000,
    'ca đêm': 39000,
    'tăng ca': 45000,
  },
  'ngày nghỉ': {
    'ca ngày': 60000,
    'ca đêm': 78000,
    'tăng ca': 90000,
  },
  'ngày lễ': {
    'ca ngày': 90000,
    'ca đêm': 117000,
    'tăng ca': 135000,
  },
};

const CATEGORY_BASE_STRUCTURE: DayConfiguration = {
  'ngày thường': {
    'phổ thông': 30000,
    'kinh nghiệm': 40000,
    'chuyên gia': 60000,
  },
  'ngày nghỉ': {
    'phổ thông': 50000,
    'kinh nghiệm': 65000,
    'chuyên gia': 90000,
  },
  'ngày lễ': {
    'phổ thông': 75000,
    'kinh nghiệm': 100000,
    'chuyên gia': 150000,
  },
};

export const POSITION_RATE_TEMPLATES: PositionRateTemplate[] = [
  {
    type: 'hourly',
    name: 'Theo khung giờ',
    description: 'Cấu hình lương theo từng ca làm việc (ca ngày, ca đêm, tăng ca).',
    baseStructure: HOURLY_BASE_STRUCTURE,
  },
  {
    type: 'category',
    name: 'Theo danh mục',
    description: 'Cấu hình lương theo cấp bậc / danh mục công việc.',
    baseStructure: CATEGORY_BASE_STRUCTURE,
  },
];

// Create empty day configuration with default hour types
export function createEmptyDayConfiguration(hourTypes: string[]): DayConfiguration {
  const dayTypes: DayType[] = ['ngày thường', 'ngày nghỉ', 'ngày lễ'];
  const config: DayConfiguration = {} as DayConfiguration;

  dayTypes.forEach(dayType => {
    config[dayType] = {};
    hourTypes.forEach(hourType => {
      config[dayType][hourType] = 0;
    });
  });

  return config;
}

// Add new position to rates structure
export function addPositionToRates(
  rates: PayrateStructure,
  position: string,
  template?: DayConfiguration
): PayrateStructure {
  if (rates[position]) {
    return rates; // Position already exists
  }

  const newRates = { ...rates };

  if (template) {
    newRates[position] = template;
  } else {
    // Use existing hour types from other positions
    const existingHourTypes = getAllHourTypes(rates);
    const hourTypes = existingHourTypes.length > 0
      ? existingHourTypes
      : ['ca ngày', 'ca đêm', 'tăng ca']; // Default hour types

    newRates[position] = createEmptyDayConfiguration(hourTypes);
  }

  return newRates;
}

// Remove position from rates structure
export function removePositionFromRates(
  rates: PayrateStructure,
  position: string
): PayrateStructure {
  const newRates = { ...rates };
  delete newRates[position];
  return newRates;
}

// Update rate value at specific path
export function updateRateValue(
  rates: PayrateStructure,
  position: string,
  dayType: DayType,
  hourType: string,
  value: number
): PayrateStructure {
  const newRates = { ...rates };

  if (!newRates[position]) {
    newRates[position] = createEmptyDayConfiguration([hourType]);
  }

  if (!newRates[position][dayType]) {
    newRates[position][dayType] = {};
  }

  newRates[position][dayType][hourType] = value;

  return newRates;
}

// Convert rates to matrix format for easier rendering
export function ratesToMatrix(rates: PayrateStructure): MatrixCell[] {
  const matrix: MatrixCell[] = [];

  Object.entries(rates).forEach(([position, dayConfig]) => {
    Object.entries(dayConfig).forEach(([dayType, hourConfig]) => {
      Object.entries(hourConfig).forEach(([hourType, rate]) => {
        matrix.push({
          position,
          dayType: dayType as DayType,
          hourType,
          rate,
          path: `${position}.${dayType}.${hourType}`
        });
      });
    });
  });

  return matrix;
}

// Create empty payrate structure for new flexible projects.
export function createFlexiblePayrateStructure(): PayrateStructure {
  return {
    'phổ thông': {
      'ngày thường': {},
    },
  };
}

// Create default payrate structure for new projects
export function createDefaultPayrateStructure(): PayrateStructure {
  const defaultPosition = 'phổ thông';
  const structure: PayrateStructure = {};

  structure[defaultPosition] = {
    'ngày thường': {
      'ca ngày': 30000,
      'ca đêm': 39000,
      'tăng ca': 45000
    },
    'ngày nghỉ': {
      'ca ngày': 60000,
      'ca đêm': 78000,
      'tăng ca': 90000
    },
    'ngày lễ': {
      'ca ngày': 90000,
      'ca đêm': 117000,
      'tăng ca': 135000
    }
  };

  return structure;
}

// Validation for flexible projects
export function validateFlexiblePayrateStructure(rates: PayrateStructure): ValidationResult {
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
    const hourConfig = dayConfig?.['ngày thường'] as Record<string, number> | undefined;

    if (!hourConfig || typeof hourConfig !== 'object') {
      errors.push(`Vị trí "${position}" chưa có ca làm việc`);
      continue;
    }

    const shifts = Object.keys(hourConfig);
    if (shifts.length === 0) {
      errors.push(`Vị trí "${position}" cần có ít nhất một ca làm việc`);
      continue;
    }

    for (const shift of shifts) {
      const rate = hourConfig[shift];
      if (typeof rate !== 'number' || rate < 0) {
        errors.push(`Mức lương cho ca "${shift}" của vị trí "${position}" phải là số không âm`);
      }
    }
  }

  return { valid: errors.length === 0, errors };
}

// Validation function for payrate structure
export function validatePayrateStructure(rates: PayrateStructure): ValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  if (!rates || typeof rates !== 'object') {
    errors.push('Cấu hình payrate phải là object');
    return { valid: false, errors, warnings };
  }

  const positions = Object.keys(rates);
  if (positions.length === 0) {
    errors.push('Phải có ít nhất một vị trí');
    return { valid: false, errors, warnings };
  }

  for (const position of positions) {
    const dayConfig = rates[position];
    if (!dayConfig || typeof dayConfig !== 'object') {
      errors.push(`Cấu hình cho vị trí "${position}" không hợp lệ`);
      continue;
    }

    const dayTypes: DayType[] = ['ngày thường', 'ngày nghỉ', 'ngày lễ'];
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
    errors,
    warnings
  };
}
