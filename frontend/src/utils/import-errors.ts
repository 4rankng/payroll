import type { ImportError } from '@/types/api/timesheet.types';

const TECHNICAL_VALUE_PATTERN =
  /^(?:id\s+\d+)\b|\b(?:onepay|9pay|provider|error|failed|internal|timeout|employee_id)\b|[_{}[\]=]/iu;

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
  if (normalized.includes('thiếu')) {
    return 'Thiếu dữ liệu bắt buộc';
  }
  if (normalized.includes('không hợp lệ')) {
    return 'Dữ liệu không hợp lệ';
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

/**
 * Parse the JSON-encoded error_detail string from a PartnerImportFile.
 * Returns an empty array if the detail is missing, empty, or invalid JSON.
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
      }];
    });
  } catch {
    return [];
  }
}
