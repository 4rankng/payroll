import { type ColumnDef } from "@tanstack/react-table";
import { format } from "date-fns";
import { MoreHorizontal, MapPin, Check, X, Zap } from "lucide-react";
import { formatCurrency } from "@/utils/formatters";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import {
  canApproveAttendance,
  canCreditAttendanceQuota,
  canRejectAttendance,
  getAttendanceOperationalStatus,
  getAttendanceReviewStatusLabel,
  needsAttendanceApprovalRepair,
} from "@/utils/attendanceReviewState";

/** Row-action callbacks wired by the page. Each is optional so the table can
 * render without them (e.g. read-only contexts). */
export interface AttendanceRowActions {
  onViewMap: (row: AdminAttendanceResponse) => void;
  onApprove: (row: AdminAttendanceResponse) => void;
  onReject: (row: AdminAttendanceResponse) => void;
  /** Admin-only "credit now": bank a completed shift's earning into the
   * advance quota immediately, skipping the post-checkout hold. Omitted by
   * pages for roles without the admin attendance endpoints. */
  onCreditQuota?: (row: AdminAttendanceResponse) => void;
}

const SYSTEM_STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  checked_in: { label: "Đang làm", className: "bg-blue-50 text-blue-700 border-blue-200" },
  completed: { label: "Hoàn thành", className: "bg-green-50 text-green-700 border-green-200" },
  orphaned: { label: "Thiếu Check-out", className: "bg-red-50 text-red-700 border-red-200" },
  rejected: { label: "Đã từ chối", className: "bg-orange-50 text-orange-700 border-orange-200" },
};

