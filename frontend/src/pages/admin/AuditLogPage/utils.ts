import { format } from "date-fns";
import { formatDateTimeFull } from '@/utils/formatters';
import { VIETNAMESE_AUDIT_LABELS } from '@/types/api/audit.types';

// ─── Action badge colours ────────────────────────────────────────────────────

export type ActionVariant =
  | 'green' | 'blue' | 'red' | 'yellow' | 'teal' | 'orange' | 'gray';

const ACTION_VARIANT_MAP: Record<string, ActionVariant> = {
  CREATE: 'green',
  BULK_CREATE: 'green',
  UPDATE: 'blue',
  APPROVE: 'blue',
  BULK_APPROVE: 'blue',
  SETTLE: 'blue',
  EXTERNAL_PAY: 'blue',
  DELETE: 'red',
  REJECT: 'red',
  BULK_REJECT: 'red',
  LOGIN: 'yellow',
  LOGOUT: 'yellow',
  CHANGE_PASSWORD: 'yellow',
  FAILED_LOGIN: 'red',
  IMPORT: 'teal',
  EXPORT: 'teal',
  DATA_EXPORT: 'teal',
  VIEW: 'gray',
  BULK_RESET: 'orange',
  SUSPICIOUS_ACTIVITY: 'red',
  TOKEN_REVOKED: 'orange',
  PERMISSION_DENIED: 'orange',
  BULK_OPERATION: 'orange',
};

// Uses semantic CSS variables — works on both light and dark themes
export const VARIANT_CLASSES: Record<ActionVariant, string> = {
  green:  'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-400 dark:border-emerald-800',
  blue:   'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-950/40 dark:text-blue-400 dark:border-blue-800',
  red:    'bg-red-50 text-red-700 border-red-200 dark:bg-red-950/40 dark:text-red-400 dark:border-red-800',
  yellow: 'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-950/40 dark:text-amber-400 dark:border-amber-800',
  teal:   'bg-teal-50 text-teal-700 border-teal-200 dark:bg-teal-950/40 dark:text-teal-400 dark:border-teal-800',
  orange: 'bg-orange-50 text-orange-700 border-orange-200 dark:bg-orange-950/40 dark:text-orange-400 dark:border-orange-800',
  gray:   'bg-muted text-muted-foreground border-border',
};

export function getActionVariant(action: string): ActionVariant {
  return ACTION_VARIANT_MAP[action] ?? 'gray';
}

export function getActionLabel(action: string): string {
  return (VIETNAMESE_AUDIT_LABELS.actions as Record<string, string>)[action] ?? action;
}

export function getEntityLabel(entityType: string): string {
  return (VIETNAMESE_AUDIT_LABELS.entities as Record<string, string>)[entityType] ?? entityType;
}

// ─── Metadata parsing ────────────────────────────────────────────────────────

export interface AuditFieldChange {
  before: unknown;
  after: unknown;
}

export type MetadataShape =
  | {
      kind: 'changed_fields';
      fields: Record<string, AuditFieldChange>;
      context: Record<string, unknown>;
    }
  | { kind: 'details'; data: Record<string, unknown> }
  | { kind: 'invalid'; raw: string }
  | { kind: 'empty' };

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function extractFieldChanges(
  value: unknown,
): { fields: Record<string, AuditFieldChange>; malformed: boolean } {
  if (!isRecord(value)) return { fields: {}, malformed: true };

  let malformed = false;
  const fields = Object.fromEntries(
    Object.entries(value).flatMap(([field, rawChange]) => {
      if (!isRecord(rawChange)) {
        malformed = true;
        return [];
      }

      const hasBefore = 'before' in rawChange || 'old' in rawChange;
      const hasAfter = 'after' in rawChange || 'new' in rawChange;
      if (!hasBefore || !hasAfter) {
        malformed = true;
        return [];
      }

      return [[field, {
        before: 'before' in rawChange ? rawChange.before : rawChange.old,
        after: 'after' in rawChange ? rawChange.after : rawChange.new,
      }]];
    }),
  );

  return { fields, malformed };
}

function extractLegacyPairs(data: Record<string, unknown>): Record<string, AuditFieldChange> {
  const fields: Record<string, AuditFieldChange> = {};

  for (const [key, before] of Object.entries(data)) {
    if (!key.startsWith('old_')) continue;
    const field = key.slice(4);
    const afterKey = `new_${field}`;
    if (afterKey in data) {
      fields[field] = { before, after: data[afterKey] };
      delete data[key];
      delete data[afterKey];
    }
  }

  return fields;
}

