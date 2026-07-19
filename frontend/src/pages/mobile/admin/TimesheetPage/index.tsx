import { useState, useMemo, useEffect, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams, useNavigate } from "react-router-dom";
import { TimesheetPageHeaderMobile } from "@/components/timesheet/mobile/TimesheetPageHeaderMobile";
import { TimesheetDisplaySection } from "@/components/timesheet/TimesheetDisplaySection";
import { ChuyenLoDialog } from "@/components/timesheet/ChuyenLoDialog";
import { BulkTransferExportDialog } from "@/components/timesheet/BulkTransferExportDialog";
import type { BulkTransferExportParams } from "@/services/api/bulk-transfer.service";
import {
  PayrollReportExportDialog,
  PayrollReportExportParams,
} from "@/components/timesheet/PayrollReportExportDialog";
import {
  TimesheetsExportDialog,
  TimesheetsExportParams,
} from "@/components/timesheet/TimesheetsExportDialog";
import { BulkTransferResultUploadDialog } from "@/components/timesheet/BulkTransferResultUploadDialog";
import { UploadHistorySheet } from "@/components/timesheet/UploadHistorySheet";
import { BCCUploadModal } from "@/components/timesheet/BCCUploadModal";
import { BulkTransferHistoryDialog } from "@/components/transaction/BulkTransferHistoryDialog";
import { BulkTransferHistoryDetailDialog } from "@/components/transaction/BulkTransferHistoryDetailDialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { MissingBankDetailsSection } from "@/components/employees/MissingBankDetailsSection";
import { useTimesheetManagement } from "@/hooks/timesheet/useTimesheetManagement";
import { useTimesheetStatsConfig } from "@/hooks/useTimesheetStatsConfig";
import {
  useExportBulkTransfer,
  useExportPayrollReport,
  useExportApprovedTimesheets,
} from "@/hooks/api/usePayrolls";
import {
  useApproveAllTimesheets,
  useCashReadiness,
  useTimesheetSummary,
} from "@/hooks/api/useTimesheets";
import { PayrollControlCenter } from "@/components/timesheet/PayrollControlCenter";
import { useModalNavigation } from "@/hooks/useModalNavigation";
import { useSettingByKey } from "@/hooks/api/useSettings";
import { MODAL_IDS } from "@/constants/modalRegistry";
import { toast } from "@/components/ui/sonner";
import { Skeleton } from "@/components/ui/skeleton";
import { MobileTimesheetEntry } from "@/components/sheets/timesheet-entry/mobile/MobileTimesheetEntry";
import { createProjectBulkApprovalContent } from "@/utils/timesheetBulkHelpers";
import type { Timesheet } from "@/types/api/timesheet.types";

