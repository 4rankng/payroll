import { useId, useMemo, useState } from "react";
import { format } from "date-fns";
import {
  AlertTriangle,
  Calendar,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  History,
} from "lucide-react";
import { useAttendanceHistory } from "@/hooks/api/useAttendance";
import type { AttendanceRecord } from "@/services/attendance";
import {
  getAttendanceIssueSummary,
  isSalaryMissing,
  isSalaryRecorded,
  type AttendanceIssueDetail,
} from "@/utils/attendanceHelpers";
import {
  getAttendanceHistoryMonth,
  getNextAttendanceHistoryMonth,
  getPreviousAttendanceHistoryMonth,
} from "@/utils/attendanceHistoryMonth";
import { formatDate } from "@/utils/formatters";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

function formatTime(time: string | undefined | null, fallback = "--:--"): string {
  if (!time) return fallback;
  try {
    return format(new Date(time), "HH:mm");
  } catch {
    return fallback;
  }
}

interface EmployeeAttendanceHistoryCardProps {
  className?: string;
  style?: React.CSSProperties;
}

function HistoryIssueChips({ details }: { details: AttendanceIssueDetail[] }) {
  if (details.length === 0) return null;

  return (
    <div className="mt-2 grid grid-cols-[repeat(auto-fit,minmax(5rem,1fr))] gap-2">
      {details.map((detail) => (
        <div key={detail.label} className="rounded-lg border border-orange-100 bg-white/80 px-2.5 py-2">
          <p className="employee-type-label-caps whitespace-nowrap text-slate-500">
            {detail.label}
          </p>
          <p className="employee-type-inline-amount mt-1 text-orange-950">
            {detail.value}
          </p>
        </div>
      ))}
    </div>
  );
}

function AttendanceStatusPill({ status }: { status: AttendanceRecord["status"] }) {
  const statusStyle = {
    completed: "bg-emerald-50 text-emerald-700",
    rejected: "bg-orange-50 text-orange-700",
    orphaned: "bg-red-50 text-red-700",
    checked_in: "bg-sky-50 text-sky-700",
  }[status];
  const statusLabel = {
    completed: "Hoàn thành",
    rejected: "Đã từ chối",
    orphaned: "Thiếu tan ca",
    checked_in: "Đang làm",
  }[status];

  return (
    <span className={cn("employee-type-pill rounded-full px-2.5 py-1.5", statusStyle)}>
      {statusLabel}
    </span>
  );
}

function AttendanceHistoryRow({ attendance }: { attendance: AttendanceRecord }) {
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const detailId = useId();
  const salaryRecorded = isSalaryRecorded(attendance);
  const salaryMissing = isSalaryMissing(attendance);
  const needsDetail =
    attendance.status === "rejected" ||
    attendance.status === "orphaned" ||
    salaryMissing;
  const fallbackDescription =
    attendance.status === "rejected"
      ? "Ca làm này đã bị từ chối. Liên hệ quản lý nếu bạn cần kiểm tra lại."
      : attendance.status === "orphaned"
        ? "Ca làm thiếu giờ tan ca nên chưa thể ghi nhận đầy đủ."
        : "Ca đã hoàn thành nhưng tiền công chưa được ghi nhận.";
  const issue = getAttendanceIssueSummary(
    attendance.salary_reject_reason || attendance.salary_message,
    fallbackDescription
  );

  return (
    <article className="px-4 py-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="employee-type-card-title text-slate-950">
            Ngày {formatDate(attendance.date)}
          </p>
          <p className="employee-type-body-sm mt-1 text-slate-500">
            Vào {formatTime(attendance.check_in_time)}
            <span aria-hidden="true"> · </span>
            Tan {formatTime(attendance.check_out_time)}
          </p>
        </div>
        <AttendanceStatusPill status={attendance.status} />
      </div>

      <div className="mt-2.5 flex items-center justify-between gap-3 border-t border-slate-100 pt-2.5">
        <span className="employee-type-label text-slate-500">Tiền công</span>
        {salaryRecorded ? (
          <span className="employee-type-inline-amount text-emerald-700 tabular-nums">
            +{attendance.earning_amount?.toLocaleString("vi-VN")}đ
          </span>
        ) : (
          <span
            className={cn(
              "employee-type-body-sm font-semibold",
              salaryMissing ? "text-amber-700" : "text-slate-600"
            )}
          >
            {salaryMissing ? "Chưa ghi lương" : "Chưa có"}
          </span>
        )}
      </div>

      {needsDetail && (
        <div className="mt-2 border-t border-slate-100 pt-1">
          <button
            type="button"
            aria-expanded={isDetailOpen}
            aria-controls={detailId}
            onClick={() => setIsDetailOpen((current) => !current)}
            className="employee-type-action flex min-h-11 w-full items-center justify-between rounded-xl text-left text-orange-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-orange-500 focus-visible:ring-offset-2"
          >
            <span className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Chi tiết
            </span>
            <ChevronDown
              className={cn(
                "h-4 w-4 transition-transform duration-200",
                isDetailOpen && "rotate-180"
              )}
            />
          </button>
          {isDetailOpen && (
            <div id={detailId} className="rounded-xl bg-orange-50 px-3 py-3 text-orange-900">
              <p className="employee-type-strong">{issue.title}</p>
              <p className="employee-type-body-sm mt-1 text-orange-700">
                {issue.description}
              </p>
              <HistoryIssueChips details={issue.details} />
            </div>
          )}
        </div>
      )}
    </article>
  );
}

