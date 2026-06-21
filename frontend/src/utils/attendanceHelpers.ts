import type { AttendanceRecord } from "@/services/attendance";

type SalaryFields = Pick<AttendanceRecord, "salary_status" | "earning_amount">;

/**
 * Whether salary was recorded for a shift. Treats an explicit backend
 * salary_status of "recorded" OR a positive earning_amount as recorded,
 * so the flag stays correct even when the status field is absent.
 * Shared by the check-in card and the attendance history card.
 */
export function isSalaryRecorded(att: SalaryFields): boolean {
  return att.salary_status === "recorded" || (att.earning_amount != null && att.earning_amount > 0);
}

/**
 * Whether a completed shift is missing its salary record (checked out but
 * no earnings captured) — drives the "Chưa ghi nhận lương" warning.
 */
export function isSalaryMissing(att: Pick<AttendanceRecord, "status" | "salary_status" | "earning_amount">): boolean {
  return att.status === "completed" && !isSalaryRecorded(att);
}