const REVIEW_BADGE_CONFIG: Record<string, { label: string; className: string }> = {
  approved: { label: "Đã duyệt", className: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  rejected: { label: "Đã huỷ", className: "bg-rose-50 text-rose-700 border-rose-200" },
};

function StatusBadges({
  status,
  reviewAction,
  checkOutTime,
  quotaCreditedAt,
  earningAmount,
}: {
  status: string;
  reviewAction?: AdminAttendanceResponse["review_action"];
  checkOutTime?: string | null;
  quotaCreditedAt?: string | null;
  earningAmount?: number;
}) {
  const attendance = {
    status,
    review_action: reviewAction,
    check_out_time: checkOutTime,
    quota_credited_at: quotaCreditedAt,
    earning_amount: earningAmount,
  };
  const reviewStatusLabel = getAttendanceReviewStatusLabel(attendance);
  if (needsAttendanceApprovalRepair(attendance)) {
    return (
      <div className="inline-flex items-center rounded-full border border-amber-200 bg-amber-50 px-2 py-0.5 text-[10px] font-semibold text-amber-700">
        {reviewStatusLabel}
      </div>
    );
  }
  if (reviewAction === "approved") {
    const review = REVIEW_BADGE_CONFIG.approved;
    return (
      <div className={cn("inline-flex items-center gap-0.5 rounded-full border px-1.5 py-0.5 text-[9px] font-semibold", review.className)}>
        <Check className="h-2.5 w-2.5" />
        {reviewStatusLabel ?? review.label}
      </div>
    );
  }
  const operationalStatus = getAttendanceOperationalStatus(attendance);
  const sys = SYSTEM_STATUS_CONFIG[operationalStatus] ?? { label: "Không rõ", className: "bg-gray-100 text-gray-600 border-gray-200" };
  const review = reviewAction ? REVIEW_BADGE_CONFIG[reviewAction] : null;
  return (
    <div className="flex flex-col items-start gap-1">
      <div className={cn("inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold border", sys.className)}>
        {sys.label}
      </div>
      {review && (
        <div className={cn("inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-full text-[9px] font-semibold border", review.className)}>
          <X className="h-2.5 w-2.5" />
          {reviewStatusLabel ?? review.label}
        </div>
      )}
      {canCreditAttendanceQuota(attendance) && (
        <div className="inline-flex items-center px-1.5 py-0.5 rounded-full text-[9px] font-semibold border border-amber-200 bg-amber-50 text-amber-700">
          Chờ cộng hạn mức
        </div>
      )}
    </div>
  );
}

export function getAdminAttendanceColumns(actions?: AttendanceRowActions): ColumnDef<AdminAttendanceResponse>[] {
  const cols: ColumnDef<AdminAttendanceResponse>[] = [
    {
      accessorKey: "employee_name",
      header: "Nhân viên",
      size: 150,
      cell: ({ row }) => (
        <div>
          <div className="text-xs font-medium leading-tight">{row.original.employee_name}</div>
          <div className="text-xs text-muted-foreground/70 mt-0.5 truncate max-w-[140px]">
            {row.original.project_name || "-"}
          </div>
        </div>
      ),
    },
    {
      accessorKey: "date",
      header: "Ngày",
      size: 100,
      cell: ({ row }) => {
        try {
          return <span className="text-xs font-medium">{format(new Date(row.original.date), "dd/MM/yyyy")}</span>;
        } catch {
          return <span className="text-xs">{row.original.date}</span>;
        }
      },
    },
    {
      id: "check_in",
      header: "Check-in",
      size: 130,
      cell: ({ row }) => {
        if (!row.original.check_in_time) return <span className="text-xs text-muted-foreground">-</span>;
        try {
          return (
            <div>
              <div className="text-xs font-medium text-slate-700">{format(new Date(row.original.check_in_time), "HH:mm")}</div>
              <div className="text-[10px] text-muted-foreground truncate max-w-[120px]">{row.original.check_in_gate || "Chưa xác định"}</div>
            </div>
          );
        } catch {
          return <span className="text-xs">{row.original.check_in_time}</span>;
        }
      }
    },
    {
      id: "check_out",
      header: "Check-out",
      size: 130,
      cell: ({ row }) => {
        if (!row.original.check_out_time) return <span className="text-xs text-muted-foreground">-</span>;
        try {
          return (
            <div>
              <div className="text-xs font-medium text-slate-700">{format(new Date(row.original.check_out_time), "HH:mm")}</div>
              <div className="text-[10px] text-muted-foreground truncate max-w-[120px]">{row.original.check_out_gate || "Chưa xác định"}</div>
            </div>
          );
        } catch {
          return <span className="text-xs">{row.original.check_out_time}</span>;
        }
      }
    },
    {
      accessorKey: "earning_amount",
      header: "Thu nhập",
      size: 110,
      cell: ({ row }) => {
        const amount = row.original.earning_amount;
        if (amount == null) return <span className="text-xs text-muted-foreground">-</span>;
        return <span className="text-xs font-bold text-success">{formatCurrency(amount)}</span>;
      },
    },
    {
      accessorKey: "status",
      header: "Trạng thái",
      size: 120,
      cell: ({ row }) => (
        <StatusBadges
          status={row.original.status}
          reviewAction={row.original.review_action}
          checkOutTime={row.original.check_out_time}
          quotaCreditedAt={row.original.quota_credited_at}
          earningAmount={row.original.earning_amount}
        />
      ),
    },
  ];

  // Actions column: a kebab menu with View map (always) + Approve/Credit/Reject.
  // The review actions are hidden once the attendance reaches a terminal state:
  // "Duyệt" is hidden on Hoàn thành (completed) or Đã duyệt (approved); "Từ chối"
  // is hidden on Đã từ chối (rejected) or Đã huỷ (review-rejected) or once the
  // quota is credited; "Cộng ngay" shows only on completed shifts still
  // waiting out their quota-credit hold, and only when the page supplies the
  // (admin-only) callback. Omitted entirely when no callbacks are provided so
  // the table stays read-only in contexts that don't need it.
  if (actions) {
    cols.push({
      id: "actions",
      header: "",
      size: 48,
      cell: ({ row }) => {
        const att = row.original;
        const showApprove = canApproveAttendance(att);
        const showReject = canRejectAttendance(att);
        const onCreditQuota = actions.onCreditQuota;
        const showCreditQuota = onCreditQuota != null && canCreditAttendanceQuota(att);
        return (
          <div className="text-right">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  aria-label="Hành động"
                  className="inline-flex h-7 w-7 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                  onClick={(e) => e.stopPropagation()}
                >
                  <MoreHorizontal className="h-4 w-4" />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-44" onClick={(e) => e.stopPropagation()}>
                <DropdownMenuItem onClick={() => actions.onViewMap(att)}>
                  <MapPin className="mr-2 h-4 w-4" />
                  Xem bản đồ
                </DropdownMenuItem>
                {showApprove && (
                  <DropdownMenuItem onClick={() => actions.onApprove(att)}>
                    <Check className="mr-2 h-4 w-4" />
                    {needsAttendanceApprovalRepair(att) ? "Duyệt lại" : "Duyệt"}
                  </DropdownMenuItem>
                )}
                {showCreditQuota && (
                  <DropdownMenuItem onClick={() => onCreditQuota(att)}>
                    <Zap className="mr-2 h-4 w-4" />
                    Cộng ngay
                  </DropdownMenuItem>
                )}
                {showReject && (
                  <DropdownMenuItem
                    onClick={() => actions.onReject(att)}
                    className="text-rose-600 focus:text-rose-700"
                  >
                    <X className="mr-2 h-4 w-4" />
                    Từ chối
                  </DropdownMenuItem>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        );
      },
    });
  }

  return cols;
}

export const attendanceMobileFields = [
  {
    key: "date",
    label: "Ngày",
    render: (row: AdminAttendanceResponse) => {
      try {
        return format(new Date(row.date), "dd/MM/yyyy");
      } catch {
        return row.date;
      }
    }
  },
  {
    key: "check_in_time",
    label: "Check-in",
    render: (row: AdminAttendanceResponse) => row.check_in_time ? format(new Date(row.check_in_time), "HH:mm") : "-"
  },
  {
    key: "check_out_time",
    label: "Check-out",
    render: (row: AdminAttendanceResponse) => row.check_out_time ? format(new Date(row.check_out_time), "HH:mm") : "-"
  },
  {
    key: "earning_amount",
    label: "Thu nhập",
    render: (row: AdminAttendanceResponse) => row.earning_amount != null ? formatCurrency(row.earning_amount) : "-"
  }
];

export const attendanceEmptyState = {
  title: "Chưa có dữ liệu chấm công",
  description: "Không tìm thấy bản ghi chấm công nào phù hợp với bộ lọc hiện tại.",
};
