import { useId, useMemo, useState } from "react";
import { format } from "date-fns";
import {
  AlertTriangle,
  Calendar,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  CircleX,
  Clock3,
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
import { formatDate } from "@/utils/formatters";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

const ATTENDANCE_PREVIEW_LIMIT = 3;

function formatTime(time: string | undefined | null, fallback = "--:--"): string {
  if (!time) return fallback;
  try {
    return format(new Date(time), "HH:mm");
  } catch {
    return fallback;
  }
}

interface EmployeeAttendanceHistoryCardProps {
  /** `yyyy-MM-dd` — first day of the month to show (drives the attendance query). */
  fromDate: string;
  /** `yyyy-MM-dd` — last day of the month to show (drives the attendance query). */
  toDate: string;
  /** Period label shown in the header (e.g. "07/2026"). */
  monthLabel: string;
  /**
   * Configured self-check-in advance percentage (e.g. 70). When provided, each
   * completed shift also shows the advanceable amount floor(earning × pct / 100)
   * — the same rounding the backend uses for the quota cap — so workers see why
   * the advance limit grows slower than the listed wages.
   */
  advancePercentage?: number;
  className?: string;
  style?: React.CSSProperties;
}

function HistoryIssueChips({ details }: { details: AttendanceIssueDetail[] }) {
  if (details.length === 0) return null;

  return (
    <div className="mt-2 grid grid-cols-[repeat(auto-fit,minmax(5rem,1fr))] gap-2">
      {details.map((detail) => (
        <div key={detail.label} className="rounded-lg border border-amber-200 bg-white px-2.5 py-2">
          <p className="employee-type-label-caps whitespace-nowrap text-[#667085]">{detail.label}</p>
          <p className="employee-type-inline-amount mt-1 text-[#101828]">{detail.value}</p>
        </div>
      ))}
    </div>
  );
}

function AttendanceStatusPill({ status }: { status: AttendanceRecord["status"] }) {
  const config = {
    completed: { className: "bg-[#ECFDF3] text-[#067647]", label: "Hoàn thành", icon: CheckCircle2 },
    rejected: { className: "bg-[#FFFAEB] text-[#B54708]", label: "Đã từ chối", icon: AlertTriangle },
    orphaned: { className: "bg-[#FEF3F2] text-[#B42318]", label: "Thiếu tan ca", icon: CircleX },
    checked_in: { className: "bg-[#EFF8FF] text-[#175CD3]", label: "Đang làm", icon: Clock3 },
  }[status];
  const Icon = config.icon;

  return (
    <span className={cn("employee-type-pill inline-flex shrink-0 items-center gap-1.5 rounded-md px-2 py-1", config.className)}>
      <Icon className="h-3.5 w-3.5" strokeWidth={2.25} aria-hidden="true" />
      {config.label}
    </span>
  );
}

function AttendanceHistoryRow({ attendance, advancePercentage }: { attendance: AttendanceRecord; advancePercentage?: number }) {
  const [isDetailOpen, setIsDetailOpen] = useState(false);
  const detailId = useId();
  const salaryRecorded = isSalaryRecorded(attendance);
  const salaryMissing = isSalaryMissing(attendance);
  const needsDetail = attendance.status === "rejected" || attendance.status === "orphaned" || salaryMissing;
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
  const showAdvanceable = salaryRecorded && advancePercentage !== undefined && advancePercentage > 0;
  const advanceableAmount = showAdvanceable
    ? Math.floor(((attendance.earning_amount ?? 0) * advancePercentage) / 100)
    : null;

  return (
    <article className="px-4 py-4 sm:px-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="employee-type-card-title text-[#101828]">Ngày {formatDate(attendance.date)}</p>
          <p className="employee-type-body-sm mt-1 flex items-center gap-1.5 text-[#667085] tabular-nums">
            <Clock3 className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
            <span>{formatTime(attendance.check_in_time)} — {formatTime(attendance.check_out_time)}</span>
          </p>
        </div>
        <AttendanceStatusPill status={attendance.status} />
      </div>

      {showAdvanceable && advanceableAmount !== null ? (
        <dl
          data-testid="attendance-earnings-summary"
          className="mt-3 grid grid-cols-2 gap-x-3 border-t border-[#EAECF0] pt-3"
        >
          <div className="min-w-0 border-r border-[#EAECF0] pr-3">
            <dt className="employee-type-label text-[#667085]">Tiền công</dt>
            <dd className="employee-type-inline-amount mt-1 text-[#101828] tabular-nums">
              {attendance.earning_amount?.toLocaleString("vi-VN")}
              <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
            </dd>
          </div>
          <div className="min-w-0 text-right">
            <dt className="employee-type-label text-[#667085]">Được ứng ({advancePercentage}%)</dt>
            <dd className="employee-type-inline-amount mt-1 text-[#067647] tabular-nums">
              +{advanceableAmount.toLocaleString("vi-VN")}
              <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
            </dd>
          </div>
        </dl>
      ) : (
        <div className="mt-3 flex items-center justify-between gap-3 border-t border-[#EAECF0] pt-3">
          <span className="employee-type-label text-[#667085]">Tiền công</span>
          {salaryRecorded ? (
            <span className="employee-type-inline-amount text-[#101828] tabular-nums">
              {attendance.earning_amount?.toLocaleString("vi-VN")}
              <span className="ml-[0.2em] align-[0.1em] text-[0.58em] font-bold tracking-normal text-muted-foreground">₫</span>
            </span>
          ) : (
            <span className={cn("employee-type-body-sm font-semibold", salaryMissing ? "text-[#B54708]" : "text-[#475467]")}>{salaryMissing ? "Chưa ghi lương" : "Chưa có"}</span>
          )}
        </div>
      )}

      {needsDetail && (
        <div className="mt-3 border-t border-[#EAECF0] pt-2">
          <button
            type="button"
            aria-expanded={isDetailOpen}
            aria-controls={detailId}
            onClick={() => setIsDetailOpen((current) => !current)}
            className="employee-type-action flex min-h-11 w-full items-center justify-between rounded-lg px-1 text-left text-[#B54708] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#B54708] focus-visible:ring-offset-2"
          >
            <span className="flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" aria-hidden="true" />
              Xem lý do
            </span>
            <ChevronDown className={cn("h-4 w-4 transition-transform duration-200", isDetailOpen && "rotate-180")} />
          </button>
          {isDetailOpen && (
            <div id={detailId} className="rounded-lg bg-[#FFFAEB] px-3 py-3 text-[#7A2E0E]">
              <p className="employee-type-strong">{issue.title}</p>
              <p className="employee-type-body-sm mt-1 text-[#B54708]">{issue.description}</p>
              <HistoryIssueChips details={issue.details} />
            </div>
          )}
        </div>
      )}
    </article>
  );
}

export function EmployeeAttendanceHistoryCard({
  fromDate,
  toDate,
  monthLabel,
  advancePercentage,
  className,
  style,
}: EmployeeAttendanceHistoryCardProps) {
  const [isFullHistoryOpen, setIsFullHistoryOpen] = useState(false);
  const panelId = useId();
  const historyParams = useMemo(
    () => ({ limit: 100, from_date: fromDate, to_date: toDate }),
    [fromDate, toDate]
  );
  const { data: historyResponse, isLoading } = useAttendanceHistory(historyParams);
  const history = useMemo(() => {
    const records = historyResponse?.data ?? [];
    return [...records].sort((left, right) => {
      const leftDate = Date.parse(left.check_in_time ?? left.date);
      const rightDate = Date.parse(right.check_in_time ?? right.date);
      return rightDate - leftDate;
    });
  }, [historyResponse?.data]);
  const workdayCount = history.filter((attendance) => attendance.status === "completed").length;
  const warningCount = history.filter(
    (attendance) => attendance.status === "rejected" || attendance.status === "orphaned" || isSalaryMissing(attendance)
  ).length;
  const visibleHistory = isFullHistoryOpen ? history : history.slice(0, ATTENDANCE_PREVIEW_LIMIT);
  const hasMoreHistory = history.length > ATTENDANCE_PREVIEW_LIMIT;

  return (
    <section className={className ?? "employee-surface-card overflow-hidden"} style={style} aria-labelledby="employee-attendance-title">
      <div className="flex items-center justify-between gap-3 px-4 py-4 sm:px-5">
        <div className="flex min-w-0 items-center gap-3">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-[var(--employee-accent-soft)] text-[var(--employee-accent)]">
            <History className="h-[18px] w-[18px]" strokeWidth={2.25} aria-hidden="true" />
          </span>
          <div className="min-w-0">
            <h2 id="employee-attendance-title" className="employee-type-card-title text-[var(--employee-text)]">Chấm công</h2>
            <p className="employee-type-body-sm mt-0.5 text-[var(--employee-text-secondary)]">Bảng công tháng {monthLabel}</p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1.5" aria-label="Tóm tắt chấm công">
          <span className="employee-type-pill whitespace-nowrap rounded-md border border-[var(--employee-accent-border)] bg-[var(--employee-accent-soft)] px-2 py-1 text-[var(--employee-accent)]">{isLoading ? "Đang tải" : `${workdayCount} ngày công`}</span>
          {warningCount > 0 && <span className="employee-type-pill whitespace-nowrap rounded-md bg-[var(--employee-warning-soft)] px-2 py-1 text-[var(--employee-warning)]">{warningCount} cần kiểm tra</span>}
        </div>
      </div>

      {isLoading ? (
        <div className="space-y-3 px-4 pb-4">
          {Array.from({ length: ATTENDANCE_PREVIEW_LIMIT }).map((_, index) => <Skeleton key={index} className="h-20 w-full rounded-xl" />)}
        </div>
      ) : history.length === 0 ? (
        <div className="mx-4 mb-4 rounded-xl border border-dashed border-[#D0D5DD] px-4 py-7 text-center text-[#475467]">
          <Calendar className="mx-auto mb-2 h-8 w-8 text-[#98A2B3]" aria-hidden="true" />
          <p className="employee-type-card-title">Chưa có ca làm trong tháng {monthLabel}</p>
          <p className="employee-type-body-sm mt-1">Chọn tháng khác để xem lịch sử trước đó.</p>
        </div>
      ) : (
        <div id={panelId} className="divide-y divide-[#EAECF0] border-y border-[#EAECF0]">
          {visibleHistory.map((attendance) => <AttendanceHistoryRow key={attendance.id} attendance={attendance} advancePercentage={advancePercentage} />)}
        </div>
      )}

      {hasMoreHistory && (
        <div className="px-4 py-3 sm:px-5">
          <button type="button" aria-expanded={isFullHistoryOpen} aria-controls={panelId} onClick={() => setIsFullHistoryOpen((current) => !current)} className="employee-type-action flex min-h-11 w-full items-center justify-between rounded-lg px-3 text-[#067647] transition-colors hover:bg-[#F0FDF4] focus-visible:ring-2 focus-visible:ring-[#07883F]">
            <span>{isFullHistoryOpen ? "Thu gọn lịch chấm công" : "Xem toàn bộ lịch chấm công"}</span>
            {isFullHistoryOpen ? <ChevronUp className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
          </button>
        </div>
      )}
    </section>
  );
}
