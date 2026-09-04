import type { ImportError } from '@/types/api/timesheet.types';

const TECHNICAL_VALUE_PATTERN =
  /^(?:id\s+\d+)\b|\b(?:onepay|9pay|provider|error|failed|internal|timeout|employee_id)\b|[_{}[\]=]/iu;
const ISO_DATE_PATTERN = /\b(\d{4})-(\d{2})-(\d{2})\b/u;

export function getSafeImportErrorReason(reason: string): string {
  const normalized = reason.trim().toLocaleLowerCase('vi');

  if (
    normalized.includes('phê duyệt') ||
    normalized.includes('đã duyệt') ||
    normalized.includes('approved')
  ) {
    return 'Bảng chấm công đã được phê duyệt';
  }
  if (
    normalized.includes('trùng') ||
    normalized.includes('đã có') ||
    normalized.includes('tồn tại') ||
    normalized.includes('duplicate')
  ) {
    return 'Dữ liệu đã tồn tại';
  }
  if (normalized.includes('không tìm thấy') && normalized.includes('nhân viên')) {
    return 'Không tìm thấy nhân viên';
  }
  if (normalized.includes('không thể tạo nhân viên')) {
    return 'Không thể tạo hồ sơ nhân viên';
  }
  if (normalized.includes('không tìm thấy mức lương')) {
    return 'Chưa cấu hình mức lương phù hợp cho ca làm việc';
  }
  if (normalized.includes('bảng lương') || normalized.includes('payrate')) {
    return 'Chưa cấu hình bảng lương cho dự án';
  }
  if (
    normalized.includes('ngày trong tương lai') ||
    normalized.includes('ngày chấm công chưa đến')
  ) {
    return 'Ngày chấm công chưa đến';
  }
  if (normalized.includes('thanh toán') || normalized.includes('paid')) {
    return 'Bảng chấm công đã thanh toán';
  }
  if (normalized.includes('không thể tạo bảng chấm công')) {
    return 'Không thể tạo bảng chấm công';
  }
  if (normalized.includes('thiếu')) {
    return 'Thiếu dữ liệu bắt buộc';
  }
  if (normalized.includes('không hợp lệ')) {
    return 'Dữ liệu không hợp lệ';
  }
  // Operator-facing file-level guidance from the backend (e.g. every hour in
  // the file falls outside the chosen month): already clean Vietnamese that
  // names the data range and the fix — pass it through instead of masking it
  // behind a generic category label.
  if (normalized.includes('ngoài tháng')) {
    return reason.trim();
  }

  return 'Không thể xử lý dòng dữ liệu này';
}

function getSafeEmployeeLabel(employee: unknown): string {
  if (typeof employee !== 'string') return '';
  const normalized = employee.trim();
  if (
    normalized.length > 120 ||
    TECHNICAL_VALUE_PATTERN.test(normalized)
  ) {
    return '';
  }
  return normalized;
}

/** Extract the first ISO date in a reason and format it as DD/MM/YYYY. */
function extractErrorDate(reason: string): string | undefined {
  const dateMatch = reason.match(ISO_DATE_PATTERN);
  if (!dateMatch) return undefined;
  return `${dateMatch[3]}/${dateMatch[2]}/${dateMatch[1]}`;
}

/**
 * Parse the JSON-encoded error_detail string from a PartnerImportFile.
 * Returns an empty array if the detail is missing, empty, or invalid JSON.
 * The reason is normalized to a stable per-category message (so identical
 * errors can be grouped); the affected date, when present in the raw reason,
 * is returned separately on `date`.
 */
export function parseImportErrors(detail?: string | null): ImportError[] {
  if (!detail) return [];
  try {
    const parsed: unknown = JSON.parse(detail);
    if (!Array.isArray(parsed)) return [];
    return parsed.flatMap((value): ImportError[] => {
      if (
        !value ||
        typeof value !== 'object' ||
        typeof (value as { reason?: unknown }).reason !== 'string'
      ) {
        return [];
      }
      const item = value as {
        row?: unknown;
        employee?: unknown;
        reason: string;
      };
      return [{
        row: typeof item.row === 'number' ? item.row : 0,
        employee: getSafeEmployeeLabel(item.employee),
        reason: getSafeImportErrorReason(item.reason),
        date: extractErrorDate(item.reason),
      }];
    });
  } catch {
    return [];
  }
}

export interface GroupedImportError {
  employee: string;
  reason: string;
  count: number;
  /** Unique affected dates (DD/MM/YYYY), in first-seen order. */
  dates: string[];
  /** Unique row numbers, in first-seen order (0 excluded). */
  rows: number[];
}

/**
 * Collapse parsed errors into one group per (employee, reason) pair so the UI
 * renders "reason (các ngày …)" once instead of one identical line per row.
 */
export function groupImportErrors(errors: ImportError[]): GroupedImportError[] {
  const groups = new Map<string, GroupedImportError>();
  for (const error of errors) {
    const key = `${error.employee}${error.reason}`;
    const group = groups.get(key) ?? {
      employee: error.employee,
      reason: error.reason,
      count: 0,
      dates: [],
      rows: [],
    };
    group.count += 1;
    if (error.date && !group.dates.includes(error.date)) {
      group.dates.push(error.date);
    }
    if (error.row > 0 && !group.rows.includes(error.row)) {
      group.rows.push(error.row);
    }
    groups.set(key, group);
  }
  return Array.from(groups.values());
}

const DETAIL_LIST_CAP = 5;

/**
 * Build the context suffix for a grouped error, e.g. " (các ngày 25/06/2026, 26/06/2026)".
 * Lists are capped; the row count carries the scale once the list is truncated.
 */
export function describeGroupedError(group: GroupedImportError): string {
  const parts: string[] = [];
  if (group.dates.length > 0) {
    const truncated = group.dates.length > DETAIL_LIST_CAP;
    const shown = group.dates.slice(0, DETAIL_LIST_CAP).join(', ');
    parts.push(
      `${group.dates.length === 1 ? 'ngày' : 'các ngày'} ${shown}${truncated ? '…' : ''}`,
    );
  } else if (group.rows.length > 0) {
    const truncated = group.rows.length > DETAIL_LIST_CAP;
    const shown = group.rows.slice(0, DETAIL_LIST_CAP).join(', ');
    parts.push(
      `${group.rows.length === 1 ? 'dòng' : 'các dòng'} ${shown}${truncated ? '…' : ''}`,
    );
  }
  if (
    group.count > 1 &&
    (group.dates.length > DETAIL_LIST_CAP ||
      (group.dates.length === 0 && group.rows.length === 0))
  ) {
    parts.push(`${group.count} dòng`);
  }
  return parts.length > 0 ? ` (${parts.join(', ')})` : '';
}
