import { useState, useEffect, useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { TimesheetFilters } from "@/components/timesheet/TimesheetFilters";
import { TimesheetListTable } from "@/components/timesheet/TimesheetListTable";
import { TimesheetMobileList } from "@/components/timesheet/TimesheetMobileList";
import { TimesheetProvider } from "@/components/timesheet/TimesheetContext";
import { EditRequestTable } from "@/components/timesheet/EditRequestTable";
import { useIsMobile } from '@/hooks/useBreakpoint';
import {
  TimesheetsExportDialog,
  TimesheetsExportParams,
} from "@/components/timesheet/TimesheetsExportDialog";
import { ExportSaoKeDialog } from "@/components/transaction/ExportSaoKeDialog";
import { PaymentHistorySheet } from "@/components/payroll/PaymentHistorySheet";
import { Skeleton } from "@/components/ui/skeleton";
import { MissingBankDetailsSection } from "@/components/employees/MissingBankDetailsSection";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useTimesheetModals } from "@/hooks/useModalNavigation";
import { useExportApprovedTimesheets } from "@/hooks/api/usePayrolls";
import { useSettingByKey } from "@/hooks/api/useSettings";
import { useCreateEditRequest } from "@/hooks/api/useTimesheetEditRequests";
import { useTimesheetStatsConfig } from "@/hooks/useTimesheetStatsConfig";
import { ChevronLeft, ChevronRight, Plus, Download, History, Clock, FileUp, Users, ClipboardList, AlertCircle, Wallet, CheckCircle2, SlidersHorizontal, Files } from "lucide-react";
import { BCCUploadModal } from "@/components/timesheet/BCCUploadModal";
import { UploadHistorySheet } from "@/components/timesheet/UploadHistorySheet";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Timesheet } from "@/types/api/timesheet.types";

/* ------------------------------------------------------------------ */
/*  TimesheetStatsRow — watermark-style stat tiles                     */
/* ------------------------------------------------------------------ */

interface TimesheetStatsRowProps {
  isLoading: boolean;
  employeeCount: number;
  totalEntries: number;
  pendingCount: number;
  pendingEmployees: number;
  approvedCount: number;
  statusFilter: string;
  onToggleStatus: (status: string) => void;
}