const TimesheetPageMobile = () => {
  const [chuyenLoDialogOpen, setChuyenLoDialogOpen] = useState(false);
  const [bulkTransferDialogOpen, setBulkTransferDialogOpen] = useState(false);
  const [payrollReportDialogOpen, setPayrollReportDialogOpen] = useState(false);
  const [approvedTimesheetsDialogOpen, setApprovedTimesheetsDialogOpen] =
    useState(false);
  const [bulkTransferResultDialogOpen, setBulkTransferResultDialogOpen] =
    useState(false);
  const [bulkTransferHistoryDialogOpen, setBulkTransferHistoryDialogOpen] =
    useState(false);
  const [selectedHistoryId, setSelectedHistoryId] = useState<number | null>(
    null,
  );
  const [selectedHistoryFilename, setSelectedHistoryFilename] = useState("");
  const [selectedHistoryUploadedAt, setSelectedHistoryUploadedAt] = useState<
    string | null
  >(null);
  const [shouldLoadBulkTransferHistory, setShouldLoadBulkTransferHistory] =
    useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [bulkApproveDialogOpen, setBulkApproveDialogOpen] = useState(false);
  const [projectBulkApproveDialogOpen, setProjectBulkApproveDialogOpen] =
    useState(false);
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();

  const isEntryMode = searchParams.get("modal") === "timesheet_entry";
  const urlProjectId = searchParams.get("projectId");
  const entryEmployeeId = searchParams.get("employeeId");
  const urlEmployeeId = searchParams.get("employee");
  const urlStatus = searchParams.get("status");

  const handleEntryClose = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete("modal");
    params.delete("projectId");
    params.delete("employeeId");
    params.delete("entryId");
    navigate(`/admin/timesheet${params.toString() ? `?${params}` : ""}`, {
      replace: true,
    });
  }, [navigate, searchParams]);

  const timesheetManagement = useTimesheetManagement({ userRole: "admin" });
  const queryClient = useQueryClient();
  const { openModal } = useModalNavigation();
  const { data: bulkTransferSetting } = useSettingByKey(
    "bulk_transfer_payment_percentage",
  );
  const bulkTransferPercentage = bulkTransferSetting?.value
    ? parseFloat(bulkTransferSetting.value)
    : 0;

  useEffect(() => {
    if (
      urlEmployeeId &&
      urlEmployeeId !== timesheetManagement.selectedEmployee
    ) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    if (urlStatus && urlStatus !== timesheetManagement.statusFilter) {
      timesheetManagement.setStatusFilter(
        urlStatus as Timesheet["status"] | Timesheet["payment_status"],
      );
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
      newParams.set("view", "calendar");
    } else {
      newParams.delete("employee");
      newParams.delete("view");
    }
    if (newParams.toString() !== searchParams.toString()) {
      setSearchParams(newParams, { replace: true });
    }
  }, [timesheetManagement.selectedEmployee, searchParams, setSearchParams]);

  const statsFilters = useMemo(
    () => ({
      project_id:
        timesheetManagement.selectedProject !== "all"
          ? parseInt(timesheetManagement.selectedProject)
          : undefined,
      fromDate:
        timesheetManagement.selectedMonth !== "all"
          ? `${timesheetManagement.selectedMonth}-01`
          : undefined,
      toDate:
        timesheetManagement.selectedMonth !== "all"
          ? (() => {
              const lastDay = new Date(
                parseInt(timesheetManagement.selectedMonth.split("-")[0]),
                parseInt(timesheetManagement.selectedMonth.split("-")[1]),
                0,
              ).getDate();
              return `${timesheetManagement.selectedMonth}-${lastDay.toString().padStart(2, "0")}`;
            })()
          : undefined,
    }),
    [timesheetManagement.selectedProject, timesheetManagement.selectedMonth],
  );

  const timesheetStats = useTimesheetStatsConfig(statsFilters);

  // Pending edit-request count feeds the "Có vấn đề" KPI in the control center.
  const editCount = Number(timesheetStats.statsConfig.find((s) => s.title === "Yêu cầu sửa")?.value ?? 0);

  const exportBulkTransferMutation = useExportBulkTransfer();
  const exportPayrollReportMutation = useExportPayrollReport();
  const exportApprovedTimesheetsMutation = useExportApprovedTimesheets();
  const approveAllMutation = useApproveAllTimesheets();

  // Global (unfiltered) summary for the "Duyệt hết" confirm — backend
  // approveAll() ignores filters and approves system-wide, so we show the
  // honest, unfiltered scope here. Mirrors the desktop TimesheetPage.
  const { data: globalSummary, refetch: refetchGlobalSummary } =
    useTimesheetSummary({});
  const cashReadiness = useCashReadiness({});

  const handleAddTimesheet = useCallback(
    () => openModal(MODAL_IDS.TIMESHEET_ENTRY),
    [openModal],
  );

  const handleEditTimesheet = useCallback(
    (timesheet: Timesheet) => {
      openModal(MODAL_IDS.TIMESHEET_ENTRY, {
        projectId: timesheet.project_id?.toString(),
        employeeId: timesheet.employee_id?.toString(),
        entryId: timesheet.id?.toString(),
      });
    },
    [openModal],
  );

  const handleEditRequestRowClick = useCallback(
    (timesheet: Timesheet) => {
      queryClient.setQueryData(
        ["timesheets", "detail", timesheet.id],
        timesheet,
      );
      openModal(MODAL_IDS.TIMESHEET_DETAILS, { id: timesheet.id.toString() });
    },
    [queryClient, openModal],
  );

  const handleConfirmBulkApprove = async () => {
    try {
      await approveAllMutation.mutateAsync();
      setBulkApproveDialogOpen(false);
    } catch {
      /* handled by mutation */
    }
  };

  // Open the safe "Duyệt hết" flow — refresh the global summary first so the
  // confirm dialog shows accurate, unfiltered counts (matches desktop).
  const handleBulkApprove = useCallback(() => {
    refetchGlobalSummary();
    setBulkApproveDialogOpen(true);
  }, [refetchGlobalSummary]);

  const handleProjectBulkApprove = useCallback(() => {
    if (timesheetManagement.projectPendingTimesheets.length === 0) {
      toast({
        title: "Thông báo",
        description: "Không có bảng công pending nào cho dự án này.",
      });
      return;
    }
    setProjectBulkApproveDialogOpen(true);
  }, [timesheetManagement.projectPendingTimesheets]);

  const handleConfirmProjectBulkApprove = async () => {
    try {
      await timesheetManagement.handleProjectBulkApprove();
      setProjectBulkApproveDialogOpen(false);
      toast({ title: "Duyệt dự án thành công" });
    } catch {
      toast({
        title: "Lỗi",
        description: "Không thể duyệt bảng công dự án.",
        variant: "destructive",
      });
    }
  };

  const handleBulkTransferExportSubmit = async (
    params: BulkTransferExportParams,
  ) => {
    try {
      await exportBulkTransferMutation.mutateAsync(params);
      setBulkTransferDialogOpen(false);
    } catch {
      /* handled by mutation */
    }
  };

  const handlePayrollReportExportSubmit = async (
    params: PayrollReportExportParams,
  ) => {
    try {
      await exportPayrollReportMutation.mutateAsync(params);
      setPayrollReportDialogOpen(false);
    } catch {
      /* handled by mutation */
    }
  };

  const handleApprovedTimesheetsExportSubmit = async (
    params: TimesheetsExportParams,
  ) => {
    try {
      await exportApprovedTimesheetsMutation.mutateAsync(params);
      setApprovedTimesheetsDialogOpen(false);
    } catch {
      /* handled by mutation */
    }
  };

  const handleBulkTransferHistory = useCallback(() => {
    setBulkTransferHistoryDialogOpen(true);
    setSelectedHistoryId(null);
    setShouldLoadBulkTransferHistory(true);
  }, []);

  const handleBccHistory = useCallback(() => {
    setBccHistoryOpen(true);
  }, []);

  const handleCloseBccHistory = useCallback(() => {
    setBccHistoryOpen(false);
  }, []);

  const handleSelectHistory = useCallback(
    (id: number, filename: string, uploadedAt: string) => {
      setSelectedHistoryId(id);
      setSelectedHistoryFilename(filename);
      setSelectedHistoryUploadedAt(uploadedAt);
    },
    [],
  );

  const handleCloseHistory = useCallback(() => {
    setBulkTransferHistoryDialogOpen(false);
    setSelectedHistoryId(null);
    setSelectedHistoryFilename("");
    setSelectedHistoryUploadedAt(null);
    setShouldLoadBulkTransferHistory(false);
  }, []);

  if (isEntryMode) {
    return (
      <div className="flex min-h-[100dvh] flex-col overflow-x-clip bg-background pb-[env(safe-area-inset-bottom)]">
        <MobileTimesheetEntry
          isOpen={true}
          onClose={handleEntryClose}
          projectId={urlProjectId ? Number(urlProjectId) : undefined}
          employeeId={entryEmployeeId ? Number(entryEmployeeId) : undefined}
        />
        <div id="datepicker-portal" />
      </div>
    );
  }

  if (
    timesheetManagement.isLoading &&
    timesheetManagement.timesheets.length === 0
  ) {
    return (
      <div className="max-w-full space-y-4 overflow-hidden p-4 pb-[calc(5rem+env(safe-area-inset-bottom))]">
        <Skeleton className="h-10 w-full" />
        <div className="grid grid-cols-1 gap-3">
          {Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    );
  }

  return (
    <div className="max-w-full space-y-4 overflow-hidden p-4 pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <TimesheetPageHeaderMobile
        onAddTimesheet={handleAddTimesheet}
        onApprovedTimesheetsExport={() => setApprovedTimesheetsDialogOpen(true)}
        onPayrollReportExport={() => setPayrollReportDialogOpen(true)}
        onBulkTransferExport={() => setBulkTransferDialogOpen(true)}
        onBulkTransferResultUpload={() => setBulkTransferResultDialogOpen(true)}
        onBulkTransferHistory={handleBulkTransferHistory}
        onBulkApprove={handleBulkApprove}
        onChuyenLo={() => setChuyenLoDialogOpen(true)}
        onBccHistory={handleBccHistory}
        onBccUpload={() => setBccUploadOpen(true)}
        isApprovedExportPending={exportApprovedTimesheetsMutation.isPending}
        isPayrollReportPending={exportPayrollReportMutation.isPending}
        monthValue={timesheetManagement.selectedMonth}
        onMonthChange={timesheetManagement.setSelectedMonth}
      />

      {/* ── Payroll Control Center: cash readiness + operational KPIs ── */}
      <PayrollControlCenter
        cashReadiness={{
          data: cashReadiness.data,
          isLoading: cashReadiness.isLoading,
          isError: cashReadiness.isError,
        }}
        stats={{
          summary: timesheetStats.summary,
          editCount,
          isLoading: timesheetStats.isLoading,
        }}
        activeFilter={timesheetManagement.statusFilter}
        onFilterChange={timesheetManagement.setStatusFilter}
      />

      <MissingBankDetailsSection />

      <TimesheetDisplaySection
        timesheetManagement={timesheetManagement}
        onEdit={handleEditTimesheet}
        onApprove={timesheetManagement.handleApprove}
        onDelete={timesheetManagement.handleDelete}
        onAddTimesheet={handleAddTimesheet}
        onBulkApprove={handleBulkApprove}
        bulkTransferPercentage={bulkTransferPercentage}
        showEditRequestTable={true}
        userRole="admin"
        onEditRequestRowClick={handleEditRequestRowClick}
      />

      <BulkTransferExportDialog
        open={bulkTransferDialogOpen}
        onOpenChange={setBulkTransferDialogOpen}
        onExport={handleBulkTransferExportSubmit}
        isLoading={exportBulkTransferMutation.isPending}
      />
      <PayrollReportExportDialog
        open={payrollReportDialogOpen}
        onOpenChange={setPayrollReportDialogOpen}
        onExport={handlePayrollReportExportSubmit}
        isLoading={exportPayrollReportMutation.isPending}
      />
      <TimesheetsExportDialog
        open={approvedTimesheetsDialogOpen}
        onOpenChange={setApprovedTimesheetsDialogOpen}
        onExport={handleApprovedTimesheetsExportSubmit}
        isLoading={exportApprovedTimesheetsMutation.isPending}
      />
      <BulkTransferResultUploadDialog
        open={bulkTransferResultDialogOpen}
        onOpenChange={setBulkTransferResultDialogOpen}
      />
      <BulkTransferHistoryDialog
        open={bulkTransferHistoryDialogOpen && selectedHistoryId === null}
        onOpenChange={handleCloseHistory}
        onSelectHistory={handleSelectHistory}
        shouldFetchHistories={shouldLoadBulkTransferHistory}
      />
      <BulkTransferHistoryDetailDialog
        open={bulkTransferHistoryDialogOpen && selectedHistoryId !== null}
        historyId={selectedHistoryId}
        filename={selectedHistoryFilename}
        uploadedAt={selectedHistoryUploadedAt || undefined}
        onOpenChange={handleCloseHistory}
        onBack={() => {
          setSelectedHistoryId(null);
          setSelectedHistoryFilename("");
          setSelectedHistoryUploadedAt(null);
        }}
      />
      {/* "Chuyển lô" — unified batch-transfer dialog (now on mobile, matching desktop) */}
      <ChuyenLoDialog open={chuyenLoDialogOpen} onOpenChange={setChuyenLoDialogOpen} />

      <ConfirmDialog
        open={bulkApproveDialogOpen}
        onOpenChange={setBulkApproveDialogOpen}
        title="Duyệt hết bảng công"
        description={
          <div className="space-y-0 divide-y divide-border/60">
            <div className="flex flex-col gap-1 py-2 text-sm min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
              <span className="text-muted-foreground">Bảng công chờ duyệt</span>
              <span className="font-semibold tabular-nums min-[380px]:text-right">
                {globalSummary?.pendingApproval ?? 0}
              </span>
            </div>
            <div className="flex flex-col gap-1 py-2 text-sm min-[380px]:flex-row min-[380px]:items-center min-[380px]:justify-between">
              <span className="text-muted-foreground">Số NV liên quan</span>
              <span className="font-semibold tabular-nums min-[380px]:text-right">
                {globalSummary?.pendingEmployees ?? "—"}
              </span>
            </div>
          </div>
        }
        confirmText="Duyệt hết"
        onConfirm={handleConfirmBulkApprove}
        loading={approveAllMutation.isPending}
        disabled={(globalSummary?.pendingApproval ?? 0) === 0}
      />
      <ConfirmDialog
        open={projectBulkApproveDialogOpen}
        onOpenChange={setProjectBulkApproveDialogOpen}
        title="Duyệt bảng công theo dự án"
        description={createProjectBulkApprovalContent(
          timesheetManagement.selectedProjectDetails?.name || "Dự án đã chọn",
          timesheetManagement.projectPendingTimesheets,
        )}
        confirmText="Duyệt dự án"
        onConfirm={handleConfirmProjectBulkApprove}
        loading={timesheetManagement.isLoading}
      />

      {/* BCC Upload */}
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

      {/* BCC Upload History */}
      <UploadHistorySheet
        open={bccHistoryOpen}
        onClose={handleCloseBccHistory}
        projects={timesheetManagement.projects}
      />
    </div>
  );
};

export default TimesheetPageMobile;
