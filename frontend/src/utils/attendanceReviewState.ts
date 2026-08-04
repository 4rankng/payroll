import type { AdminAttendanceResponse } from "@/types/api/attendance.types";

type AttendanceReviewState = Pick<AdminAttendanceResponse, "status" | "review_action" | "quota_credited_at">;

export function needsAttendanceApprovalRepair(attendance: AttendanceReviewState): boolean {
  return attendance.review_action === "approved" && attendance.status === "rejected";
}

export function getAttendanceOperationalStatus(attendance: AttendanceReviewState): string {
  if (attendance.review_action === "approved" && !needsAttendanceApprovalRepair(attendance)) {
    return "completed";
  }
  if (attendance.review_action === "rejected") {
    return "rejected";
  }
  return attendance.status;
}

export function canApproveAttendance(attendance: AttendanceReviewState): boolean {
	return attendance.status !== "completed" &&
		(attendance.review_action !== "approved" || needsAttendanceApprovalRepair(attendance));
}

export function canRejectAttendance(attendance: AttendanceReviewState): boolean {
  return attendance.status !== "rejected" &&
    attendance.review_action !== "rejected" &&
    attendance.review_action !== "approved" &&
    attendance.quota_credited_at == null;
}

export function getAttendanceReviewStatusLabel(attendance: AttendanceReviewState): string | null {
  if (needsAttendanceApprovalRepair(attendance)) return "Cần duyệt lại";
  if (attendance.review_action === "approved") return "Đã duyệt";
  if (attendance.review_action === "rejected") return "Đã từ chối";
  return null;
}