function TimesheetStatsRow({
  isLoading,
  employeeCount,
  totalEntries,
  pendingCount,
  pendingEmployees,
  approvedCount,
  statusFilter,
  onToggleStatus,
}: TimesheetStatsRowProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-5 gap-2.5">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="rounded-xl border border-border/60 bg-card px-3 py-2.5">
            <div className="space-y-1.5">
              <div className="h-2.5 w-16 bg-muted rounded animate-pulse" />
              <div className="h-5 w-12 bg-muted rounded animate-pulse" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  const cells = [
    {
      key: "employees",
      label: "Nhân viên",
      value: employeeCount,
      icon: Users,
      iconText: "text-success",
      watermark: "text-success/15",
      active: false,
      onClick: undefined as (() => void) | undefined,
    },
    {
      key: "entries",
      label: "Tổng công",
      value: totalEntries,
      icon: ClipboardList,
      iconText: "text-muted-foreground",
      watermark: "text-muted-foreground/15",
      active: false,
      onClick: undefined,
    },
    {
      key: "pending_approval",
      label: "Chờ duyệt",
      value: pendingCount,
      icon: AlertCircle,
      iconText: pendingCount > 0 ? "text-warning" : "text-muted-foreground",
      watermark: pendingCount > 0 ? "text-warning/15" : "text-muted-foreground/10",
      active: statusFilter === "pending_approval",
      onClick: pendingCount > 0
        ? () => onToggleStatus(statusFilter === "pending_approval" ? "all" : "pending_approval")
        : undefined,
    },
    {
      key: "pending_payment",
      label: "NV chờ TT",
      value: pendingEmployees,
      icon: Wallet,
      iconText: pendingEmployees > 0 ? "text-info" : "text-muted-foreground",
      watermark: pendingEmployees > 0 ? "text-info/15" : "text-muted-foreground/10",
      active: statusFilter === "pending_payment",
      onClick: pendingEmployees > 0
        ? () => onToggleStatus(statusFilter === "pending_payment" ? "all" : "pending_payment")
        : undefined,
    },
    {
      key: "approved",
      label: "Đã duyệt",
      value: approvedCount,
      icon: CheckCircle2,
      iconText: approvedCount > 0 ? "text-success" : "text-muted-foreground",
      watermark: approvedCount > 0 ? "text-success/15" : "text-muted-foreground/10",
      active: statusFilter === "approved",
      onClick: approvedCount > 0
        ? () => onToggleStatus(statusFilter === "approved" ? "all" : "approved")
        : undefined,
    },
  ];

  return (
    <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-5 gap-2.5">
      {cells.map(({ key, label, value, icon: Icon, iconText, watermark, active, onClick }) => (
        <div
          key={key}
          onClick={onClick}
          role={onClick ? "button" : undefined}
          tabIndex={onClick ? 0 : undefined}
          onKeyDown={onClick ? (e) => { if (e.key === "Enter" || e.key === " ") onClick(); } : undefined}
          aria-pressed={onClick ? active : undefined}
          className={cn(
            "group relative rounded-2xl border bg-card px-3.5 py-3 overflow-hidden shadow-[0_8px_18px_-17px_rgba(15,23,42,0.42)] transition-all",
            active ? "border-primary/40 bg-primary/5" : "border-border/60",
            onClick && "cursor-pointer hover:-translate-y-0.5 hover:border-primary/25 hover:shadow-[0_14px_26px_-20px_rgba(6,101,52,0.38)]",
          )}
        >
          <Icon
            className={cn(
              "absolute right-2 top-1/2 -translate-y-1/2 h-10 w-10 pointer-events-none",
              "transition-transform duration-300 group-hover:scale-105",
              watermark,
            )}
            strokeWidth={1.5}
          />
          <div className="relative pr-10">
            <div className="flex items-center gap-1.5">
              <Icon className={cn("h-3 w-3 shrink-0", iconText)} strokeWidth={2.2} />
              <span className="text-[10px] font-semibold uppercase tracking-[0.06em] text-muted-foreground leading-tight truncate">
                {label}
              </span>
            </div>
            <p className={cn(
              "mt-1 break-words font-display text-[18px] font-bold tabular-nums leading-tight tracking-normal",
              active ? "text-primary" : "text-foreground",
            )}>
              {value.toLocaleString("vi-VN")}
            </p>
          </div>
          {active && (
            <div className="absolute top-2 right-2 h-1.5 w-1.5 rounded-full bg-primary z-10" />
          )}
        </div>
      ))}
    </div>
  );
}