export function EmployeeAttendanceHistoryCard({ className, style }: EmployeeAttendanceHistoryCardProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedMonth, setSelectedMonth] = useState(() => new Date());
  const panelId = useId();
  const month = useMemo(() => getAttendanceHistoryMonth(selectedMonth), [selectedMonth]);
  const historyParams = useMemo(
    () => ({
      limit: 100,
      from_date: month.fromDate,
      to_date: month.toDate,
    }),
    [month.fromDate, month.toDate]
  );

  const { data: historyResponse, isLoading } = useAttendanceHistory(historyParams);
  const history = historyResponse?.data || [];
  const workdayCount = history.filter((attendance) => attendance.status === "completed").length;
  const warningCount = history.filter(
    (attendance) =>
      attendance.status === "rejected" ||
      attendance.status === "orphaned" ||
      isSalaryMissing(attendance)
  ).length;

  return (
    <div className={className ?? "overflow-hidden rounded-2xl bg-white"} style={style}>
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={panelId}
        onClick={() => setIsOpen((current) => !current)}
        className="w-full px-4 py-3.5 text-left transition-colors hover:bg-slate-50/70 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-emerald-500"
      >
        <span className="flex items-center justify-between gap-3">
          <span className="flex min-w-0 items-center gap-2.5">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-emerald-50 text-employee">
              <History className="h-4 w-4" />
            </span>
            <span className="min-w-0">
              <span className="employee-type-hero-title block text-slate-950">
                Lịch sử chấm công
              </span>
              <span className="employee-type-body-sm mt-0.5 block text-slate-500">
                {month.label}
              </span>
            </span>
          </span>
          <ChevronDown
            className={cn(
              "h-5 w-5 shrink-0 text-slate-500 transition-transform duration-200",
              isOpen && "rotate-180"
            )}
          />
        </span>

        <span className="mt-2.5 flex flex-wrap items-center gap-2 pl-[3.125rem]">
          <span className="employee-type-pill rounded-full bg-emerald-50 px-2.5 py-1.5 text-emerald-700">
            {isLoading ? "Đang tải" : `${workdayCount} ngày công`}
          </span>
          <span
            className={cn(
              "employee-type-pill rounded-full px-2.5 py-1.5",
              warningCount > 0
                ? "bg-amber-50 text-amber-700"
                : "bg-slate-100 text-slate-500"
            )}
          >
            {isLoading ? "Đang kiểm tra" : `${warningCount} cần kiểm tra`}
          </span>
        </span>
      </button>

      {isOpen && (
        <div id={panelId} className="border-t border-slate-100 pb-3">
          <div className="flex items-center justify-between gap-2 px-4 py-3">
            <button
              type="button"
              aria-label="Xem tháng trước"
              className="flex h-11 w-11 items-center justify-center rounded-xl border border-slate-200 text-slate-600 transition-colors hover:bg-slate-50 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500"
              onClick={() => setSelectedMonth((current) => getPreviousAttendanceHistoryMonth(current))}
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="employee-type-card-title text-center text-slate-950">
              {month.label}
            </span>
            <button
              type="button"
              aria-label="Xem tháng sau"
              disabled={!month.canGoNext}
              className="flex h-11 w-11 items-center justify-center rounded-xl border border-slate-200 text-slate-600 transition-colors hover:bg-slate-50 hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500 disabled:pointer-events-none disabled:text-slate-300"
              onClick={() => setSelectedMonth((current) => getNextAttendanceHistoryMonth(current))}
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>

          {isLoading ? (
            <div className="space-y-3 px-4">
              {Array.from({ length: 3 }).map((_, index) => (
                <Skeleton key={index} className="h-24 w-full rounded-xl" />
              ))}
            </div>
          ) : history.length === 0 ? (
            <div className="mx-4 rounded-2xl border border-dashed border-slate-300 bg-slate-50/70 px-4 py-7 text-center text-slate-600">
              <Calendar className="mx-auto mb-2 h-8 w-8 opacity-50" />
              <p className="employee-type-card-title text-slate-500">
                Chưa có ca làm trong kỳ này
              </p>
              <p className="employee-type-body-sm mt-1">
                Chọn tháng khác để xem lịch sử trước đó.
              </p>
            </div>
          ) : (
            <div className="divide-y divide-slate-100 border-y border-slate-100">
              {history.map((attendance) => (
                <AttendanceHistoryRow key={attendance.id} attendance={attendance} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
