import { useState, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { MobilePagination } from "@/components/shared/MobilePagination";
import { StatusFilterBar } from "@/components/advance-payment/StatusFilterBar";
import { useExportAdvancePayments } from "@/hooks/api/useAdvancePayments";
import { useAdvancePaymentsPage } from "@/hooks/advance-payment/useAdvancePaymentsPage";
import { useAdminAttendancePage } from "@/hooks/advance-payment/useAdminAttendancePage";
import {
  useApproveAttendance,
  useRejectAttendance,
  useCreditAttendanceQuota,
} from "@/hooks/api/useAdminAttendance";
import { useSendPayrollReportEmail } from "@/hooks/transactions/useSendPayrollReportEmail";
import { formatVnTime } from "@/utils/vn-time";
// UU PRO restyle (W13c): lucide → verified @untitledui/icons; blue-gray hex
// chrome moved onto the utility-gray bridge. Flexible-pay quota logic and all
// flows byte-preserved.
import {
  CalendarCheck01,
  Check,
  ClockRewind,
  DownloadCloud01,
  File05,
  Mail01,
  MarkerPin01,
  Receipt,
  SwitchHorizontal02,
  UserCheck01,
  Users01,
  XClose,
  Zap,
} from "@untitledui/icons";
import { AdvancePaymentPageHeaderMobile } from "@/components/advance-payment/AdvancePaymentPageHeaderMobile";
import { MobileOverflowAction, MobileOverflowDivider } from "@/components/advance-payment/actions";
import { AdvancePaymentMobileList } from "@/components/advance-payment/AdvancePaymentMobileList";
import { FlexibleEmployeeListUploadDialog } from "@/components/advance-payment/FlexibleEmployeeListUploadDialog";
import { AdvancePaymentResultUploadDialog } from "@/components/advance-payment/AdvancePaymentResultUploadDialog";
import { ImportPayrollDialog } from "@/components/advance-payment/ImportPayrollDialog";
import { StatementDialog } from "@/components/advance-payment/StatementDialog";
import { FileHistorySheet } from "@/components/advance-payment/FileHistorySheet";
import { PayrollReportEmailDialog } from "@/components/timesheet/PayrollReportEmailDialog";
import { AdvPartnerHeroStrip } from "@/components/advance-payment/AdvPartnerHeroStrip";
import { AdvPartnerStatusOverview } from "@/components/advance-payment/AdvPartnerStatusOverview";
import { TreasuryFeePanel } from "@/components/advance-payment/TreasuryFeePanel";
import { WalletBalanceCard } from "@/components/disbursement/WalletBalanceCard";
import { TimesheetMonthSelector } from "@/components/timesheet/TimesheetMonthSelector";
import { MobilePageShell, MobileSurface } from "@/components/shared/MobilePageShell";
import { AttendanceMapDialog } from "@/components/attendance/AttendanceMapDialog";
import { AdminCreateCheckInDialog } from "@/components/attendance/AdminCreateCheckInDialog";
import {
  AttendanceReviewDialogs,
  type ReviewMode,
} from "@/components/attendance/AttendanceReviewDialogs";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { format } from "date-fns";
import { vi } from "date-fns/locale";
import { formatCurrency } from "@/utils/formatters";
import { cn } from "@/lib/utils";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";
import {
  canApproveAttendance,
  canCreditAttendanceQuota,
  canRejectAttendance,
  getAttendanceOperationalStatus,
  getAttendanceReviewStatusLabel,
  needsAttendanceApprovalRepair,
} from "@/utils/attendanceReviewState";
import type { PayrollReportEmailParams } from "@/components/timesheet/PayrollReportEmailDialog";
import { useAuth } from "@/contexts";

type AdminTab = "requests" | "attendances";

const ATTENDANCE_STATUS_LABEL: Record<string, string> = {
  checked_in: "Đang làm",
  completed: "Hoàn thành",
  orphaned: "Thiếu check-out",
  rejected: "Đã từ chối",
};

/** Mobile card for a single attendance row — mirrors desktop's mobileFields. */
export function AttendanceMobileCard({
  row,
  onViewMap,
  onApprove,
  onReject,
  onCreditQuota,
}: {
  row: AdminAttendanceResponse;
  onViewMap: (row: AdminAttendanceResponse) => void;
  onApprove: (row: AdminAttendanceResponse) => void;
  onReject: (row: AdminAttendanceResponse) => void;
  /** Admin-only "credit now": bank the earning into the advance quota
   * immediately, skipping the post-checkout hold. */
  onCreditQuota?: (row: AdminAttendanceResponse) => void;
}) {
  const fmtTime = (t?: string) => (t ? formatVnTime(t) || "-" : "-");
  const fmtDate = (d: string) => {
    try { return format(new Date(d), "dd/MM/yyyy"); } catch { return d; }
  };
  const reviewStatusLabel = getAttendanceReviewStatusLabel(row);
  const statusLabel = needsAttendanceApprovalRepair(row) || row.review_action === "approved"
    ? reviewStatusLabel
    : ATTENDANCE_STATUS_LABEL[getAttendanceOperationalStatus(row)] ?? row.status;
  const statusBadgeClass = needsAttendanceApprovalRepair(row)
    ? "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-300"
    : row.review_action === "approved"
    ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300"
    : row.review_action === "rejected"
    ? "border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-300"
    : getAttendanceOperationalStatus(row) === "checked_in"
    ? "border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950/40 dark:text-blue-300"
    : getAttendanceOperationalStatus(row) === "completed"
    ? "border-green-200 bg-green-50 text-green-700 dark:border-green-800 dark:bg-green-950/40 dark:text-green-300"
    : getAttendanceOperationalStatus(row) === "orphaned"
    ? "border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300"
    : "border-border bg-muted text-muted-foreground";

  const showApprove = canApproveAttendance(row);
  const showReject = canRejectAttendance(row);
  const showCreditQuota = onCreditQuota != null && canCreditAttendanceQuota(row);

  return (
    <div className="rounded-xl border border-border bg-card p-3.5 shadow-sm">
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-foreground">{row.employee_name}</p>
          <p className="mt-0.5 max-w-full truncate text-xs text-muted-foreground">
            {row.project_name || "—"}
          </p>
        </div>
        <span className={cn("shrink-0 rounded-full border px-2 py-0.5 text-xs font-semibold", statusBadgeClass)}>
          {statusLabel}
        </span>
      </div>

      {/* Clean metadata layout — no nested card boxes */}
      <div className="mt-2.5 space-y-1.5 border-t border-border/60 pt-2 text-xs">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5 min-w-0">
            <span className="text-muted-foreground shrink-0">Ngày:</span>
            <span className="font-medium text-foreground truncate">{fmtDate(row.date)}</span>
          </div>
          <div className="flex items-center gap-1.5 shrink-0">
            <span className="text-muted-foreground">Thu nhập:</span>
            <span className="font-financial font-semibold text-foreground tabular-nums">
              {row.earning_amount != null ? formatCurrency(row.earning_amount) : "—"}
            </span>
          </div>
        </div>

        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5 min-w-0">
            <span className="text-muted-foreground shrink-0">Vào:</span>
            <span className="font-medium tabular-nums text-foreground">{fmtTime(row.check_in_time)}</span>
            {row.check_in_gate ? (
              <span className="text-[11px] text-muted-foreground truncate max-w-[100px]">({row.check_in_gate})</span>
            ) : null}
          </div>
          <div className="flex items-center gap-1.5 shrink-0">
            <span className="text-muted-foreground">Ra:</span>
            <span className="font-medium tabular-nums text-foreground">{fmtTime(row.check_out_time)}</span>
            {row.check_out_gate ? (
              <span className="text-[11px] text-muted-foreground truncate max-w-[100px]">({row.check_out_gate})</span>
            ) : null}
          </div>
        </div>
      </div>

      {showCreditQuota && (
        <p className="mt-2 text-xs font-medium text-fg-warning-primary">Chờ cộng hạn mức</p>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="outline"
          className="min-h-11 flex-1 gap-1.5 px-2 text-xs"
          onClick={() => onViewMap(row)}
        >
          <MarkerPin01 className="h-4 w-4" />
          Bản đồ
        </Button>
        {showApprove && (
          <Button
            type="button"
            variant="outline"
            className="min-h-11 flex-1 gap-1.5 px-2 text-xs text-fg-success-primary"
            onClick={() => onApprove(row)}
          >
            <Check className="h-4 w-4" />
            {needsAttendanceApprovalRepair(row) ? "Duyệt lại" : "Duyệt"}
          </Button>
        )}
        {showCreditQuota && (
          <Button
            type="button"
            variant="outline"
            className="min-h-11 flex-1 gap-1.5 px-2 text-xs text-fg-warning-primary"
            onClick={() => onCreditQuota(row)}
          >
            <Zap className="h-4 w-4" />
            Cộng ngay
          </Button>
        )}
        {showReject && (
          <Button
            type="button"
            variant="outline"
            className="min-h-11 flex-1 gap-1.5 px-2 text-xs text-fg-error-primary"
            onClick={() => onReject(row)}
          >
            <XClose className="h-4 w-4" />
            Từ chối
          </Button>
        )}
      </div>
    </div>
  );
}

const AdvancePaymentsPageMobile = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAdvPartner = user?.role === "adv_partner";

  const [isImportSheetOpen, setIsImportSheetOpen] = useState(false);
  const [isEmployeeListUploadOpen, setIsEmployeeListUploadOpen] = useState(false);
  const [isResultUploadOpen, setIsResultUploadOpen] = useState(false);
  const [isStatementSheetOpen, setIsStatementSheetOpen] = useState(false);
  const [isHistorySheetOpen, setIsHistorySheetOpen] = useState(false);
  const [isEmailDialogOpen, setIsEmailDialogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<AdminTab>("requests");
  const [cancelTargetId, setCancelTargetId] = useState<number | null>(null);
  const [selectedAttendance, setSelectedAttendance] =
    useState<AdminAttendanceResponse | null>(null);
  const [reviewMode, setReviewMode] = useState<ReviewMode>(null);
  const [reviewRow, setReviewRow] =
    useState<AdminAttendanceResponse | null>(null);
  const [isCreateCheckInOpen, setIsCreateCheckInOpen] = useState(false);

  const page = useAdvancePaymentsPage({ employeesTabActive: false });
  const attendance = useAdminAttendancePage({ active: activeTab === "attendances" });
  const approveAttendanceMutation = useApproveAttendance();
  const rejectAttendanceMutation = useRejectAttendance();
  const creditQuotaMutation = useCreditAttendanceQuota();
  const exportBatchMutation = useExportAdvancePayments();
  const sendEmailMutation = useSendPayrollReportEmail();

  const summaryData = page.summary?.data;

  const heroProps = useMemo(
    () => ({
      totalAmount: summaryData?.totalAmount ?? 0,
      totalPaid: summaryData?.totalPaid ?? 0,
    }),
    [summaryData],
  );

  const statusProps = useMemo(
    () => ({
      totalPaid: summaryData?.totalPaid ?? 0,
      totalPending: summaryData?.totalPending ?? 0,
      totalFailed: summaryData?.totalFailed ?? 0,
      totalCancelled: summaryData?.totalCancelled ?? 0,
      totalRequests: summaryData?.totalRequests ?? 0,
      totalPaidAmount: summaryData?.totalPaidAmount ?? 0,
      totalPendingAmount: summaryData?.totalPendingAmount ?? 0,
      totalFailedAmount: summaryData?.totalFailedAmount ?? 0,
      totalCancelledAmount: summaryData?.totalCancelledAmount ?? 0,
      successRate: summaryData?.successRate ?? 0,
    }),
    [summaryData],
  );

  const feePanelProps = useMemo(
    () => ({
      totalFeeEarned: summaryData?.totalFeeEarned ?? 0,
      feePercentage: summaryData?.feePercentage ?? 0,
      avgFeePerRequest: summaryData?.avgFeePerRequest ?? 0,
      avgFeePerEmployee: summaryData?.avgFeePerEmployee ?? 0,
    }),
    [summaryData],
  );

  const handleSendEmail = useCallback(
    async (params: PayrollReportEmailParams) => {
      try {
        await sendEmailMutation.mutateAsync(params);
        setIsEmailDialogOpen(false);
      } catch { /* handled by mutation */ }
    },
    [sendEmailMutation],
  );

  /** Confirm-then-cancel — restores the safety guard desktop has. */
  const handleRequestCancel = useCallback((id: number) => {
    setCancelTargetId(id);
  }, []);

  const handleConfirmCancel = useCallback(async () => {
    if (cancelTargetId == null) return;
    try {
      await page.cancelMutation.mutateAsync(cancelTargetId);
    } catch { /* handled by mutation */ } finally {
      setCancelTargetId(null);
    }
  }, [cancelTargetId, page.cancelMutation]);

  const handleReviewClose = useCallback(() => {
    setReviewMode(null);
    setReviewRow(null);
  }, []);

  const handleApproveAttendance = useCallback(
    (row: AdminAttendanceResponse, note: string) => {
      approveAttendanceMutation.mutate(
        { id: row.id, note },
        { onSuccess: handleReviewClose },
      );
    },
    [approveAttendanceMutation, handleReviewClose],
  );

  const handleRejectAttendance = useCallback(
    (row: AdminAttendanceResponse, note: string) => {
      rejectAttendanceMutation.mutate(
        { id: row.id, note },
        { onSuccess: handleReviewClose },
      );
    },
    [rejectAttendanceMutation, handleReviewClose],
  );

  const handleCreditAttendanceQuota = useCallback(
    (row: AdminAttendanceResponse) => {
      creditQuotaMutation.mutate(
        row.id,
        { onSuccess: handleReviewClose },
      );
    },
    [creditQuotaMutation, handleReviewClose],
  );

  if (page.summaryLoading && page.requests.length === 0) {
    return (
      <div className="max-w-full space-y-4 overflow-hidden p-4 pb-[calc(5rem+env(safe-area-inset-bottom))]">
        <Skeleton className="h-10 w-full" />
        <div className="grid grid-cols-1 gap-px min-[380px]:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-none first:rounded-tl-xl [&:nth-child(2)]:rounded-tr-xl [&:nth-child(3)]:rounded-bl-xl last:rounded-br-xl" />
          ))}
        </div>
        <div className="grid grid-cols-1 gap-px min-[380px]:grid-cols-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-20" />
          ))}
        </div>
        <div className="space-y-2.5">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <MobilePageShell className="space-y-3">
      <MobileSurface className="p-3">
        <AdvancePaymentPageHeaderMobile
          onImportPayroll={() => setIsImportSheetOpen(true)}
          onViewEmployees={() => navigate("employees")}
          renderOverflowContent={(close) => (
            <>
              <MobileOverflowAction
                icon={DownloadCloud01}
                label="Chuyển lô"
                onClick={() => { exportBatchMutation.mutate(undefined); close(); }}
                disabled={exportBatchMutation.isPending}
                isLoading={exportBatchMutation.isPending}
              />
              {!isAdvPartner && (
                <MobileOverflowAction
                  icon={SwitchHorizontal02}
                  label="Nhập KQ"
                  onClick={() => { setIsResultUploadOpen(true); close(); }}
                />
              )}
              <MobileOverflowAction
                icon={CalendarCheck01}
                label="Cấu hình điểm danh"
                onClick={() => {
                  navigate(
                    isAdvPartner
                      ? "/adv-partner/advance-payments/check-in-settings"
                      : "/admin/advance-payments/check-in-settings",
                  );
                  close();
                }}
              />
              <MobileOverflowAction
                icon={File05}
                label="Sao kê"
                onClick={() => { setIsStatementSheetOpen(true); close(); }}
              />
              <MobileOverflowAction
                icon={Mail01}
                label="Email sao kê"
                onClick={() => { setIsEmailDialogOpen(true); close(); }}
                disabled={sendEmailMutation.isPending}
                isLoading={sendEmailMutation.isPending}
              />
              <MobileOverflowDivider />
              {!isAdvPartner && (
                <MobileOverflowAction
                  icon={ClockRewind}
                  label="Lịch sử file"
                  onClick={() => { setIsHistorySheetOpen(true); close(); }}
                />
              )}
            </>
          )}
        />

        <div className="mt-3 overflow-x-auto rounded-xl border border-utility-gray-300 bg-utility-gray-50 p-1">
          <TimesheetMonthSelector
            value={page.selectedMonth}
            onChange={page.setSelectedMonth}
            className="min-w-max"
          />
        </div>
      </MobileSurface>

      <div className="grid gap-3">
        {!isAdvPartner && (
          <WalletBalanceCard
            compact
            monthlyProviderFee={page.providerFees.monthlyProviderFee}
            totalProviderFee={page.providerFees.totalProviderFee}
            className="rounded-2xl border border-utility-gray-300 shadow-[0_16px_42px_-30px_rgba(15,23,42,0.75)]"
          />
        )}

        {/* Treasury summary — one card, stacked stat rows (mirrors desktop hero) */}
        <section
          aria-label="Tổng quan kỳ ứng lương"
          className="overflow-hidden rounded-2xl border border-utility-gray-300 bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05),0_14px_32px_-28px_rgba(15,49,103,0.55)]"
        >
          <div className="treasury-grid">
            <AdvPartnerHeroStrip
              {...heroProps}
              compact
              isLoading={page.summaryLoading}
              className="rounded-none border-0 shadow-none"
            />
            <TreasuryFeePanel
              {...feePanelProps}
              compact
              isLoading={page.summaryLoading}
            />
          </div>
        </section>
      </div>

      {/* Tab switcher: Yêu cầu / Chấm công — restores desktop tab parity */}
      {!isAdvPartner && (
        <div className="grid grid-cols-2 gap-1 rounded-xl bg-muted/60 p-1">
          <button
            onClick={() => setActiveTab("requests")}
            className={cn(
              "flex min-h-11 items-center justify-center gap-1.5 rounded-lg py-2 text-xs font-semibold transition-colors",
              activeTab === "requests"
                ? "bg-white text-primary shadow-sm"
                : "text-muted-foreground",
            )}
          >
            <Receipt className="h-3.5 w-3.5" />
            Yêu cầu
          </button>
          <button
            onClick={() => setActiveTab("attendances")}
            className={cn(
              "flex min-h-11 items-center justify-center gap-1.5 rounded-lg py-2 text-xs font-semibold transition-colors",
              activeTab === "attendances"
                ? "bg-white text-primary shadow-sm"
                : "text-muted-foreground",
            )}
          >
            <CalendarCheck01 className="h-3.5 w-3.5" />
            Chấm công
          </button>
        </div>
      )}

      {/* Requests list */}
      {activeTab === "requests" && (
      <MobileSurface className="space-y-3 p-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.14em] text-fg-secondary">
              Yêu cầu
            </p>
            <h2 className="text-[15px] font-bold leading-tight text-fg-primary">
              Danh sách ứng lương
            </h2>
          </div>
          <div className="grid h-8 min-w-8 place-items-center rounded-full bg-utility-gray-100 px-2 text-xs font-semibold tabular-nums text-fg-secondary">
            {page.statusCounts?.all ?? 0}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <MobileSearchInput
            value={page.searchInput}
            onSearch={page.handleSearch}
            placeholder="Tìm tên hoặc CCCD..."
            className="min-w-0 flex-1"
          />
          {/* Horizontally scrollable filter bar with right-edge fade hint */}
          <div className="relative shrink-0">
            <div className="overflow-x-auto pb-0.5 scrollbar-none">
              <StatusFilterBar
                value={((page.filters.status as string) || "all") as import("@/types/api/advance-payment.types").AdvancePaymentRequestStatus | "all"}
                onChange={page.handleStatusChange}
                counts={page.statusCounts}
                triggerClassName="h-11 min-w-[108px] rounded-lg border-utility-gray-300 bg-utility-gray-50 px-3 text-[13px]"
              />
            </div>
          </div>
        </div>

        <AdvancePaymentMobileList
          requests={page.requests}
          isCancelling={page.cancelMutation.isPending}
          cancellingId={page.cancelMutation.variables ?? undefined}
          onCancel={handleRequestCancel}
          onRetry={(id) => page.retryMutation.mutate(id)}
          isRetrying={page.retryMutation.isPending}
          retryingId={page.retryMutation.variables ?? undefined}
          pollingIds={page.pollingIds}
          pagination={page.pagination}
          onPageChange={page.handlePageChange}
        />
      </MobileSurface>
      )}

      {/* Attendance ("Chấm công") — restores desktop tab parity */}
      {activeTab === "attendances" && !isAdvPartner && (
        <MobileSurface className="space-y-3 p-3">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.14em] text-fg-secondary">
                Chấm công
              </p>
              <h2 className="text-[15px] font-bold leading-tight text-fg-primary">
                Chấm công nhân viên
              </h2>
            </div>
            <div className="grid h-8 min-w-8 place-items-center rounded-full bg-utility-gray-100 px-2 text-xs font-semibold tabular-nums text-fg-secondary">
              {attendance.totalRecords ?? 0}
            </div>
          </div>

          {/* Attendance status filter */}
          <Select
            value={attendance.filters.status || "all"}
            onValueChange={attendance.handleStatusChange}
          >
            <SelectTrigger className="h-11 w-full text-sm">
              <SelectValue placeholder="Trạng thái" />
            </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">Tất cả trạng thái</SelectItem>
            <SelectItem value="checked_in">Đang làm</SelectItem>
            <SelectItem value="completed">Hoàn thành</SelectItem>
            <SelectItem value="orphaned">Thiếu check-out</SelectItem>
            <SelectItem value="rejected">Đã từ chối</SelectItem>
          </SelectContent>
        </Select>

          <Button
            type="button"
            className="min-h-11 w-full gap-1.5"
            onClick={() => setIsCreateCheckInOpen(true)}
          >
            <UserCheck01 className="h-4 w-4" />
            Tạo check-in
          </Button>

          {attendance.isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-24 rounded-xl" />
              ))}
            </div>
          ) : attendance.attendances.length === 0 ? (
            <div className="py-12 text-center">
              <Users01 className="mx-auto h-10 w-10 text-muted-foreground" />
              <p className="mt-3 text-sm font-medium text-muted-foreground">
                Không có dữ liệu chấm công
              </p>
            </div>
          ) : (
            <>
              <div className="space-y-2">
                {attendance.attendances.map((row) => (
                  <AttendanceMobileCard
                    key={row.id}
                    row={row}
                    onViewMap={setSelectedAttendance}
                    onApprove={(attendanceRow) => {
                      setReviewRow(attendanceRow);
                      setReviewMode("approve");
                    }}
                    onCreditQuota={(attendanceRow) => {
                      setReviewRow(attendanceRow);
                      setReviewMode("credit");
                    }}
                    onReject={(attendanceRow) => {
                      setReviewRow(attendanceRow);
                      setReviewMode("reject");
                    }}
                  />
                ))}
              </div>
              {attendance.pagination && (
                <MobilePagination
                  pagination={{
                    page: attendance.pagination.page,
                    pageSize: attendance.pagination.limit,
                    totalPages: attendance.pagination.totalPages,
                    totalRecords: attendance.pagination.totalRecords,
                  }}
                  onPageChange={attendance.handlePageChange}
                />
              )}
            </>
          )}
        </MobileSurface>
      )}

      <AdvPartnerStatusOverview
        {...statusProps}
        isLoading={page.summaryLoading}
        className="rounded-2xl border-utility-gray-300 bg-white"
      />

      <ImportPayrollDialog open={isImportSheetOpen} onOpenChange={setIsImportSheetOpen} />
      <FlexibleEmployeeListUploadDialog open={isEmployeeListUploadOpen} onOpenChange={setIsEmployeeListUploadOpen} />
      <AdvancePaymentResultUploadDialog open={isResultUploadOpen} onOpenChange={setIsResultUploadOpen} />
      <StatementDialog open={isStatementSheetOpen} onOpenChange={setIsStatementSheetOpen} forMonth={page.flexPayMonth} />
      <FileHistorySheet open={isHistorySheetOpen} onOpenChange={setIsHistorySheetOpen} />
      <AttendanceReviewDialogs
        mode={reviewMode}
        row={reviewRow}
        onClose={handleReviewClose}
        onApprove={handleApproveAttendance}
        onReject={handleRejectAttendance}
        onCreditQuota={handleCreditAttendanceQuota}
        approveLoading={approveAttendanceMutation.isPending}
        rejectLoading={rejectAttendanceMutation.isPending}
        creditLoading={creditQuotaMutation.isPending}
      />
      <AdminCreateCheckInDialog
        open={isCreateCheckInOpen}
        onOpenChange={setIsCreateCheckInOpen}
      />
      <AttendanceMapDialog
        row={selectedAttendance}
        onClose={() => setSelectedAttendance(null)}
      />

      <PayrollReportEmailDialog
        open={isEmailDialogOpen}
        onOpenChange={setIsEmailDialogOpen}
        onSendEmail={handleSendEmail}
        isLoading={sendEmailMutation.isPending}
      />

      {/* Cancel confirmation — restores desktop's safety guard */}
      <AlertDialog
        open={cancelTargetId !== null}
        onOpenChange={(open) => !open && setCancelTargetId(null)}
      >
        <AlertDialogContent className="max-w-sm">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-sm">Hủy yêu cầu ứng lương?</AlertDialogTitle>
            <AlertDialogDescription className="text-xs">
              Yêu cầu sẽ được chuyển sang trạng thái đã hủy.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="min-h-11 text-xs" disabled={page.cancelMutation.isPending}>
              Giữ lại
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={handleConfirmCancel}
              disabled={page.cancelMutation.isPending}
              className="min-h-11 gap-1.5 text-xs"
            >
              {page.cancelMutation.isPending ? "Đang hủy..." : "Hủy yêu cầu"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </MobilePageShell>
  );
};

export default AdvancePaymentsPageMobile;