export function parseMetadata(raw: string | null | undefined): MetadataShape {
  if (!raw) return { kind: 'empty' };

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return { kind: 'invalid', raw };
  }

  if (!isRecord(parsed)) return { kind: 'invalid', raw };

  const data = { ...parsed };
  delete data.location;

  const legacyFields = extractLegacyPairs(data);
  if ('changed_fields' in data) {
    const { fields, malformed } = extractFieldChanges(data.changed_fields);
    delete data.changed_fields;
    const combinedFields = { ...fields, ...legacyFields };
    if (malformed) {
      return { kind: 'invalid', raw };
    }
    return { kind: 'changed_fields', fields: combinedFields, context: data };
  }

  if (Object.keys(legacyFields).length > 0) {
    return { kind: 'changed_fields', fields: legacyFields, context: data };
  }

  if (Object.keys(data).length === 0) return { kind: 'empty' };
  return { kind: 'details', data };
}

// Extract location from any metadata shape
export function extractLocation(raw: string | null | undefined): { country: string; city: string; region: string } | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (parsed?.location && typeof parsed.location === 'object') {
      const loc = parsed.location as Record<string, string>;
      if (loc.country || loc.city) {
        return { country: loc.country ?? '', city: loc.city ?? '', region: loc.region ?? '' };
      }
    }
  } catch {
    // ignore
  }
  return null;
}

// Extract the login identifier (what the user typed) from login metadata
export function extractLoginIdentifier(raw: string | null | undefined): string | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    // failed login uses attempted_identifier, successful login uses login_identifier
    return (parsed?.attempted_identifier as string) || (parsed?.login_identifier as string) || null;
  } catch {
    return null;
  }
}

// ─── Formatting helpers ──────────────────────────────────────────────────────

export function formatVND(value: unknown): string {
  const num = typeof value === 'number' ? value : Number(value);
  if (isNaN(num)) return String(value);
  return new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(num);
}

export function formatFieldName(key: string): string {
  const label = AUDIT_FIELD_LABELS[key];
  if (label) return label;
  return key
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function formatValue(value: unknown): string {
  if (value === null || value === undefined) return 'Không có';
  if (value === '') return 'Trống';
  if (typeof value === 'boolean') return value ? 'Có' : 'Không';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

const AUDIT_FIELD_LABELS: Record<string, string> = {
  account: 'Tài khoản',
  address: 'Địa chỉ',
  amount: 'Số tiền',
  approver_id: 'Người duyệt',
  bank_account_name: 'Tên chủ tài khoản',
  bank_account_number: 'Số tài khoản ngân hàng',
  bank_code: 'Mã ngân hàng',
  bank_id: 'Ngân hàng',
  branch_name: 'Tên chi nhánh',
  cccd: 'CCCD',
  client_name: 'Khách hàng',
  code: 'Mã',
  completed: 'Thành công',
  count: 'Số lượng',
  created_by: 'Người tạo',
  date: 'Ngày',
  date_of_birth: 'Ngày sinh',
  effective_date: 'Ngày hiệu lực',
  email: 'Email',
  employee_id: 'Nhân viên',
  employee_name: 'Tên nhân viên',
  failed: 'Thất bại',
  file_name: 'Tên tệp',
  filename: 'Tên tệp',
  fullname: 'Họ và tên',
  hours_worked: 'Số giờ làm',
  method: 'Phương thức',
  mobile: 'Số điện thoại',
  name: 'Tên',
  notes: 'Ghi chú',
  pay_rate: 'Mức lương',
  payment_status: 'Trạng thái thanh toán',
  project_id: 'Dự án',
  project_name: 'Tên dự án',
  reason: 'Lý do',
  record_count: 'Số bản ghi',
  reference: 'Mã tham chiếu',
  role: 'Vai trò',
  schedule_id: 'Mã biểu phí',
  status: 'Trạng thái',
  summary: 'Giá trị',
  total: 'Tổng số',
  total_amount: 'Tổng tiền',
  transaction_type: 'Loại giao dịch',
  username: 'Tên đăng nhập',
};

const MONEY_FIELDS = new Set([
  'amount',
  'outstanding_principal',
  'principal_amount',
  'pay_rate',
  'settlement_amount',
  'total_amount',
  'total_interest_paid',
]);

export function formatAuditValue(key: string, value: unknown): string {
  if (value === null || value === undefined || value === '') return formatValue(value);
  return MONEY_FIELDS.has(key) ? formatVND(value) : formatValue(value);
}

export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) return 'vừa xong';
  if (diffMin < 60) return `${diffMin} phút trước`;
  if (diffHour < 24) return `${diffHour} giờ trước`;
  if (diffDay < 7) return `${diffDay} ngày trước`;
  return format(date, 'dd/MM/yyyy');
}

export function formatDateTime(dateStr: string): string {
  return formatDateTimeFull(dateStr);
}
