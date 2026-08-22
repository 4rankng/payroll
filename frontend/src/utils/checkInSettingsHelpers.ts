import type {
  CheckInConfigurationEmployee,
  CheckInConfigurationStatus,
} from "@/types/api/project-employee.types";
import { formatDateTime } from "@/utils/formatters";

export const DEFAULT_CHECK_IN_PAGE_SIZE = 20;

export type CheckInEmployeeState =
  | "pending"
  | "disabled"
  | "used"
  | "unused";

export interface CheckInStatusFilter {
  value: CheckInConfigurationStatus;
  label: string;
  count?: number;
}

export function formatCheckInMonth(month?: string): string {
  if (!month) return "tháng hiện tại";
  const [year, monthNumber] = month.split("-");
  return `tháng ${monthNumber}/${year}`;
}

export function formatLastCheckIn(value?: string | null): string {
  if (!value) return "Chưa có";
  const normalized = value.includes("T") ? value : `${value.replace(" ", "T")}+07:00`;
  const parsed = new Date(normalized);
  if (Number.isNaN(parsed.getTime())) return value;
  return formatDateTime(parsed);
}

export function getCheckInEmployeeState(
  employee: CheckInConfigurationEmployee,
): CheckInEmployeeState {
  if (employee.pending_check_in_enable) return "pending";
  if (!employee.check_in_enabled) return "disabled";
  if (employee.attendance_count > 0) return "used";
  return "unused";
}

export function buildCheckInStatusFilters(summary?: {
  enabled: number;
  active: number;
  inactive: number;
  pending: number;
}): CheckInStatusFilter[] {
  return [
    { value: "all", label: "Tất cả" },
    { value: "enabled", label: "Đang bật", count: summary?.enabled },
    { value: "active", label: "Đã điểm danh", count: summary?.active },
    { value: "inactive", label: "Chưa điểm danh", count: summary?.inactive },
    { value: "pending", label: "Chờ kích hoạt", count: summary?.pending },
  ];
}