export default function TimesheetsPage() {
  const [approvedTimesheetsDialogOpen, setApprovedTimesheetsDialogOpen] =
    useState(false);
  const [paymentHistorySheetOpen, setPaymentHistorySheetOpen] = useState(false);
  const [exportSaoKeOpen, setExportSaoKeOpen] = useState(false);
  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const { openTimesheetEntry, openTimesheetDetails } = useTimesheetModals();
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();

  const createEditRequestMutation = useCreateEditRequest();
  const [requestingTimesheetId, setRequestingTimesheetId] = useState<
    number | null
  >(null);

  const urlEmployeeId = searchParams.get("employee");
  const urlProjectId = searchParams.get("project");

  const timesheetManagement = useTimesheetManagement({
    userRole: "partner",
    useYearToDate: false,
  });

  const currentMonthStr = useMemo(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}`;
  }, []);

  useEffect(() => {
    if (timesheetManagement.selectedMonth === "all") {
      timesheetManagement.setSelectedMonth(currentMonthStr);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (urlProjectId && urlProjectId !== timesheetManagement.selectedProject) {
      timesheetManagement.setSelectedProject(urlProjectId);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const displayMonth = timesheetManagement.selectedMonth === "all"
    ? currentMonthStr
    : timesheetManagement.selectedMonth;

  const statsFilters = useMemo(() => ({
    project_id: timesheetManagement.selectedProject !== "all" ? parseInt(timesheetManagement.selectedProject) : undefined,
    fromDate: displayMonth ? `${displayMonth}-01` : undefined,
    toDate: displayMonth ? (() => {
      const lastDay = new Date(
        parseInt(displayMonth.split("-")[0]),
        parseInt(displayMonth.split("-")[1]),
        0,
      ).getDate();
      return `${displayMonth}-${lastDay.toString().padStart(2, "0")}`;
    })() : undefined,
  }), [timesheetManagement.selectedProject, displayMonth]);

  const timesheetStats = useTimesheetStatsConfig(statsFilters);

  const handlePrevMonth = useCallback(() => {
    const [y, m] = displayMonth.split("-").map(Number);
    const d = new Date(y, m - 2, 1);
    timesheetManagement.setSelectedMonth(
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`,
    );
  }, [displayMonth, timesheetManagement]);

  const handleNextMonth = useCallback(() => {
    const [y, m] = displayMonth.split("-").map(Number);
    const d = new Date(y, m, 1);
    timesheetManagement.setSelectedMonth(
      `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`,
    );
  }, [displayMonth, timesheetManagement]);

  const monthLabel = useMemo(() => {
    const [y, m] = displayMonth.split("-").map(Number);
    return `Tháng ${m} / ${y}`;
  }, [displayMonth]);
  const { data: bulkTransferSetting } = useSettingByKey(
    "bulk_transfer_payment_percentage",
  );
  const bulkTransferPercentage = bulkTransferSetting?.value
    ? parseFloat(bulkTransferSetting.value)
    : 0;

  const exportApprovedTimesheetsMutation = useExportApprovedTimesheets();

  useEffect(() => {
    if (
      urlEmployeeId &&
      urlEmployeeId !== timesheetManagement.selectedEmployee
    ) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const newParams = new URLSearchParams(searchParams);
    const isSpecific =
      timesheetManagement.selectedEmployee &&
      timesheetManagement.selectedEmployee !== "all" &&
      timesheetManagement.selectedEmployee.trim() !== "";
    if (isSpecific) {
      newParams.set("employee", timesheetManagement.selectedEmployee);
    } else {
      newParams.delete("employee");
      newParams.delete("view");
    }
    if (newParams.toString() !== searchParams.toString()) {
      setSearchParams(newParams, { replace: true });
    }
  }, [timesheetManagement.selectedEmployee, searchParams, setSearchParams]);

  const handleApprovedTimesheetsExportSubmit = useCallback(
    async (params: TimesheetsExportParams) => {
      try {
        await exportApprovedTimesheetsMutation.mutateAsync(params);
        setApprovedTimesheetsDialogOpen(false);
      } catch {
        /* handled by mutation */
      }
    },
    [exportApprovedTimesheetsMutation],
  );

  const handleDelete = useCallback(
    async (timesheet: Timesheet) => {
      await timesheetManagement.handleDelete(timesheet);
    },
    [timesheetManagement],
  );

  const handleRequestEdit = useCallback(
    async (timesheet: Timesheet, onSuccess?: () => Promise<void> | void) => {
      if (createEditRequestMutation.isPending) return;
      setRequestingTimesheetId(timesheet.id);
      try {
        await createEditRequestMutation.mutateAsync(timesheet.id);
        if (onSuccess) await onSuccess();
      } finally {
        setRequestingTimesheetId(null);
      }
    },
    [createEditRequestMutation],
  );

  const handleEditRequestRowClick = useCallback(
    (timesheet: Timesheet) => {
      queryClient.setQueryData(
        ["timesheets", "detail", timesheet.id],
        timesheet,
      );
      openTimesheetDetails(timesheet.id.toString());
    },
    [queryClient, openTimesheetDetails],
  );

  const summaryStats = useMemo(() => {
    const s = timesheetStats.summary;
    if (!s) return null;
    return {
      employeeCount: s.totalEmployees ?? 0,
      totalEntries: s.totalEntries,
      pendingCount: s.pendingApproval,
      pendingEmployees: s.pendingEmployees ?? 0,
      approvedCount: s.approvedEntries,
    };
  }, [timesheetStats.summary]);

  if (
    timesheetManagement.isLoading &&
    timesheetManagement.timesheets.length === 0
  ) {
    return (
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5 animate-fade-in">
        <div className="space-y-2">
          <Skeleton className="h-7 w-28" />
          <Skeleton className="h-4 w-56" />
        </div>
        <div className="flex gap-2">
          <Skeleton className="h-9 w-28 rounded-xl" />
          <Skeleton className="h-9 w-32 rounded-xl" />
          <Skeleton className="h-9 w-36 rounded-xl" />
        </div>
        <Skeleton className="h-11 w-full max-w-lg rounded-xl" />
        <div className="rounded-lg border bg-background overflow-hidden">
          <div className="border-b px-4 py-3">
            <div className="flex gap-6">
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} className="h-3 w-16" />
              ))}
            </div>
          </div>
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="border-b px-4 py-3 flex items-center gap-3">
              <div className="h-8 w-8 rounded-full bg-muted animate-pulse" />
              <div className="flex-1 space-y-2">
                <Skeleton className="h-3.5 w-32" />
                <Skeleton className="h-2.5 w-20" />
              </div>
              <Skeleton className="h-5 w-16 rounded-md" />
              <Skeleton className="h-3.5 w-12" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full bg-[radial-gradient(circle_at_100%_0%,rgba(8,120,62,0.12),transparent_29rem)] px-4 py-5 lg:px-8 lg:py-7">
      <div className="mx-auto max-w-[1320px] space-y-4">

        {/* Header */}
        <div className="relative overflow-hidden rounded-3xl border border-primary/15 bg-card p-5 shadow-[0_20px_54px_-42px_rgba(6,101,52,0.44)] opacity-0 animate-fade-in-up [animation-delay:50ms] [animation-fill-mode:forwards] md:p-6">
          <div className="pointer-events-none absolute -right-14 -top-16 h-48 w-48 rounded-full border-[26px] border-primary/15" />
          <div className="relative flex items-start justify-between gap-4 flex-wrap">
            <div>
              <h1 className="text-xl font-bold tracking-tight text-foreground flex items-center gap-2 sm:text-2xl">
                <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary ring-1 ring-primary/15">
                  <Clock className="h-5 w-5" />
                </span>
                Bảng công
              </h1>
              <p className="text-sm text-muted-foreground mt-1">Theo dõi bảng công, yêu cầu sửa và lịch sử chi trả theo tháng</p>
            </div>
            <div className="flex items-center gap-1 rounded-2xl border border-border/70 bg-muted/35 p-1 shadow-soft">
              <button
                onClick={handlePrevMonth}
                className="flex items-center justify-center h-10 w-10 rounded-xl text-muted-foreground hover:text-foreground hover:bg-card transition-colors"
                aria-label="Tháng trước"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <span className="min-w-[124px] text-center text-sm font-bold text-foreground tabular-nums px-1">
                {monthLabel}
              </span>
              <button
                onClick={handleNextMonth}
                className="flex items-center justify-center h-10 w-10 rounded-xl text-muted-foreground hover:text-foreground hover:bg-card transition-colors"
                aria-label="Tháng sau"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>

        {/* Stats — watermark-style stat tiles. 5 cells: 2-col on mobile, 5-col on lg+ */}
        {summaryStats && (
          <div className="opacity-0 animate-fade-in-up [animation-delay:100ms] [animation-fill-mode:forwards]">
            <TimesheetStatsRow
              isLoading={timesheetStats.isLoading}
              employeeCount={summaryStats.employeeCount}
              totalEntries={summaryStats.totalEntries}
              pendingCount={summaryStats.pendingCount}
              pendingEmployees={summaryStats.pendingEmployees}
              approvedCount={summaryStats.approvedCount}
              statusFilter={timesheetManagement.statusFilter}
              onToggleStatus={timesheetManagement.setStatusFilter}
            />
          </div>
        )}

        {/* Action buttons */}
        <div className="opacity-0 animate-fade-in-up [animation-delay:120ms] [animation-fill-mode:forwards] flex flex-wrap gap-2 items-center rounded-2xl border border-border/60 bg-card p-3 shadow-[0_10px_24px_-22px_rgba(15,23,42,0.46)]">
          <div className="hidden items-center gap-2 border-r border-border pr-3 lg:flex">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-primary"><Files className="h-4 w-4" /></div>
            <span className="text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">Tác vụ bảng công</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPaymentHistorySheetOpen(true)}
            className="gap-1.5"
          >
            <History className="h-3.5 w-3.5" />
            Lịch sử trả lương
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setApprovedTimesheetsDialogOpen(true)}
            className="gap-1.5"
          >
            <Download className="h-3.5 w-3.5" />
            Xuất bảng công
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setExportSaoKeOpen(true)}
            className="gap-1.5"
          >
            <Download className="h-3.5 w-3.5" />
            Xuất sao kê
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setBccHistoryOpen(true)}
            className="gap-1.5"
          >
            <History className="h-3.5 w-3.5" />
            Lịch sử BCC
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setBccUploadOpen(true)}
            className="gap-1.5"
          >
            <FileUp className="h-3.5 w-3.5" />
            Tải lên BCC
          </Button>
          <Button
            size="sm"
            onClick={() => openTimesheetEntry()}
            className="gap-1.5 sm:ml-auto"
          >
            <Plus className="h-3.5 w-3.5" />
            Nhập công
          </Button>
        </div>

        {/* Edit requests */}
        <EditRequestTable
          userRole="partner"
          onRowClick={handleEditRequestRowClick}
        />

        <MissingBankDetailsSection />

        {/* Filters + Table */}
        <TimesheetProvider
          management={timesheetManagement}
          onEdit={timesheetManagement.handleEdit}
          onDelete={handleDelete}
          userRole="partner"
          onRequestEdit={handleRequestEdit}
          requestingTimesheetId={requestingTimesheetId}
          bulkTransferPercentage={bulkTransferPercentage}
          onExportExcel={() => timesheetManagement.handleExportExcel()}
        >
          <div className="space-y-3">
            <div className="opacity-0 animate-fade-in-up [animation-delay:150ms] [animation-fill-mode:forwards]">
              <div className="flex items-center gap-2.5 flex-wrap rounded-2xl border border-border/60 bg-card px-3 py-3 shadow-[0_10px_24px_-22px_rgba(15,23,42,0.46)]">
                <div className="hidden items-center gap-2 border-r border-border pr-3 lg:flex">
                  <SlidersHorizontal className="h-4 w-4 text-primary" />
                  <span className="text-[11px] font-bold uppercase tracking-[0.08em] text-muted-foreground">Lọc dữ liệu</span>
                </div>
                <TimesheetFilters />
              </div>
            </div>

            <div className="overflow-hidden rounded-3xl border border-border/60 bg-card shadow-[0_14px_32px_-25px_rgba(15,23,42,0.50)] opacity-0 animate-fade-in-up [animation-delay:200ms] [animation-fill-mode:forwards]">
              <div className="flex items-center justify-between gap-3 border-b border-border/55 bg-muted/25 px-4 py-3 sm:px-5">
                <div>
                  <p className="text-[12px] font-bold text-foreground">Chi tiết bảng công</p>
                  <p className="mt-0.5 text-[10.5px] text-muted-foreground">Nhấn vào một dòng để xem hoặc xử lý</p>
                </div>
                <span className="rounded-full bg-muted px-2 py-1 text-[10px] font-bold tabular-nums text-muted-foreground">{timesheetManagement.timesheets.length} bản ghi</span>
              </div>
              {isMobile ? <TimesheetMobileList /> : <TimesheetListTable />}
            </div>
          </div>
        </TimesheetProvider>

      </div>

      <TimesheetsExportDialog
        open={approvedTimesheetsDialogOpen}
        onOpenChange={setApprovedTimesheetsDialogOpen}
        onExport={handleApprovedTimesheetsExportSubmit}
        isLoading={exportApprovedTimesheetsMutation.isPending}
      />

      <PaymentHistorySheet
        isOpen={paymentHistorySheetOpen}
        onClose={() => setPaymentHistorySheetOpen(false)}
      />

      <ExportSaoKeDialog
        open={exportSaoKeOpen}
        onOpenChange={setExportSaoKeOpen}
      />

      <BCCUploadModal
        open={bccUploadOpen}
        onClose={() => setBccUploadOpen(false)}
        projectId={
          timesheetManagement.selectedProject !== "all"
            ? parseInt(timesheetManagement.selectedProject, 10)
            : 0
        }
        projects={timesheetManagement.projects}
      />

      <UploadHistorySheet
        open={bccHistoryOpen}
        onClose={() => setBccHistoryOpen(false)}
        projectId={
          timesheetManagement.selectedProject !== "all"
            ? parseInt(timesheetManagement.selectedProject, 10)
            : undefined
        }
      />
    </div>
  );
}
