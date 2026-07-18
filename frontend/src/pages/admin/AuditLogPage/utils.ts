import { format } from "date-fns";
import { formatDateTimeFull } from '@/utils/formatters';
import { VIETNAMESE_AUDIT_LABELS } from '@/types/api/audit.types';

// ─── Action badge colours ────────────────────────────────────────────────────

export type ActionVariant =
  | 'green' | 'blue' | 'red' | 'yellow' | 'purple' | 'orange' | 'gray';

const ACTION_VARIANT_MAP: Record<string, ActionVariant> = {
  CREATE: 'green',
  BULK_CREATE: 'green',
  UPDATE: 'blue',
  APPROVE: 'blue',
  BULK_APPROVE: 'blue',
  SETTLE: 'blue',
  DELETE: 'red',
  REJECT: 'red',
  BULK_REJECT: 'red',
  LOGIN: 'yellow',
  LOGOUT: 'yellow',
  CHANGE_PASSWORD: 'yellow',
  FAILED_LOGIN: 'red',
  IMPORT: 'purple',
  EXPORT: 'purple',
  DATA_EXPORT: 'purple',
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
  purple: 'bg-purple-50 text-purple-700 border-purple-200 dark:bg-purple-950/40 dark:text-purple-400 dark:border-purple-800',
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
  | { kind: 'changed_fields'; fields: Record<string, AuditFieldChange> }
  | { kind: 'auth'; data: Record<string, unknown> }
  | { kind: 'bulk'; data: Record<string, unknown> }
  | { kind: 'financial'; data: Record<string, unknown> }
  | { kind: 'identity'; data: Record<string, unknown> }
  | { kind: 'relationship'; data: Record<string, unknown> }
  | { kind: 'raw'; data: Record<string, unknown> }
  | { kind: 'empty' };

export function parseMetadata(raw: string | null | undefined): MetadataShape {
  if (!raw) return { kind: 'empty' };

  let parsed: Record<string, unknown>;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return { kind: 'empty' };
  }

  if (!parsed || typeof parsed !== 'object') return { kind: 'empty' };

  if ('changed_fields' in parsed && typeof parsed.changed_fields === 'object' && parsed.changed_fields !== null) {
    const fields = Object.fromEntries(
      Object.entries(parsed.changed_fields).flatMap(([field, value]) => {
        if (!value || typeof value !== 'object') return [];

        const change = value as Record<string, unknown>;
        return [[field, {
          before: 'before' in change ? change.before : change.old,
          after: 'after' in change ? change.after : change.new,
        }]];
      }),
    );

    return {
      kind: 'changed_fields',
      fields,
    };
  }

  // Auth events: login (success or failure), logout, password change
  if ('success' in parsed || 'reason' in parsed || 'method' in parsed ||
      'attempted_identifier' in parsed || 'login_identifier' in parsed) {
    return { kind: 'auth', data: parsed };
  }

  if ('count' in parsed || 'total' in parsed || 'completed' in parsed) {
    return { kind: 'bulk', data: parsed };
  }

  if ('amount' in parsed || 'total_amount' in parsed || 'settlement_amount' in parsed) {
    return { kind: 'financial', data: parsed };
  }

  if (('project_id' in parsed || 'project_name' in parsed) && ('employee_id' in parsed || 'employee_name' in parsed)) {
    return { kind: 'relationship', data: parsed };
  }

  if ('username' in parsed || 'fullname' in parsed || 'cccd' in parsed || 'email' in parsed) {
    return { kind: 'identity', data: parsed };
  }

  return { kind: 'raw', data: parsed };
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
  return key
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'boolean') return value ? 'Có' : 'Không';
  if (typeof value === 'object') return JSON.stringify(value);
  return String(value);
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
