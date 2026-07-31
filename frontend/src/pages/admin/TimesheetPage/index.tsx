import { useState, useMemo, useEffect, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { Plus, FileText, ArrowRightLeft, FileUp, History, MoreVertical, CheckCheck, FileSpreadsheet, Banknote, Trash2 } from 'lucide-react';
import { PageHeader } from '@/components/shared/PageHeader';
import { MissingBankDetailsSection } from '@/components/employees/MissingBankDetailsSection';
import { TimesheetDisplaySection } from '@/components/timesheet/TimesheetDisplaySection';
import { ChuyenLoDialog } from '@/components/timesheet/ChuyenLoDialog';
import { BulkTransferExportDialog } from '@/components/timesheet/BulkTransferExportDialog';
import { TimesheetsExportDialog, TimesheetsExportParams } from '@/components/timesheet/TimesheetsExportDialog';
import { BulkTransferResultUploadDialog } from '@/components/timesheet/BulkTransferResultUploadDialog';
import { BulkTransferHistoryDialog } from '@/components/transaction/BulkTransferHistoryDialog';
import { UploadHistorySheet } from '@/components/timesheet/UploadHistorySheet';
import { BCCUploadModal } from '@/components/timesheet/BCCUploadModal';
import { RejectUnpaidTimesheetsDialog } from '@/components/timesheet/RejectUnpaidTimesheetsDialog';
import { BulkTransferHistoryDetailDialog } from '@/components/transaction/BulkTransferHistoryDetailDialog';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useTimesheetManagement } from '@/hooks/timesheet/useTimesheetManagement';
import { useExportApprovedTimesheets } from '@/hooks/api/usePayrolls';
import { useApproveAllTimesheets, useCashReadiness, useTimesheetSummary } from '@/hooks/api/useTimesheets';
import { useExportOnePayBulk } from '@/hooks/api/useOnePayExport';
import { PayrollControlCenter } from '@/components/timesheet/PayrollControlCenter';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useSettingByKey } from '@/hooks/api/useSettings';
import { MODAL_IDS } from '@/constants/modalRegistry';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { Skeleton } from '@/components/ui/skeleton';
import type { Timesheet } from '@/types/api/timesheet.types';
import type { BulkTransferExportParams } from '@/services/api/bulk-transfer.service';

const TimesheetPage = () => {
  const [chuyenLoDialogOpen, setChuyenLoDialogOpen] = useState(false);
  const [onePayDialogOpen, setOnePayDialogOpen] = useState(false);
  const [approvedTimesheetsDialogOpen, setApprovedTimesheetsDialogOpen] = useState(false);
  const [bulkApproveDialogOpen, setBulkApproveDialogOpen] = useState(false);
  const [bulkTransferResultDialogOpen, setBulkTransferResultDialogOpen] = useState(false);
  const [bulkTransferHistoryDialogOpen, setBulkTransferHistoryDialogOpen] = useState(false);
  const [selectedHistoryId, setSelectedHistoryId] = useState<number | null>(null);
  const [selectedHistoryFilename, setSelectedHistoryFilename] = useState<string>('');
  const [selectedHistoryUploadedAt, setSelectedHistoryUploadedAt] = useState<string | null>(null);
  const [shouldLoadBulkTransferHistory, setShouldLoadBulkTransferHistory] = useState(false);
  const [bccHistoryOpen, setBccHistoryOpen] = useState(false);
  const [bccUploadOpen, setBccUploadOpen] = useState(false);
  const [rejectUnpaidDialogOpen, setRejectUnpaidDialogOpen] = useState(false);
  const [searchParams, setSearchParams] = useSearchParams();

  // Get initial values from URL parameters
  const urlEmployeeId = searchParams.get('employee');
  const urlStatus = searchParams.get('status');

  const timesheetManagement = useTimesheetManagement({ userRole: 'admin' });
  const queryClient = useQueryClient();
  const { openModal } = useModalNavigation();
  const { data: bulkTransferSetting } = useSettingByKey('bulk_transfer_payment_percentage');

  // Calculate bulk transfer percentage from setting
  const bulkTransferPercentage = bulkTransferSetting?.value ? parseFloat(bulkTransferSetting.value) : 0;

  // Global summary (no filters) for "Duyệt hết" confirm dialog — backend
  // approveAll() ignores filters and approves system-wide, so we show the
  // honest, unfiltered scope here.
  const { data: globalSummary, refetch: refetchGlobalSummary } = useTimesheetSummary({});
  const cashReadiness = useCashReadiness({});

  // Initialize employee and status selection from URL on mount only
  useEffect(() => {
    if (urlEmployeeId && urlEmployeeId !== timesheetManagement.selectedEmployee) {
      timesheetManagement.setSelectedEmployee(urlEmployeeId);
    }
    if (urlStatus && urlStatus !== timesheetManagement.statusFilter) {
      timesheetManagement.setStatusFilter(urlStatus as Timesheet['status'] | Timesheet['payment_status']);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update URL when employee selection changes
  useEffect(() => {
    const newParams = new URLSearchParams(searchParams);
    const isSpecific = timesheetManagement.selectedEmployee &&
                       timesheetManagement.selectedEmployee !== 'all' &&
                       timesheetManagement.selectedEmployee.trim() !== '';
    if (isSpecific) {
      newParams.set('employee', timesheetManagement.selectedEmployee);
    } else {
      newParams.delete('employee');
      newParams.delete('view');
    }
    if (newParams.toString() !== searchParams.toString()) {
      setSearchParams(newParams, { replace: true });
    }
  }, [timesheetManagement.selectedEmployee, searchParams, setSearchParams]);


  // Use the same filters for stats config to ensure consistent date range
  const statsFilters = useMemo(() => ({
    project_id: timesheetManagement.selectedProject !== 'all' ? parseInt(timesheetManagement.selectedProject) : undefined,
    fromDate: timesheetManagement.selectedMonth !== 'all' ? `${timesheetManagement.selectedMonth}-01` : undefined,
    toDate: timesheetManagement.selectedMonth !== 'all' ? (() => {
      const lastDay = new Date(
        parseInt(timesheetManagement.selectedMonth.split('-')[0]),
        parseInt(timesheetManagement.selectedMonth.split('-')[1]),
        0
      ).getDate();
      return `${timesheetManagement.selectedMonth}-${lastDay.toString().padStart(2, '0')}`;
    })() : undefined,
  }), [timesheetManagement.selectedProject, timesheetManagement.selectedMonth]);

  const { data: timesheetSummary, isLoading: isTimesheetSummaryLoading } = useTimesheetSummary(statsFilters);

  const exportApprovedTimesheetsMutation = useExportApprovedTimesheets();
  const approveAllMutation = useApproveAllTimesheets();
  const exportOnePayMutation = useExportOnePayBulk();

  const handleAddTimesheet = () => {
    openModal(MODAL_IDS.TIMESHEET_ENTRY);
  };

  const handleEditTimesheet = (timesheet: Timesheet) => {
    openModal(MODAL_IDS.TIMESHEET_ENTRY, {
      projectId: timesheet.project_id?.toString(),
      employeeId: timesheet.employee_id?.toString(),
      entryId: timesheet.id?.toString()
    });
  };

  const handleEditRequestRowClick = (timesheet: Timesheet) => {
    queryClient.setQueryData(['timesheets', 'detail', timesheet.id], timesheet);
    openModal(MODAL_IDS.TIMESHEET_DETAILS, {
      id: timesheet.id.toString()
    });
  };




  const handleBulkApprove = () => {
    refetchGlobalSummary();
    setBulkApproveDialogOpen(true);
  };

  const handleConfirmBulkApprove = async () => {
    try {
      await approveAllMutation.mutateAsync();
      setBulkApproveDialogOpen(false);
    } catch {
      // Error is handled by the mutation
    }
  };

  // "Chuyển lô" primary button — opens the unified tabbed dialog
  const handleChuyenLo = () => {
    setChuyenLoDialogOpen(true);
  };

  const handleApprovedTimesheetsExport = () => {
    setApprovedTimesheetsDialogOpen(true);
  };

  const handleApprovedTimesheetsExportSubmit = async (params: TimesheetsExportParams) => {
    try {
      await exportApprovedTimesheetsMutation.mutateAsync(params);
      setApprovedTimesheetsDialogOpen(false);
    } catch {
      // Error is handled by the mutation's onError callback
    }
  };

  // OnePay uses weekly date ranges. Sending for_month switches the backend
  // to the monthly employee cohort, so this action reuses the weekly period
  // selector from "Chuyển lô" without exposing the monthly toggle.
  const handleChuyenOnePay = () => {
    setOnePayDialogOpen(true);
  };

  const handleChuyenOnePaySubmit = async (params: BulkTransferExportParams) => {
    try {
      await exportOnePayMutation.mutateAsync(params);
      setOnePayDialogOpen(false);
    } catch {
      // Error + toast handled by useExportOnePayBulk onError.
      // Keep the selected period in place so the admin can retry.
    }
  };

  const onePayProjectIds = useMemo(() => {
    if (timesheetManagement.selectedProject === 'all') return [];
    const projectId = parseInt(timesheetManagement.selectedProject, 10);
    return Number.isNaN(projectId) ? [] : [projectId];
  }, [timesheetManagement.selectedProject]);

  const handleApprove = async (timesheet: Timesheet) => {
    await timesheetManagement.handleApprove(timesheet);
  };

  const handleDelete = async (timesheet: Timesheet) => {
    await timesheetManagement.handleDelete(timesheet);
  };

  const handleBulkTransferResultUpload = () => {
    setBulkTransferResultDialogOpen(true);
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

  const handleSelectHistory = useCallback((id: number, filename: string, uploadedAt: string) => {
    setSelectedHistoryId(id);
    setSelectedHistoryFilename(filename);
    setSelectedHistoryUploadedAt(uploadedAt);
  }, []);

  const handleBackToHistoryList = useCallback(() => {
    setSelectedHistoryId(null);
    setSelectedHistoryFilename('');
    setSelectedHistoryUploadedAt(null);
  }, []);

  const handleCloseHistory = useCallback(() => {
    setBulkTransferHistoryDialogOpen(false);
    setSelectedHistoryId(null);
    setSelectedHistoryFilename('');
    setSelectedHistoryUploadedAt(null);
    setShouldLoadBulkTransferHistory(false);
  }, []);


  if (timesheetManagement.isLoading && timesheetManagement.timesheets.length === 0) {
    return (
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1.5">
            <Skeleton className="h-7 w-40" />
            <Skeleton className="h-4 w-56" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-9 w-24" />
            <Skeleton className="h-9 w-9" />
          </div>
        </div>
        <Skeleton className="h-12 w-full rounded-xl" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-80" />
      </div>
    );
  }

  return (
    <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
      {/* ── Header ── */}
      <PageHeader
        title="Bảng công"
        description="Theo dõi và duyệt bảng công"
      >
        <TimesheetMonthSelector
          value={timesheetManagement.selectedMonth}
          onChange={timesheetManagement.setSelectedMonth}
        />
        <div className="flex items-center gap-px rounded-xl border border-border overflow-hidden">
          <button
            onClick={handleBulkApprove}
            className="inline-flex items-center gap-1.5 h-8 px-3 bg-background text-foreground text-sm font-medium whitespace-nowrap hover:bg-muted transition-colors border-r border-border"
          >
            <CheckCheck className="h-4 w-4 shrink-0" />
            Duyệt hết
          </button>
          <button
            onClick={handleChuyenLo}
            className="inline-flex items-center gap-1.5 h-8 px-3 bg-primary text-primary-foreground text-sm font-medium whitespace-nowrap hover:bg-primary/90 transition-colors border-r border-border"
          >
            <ArrowRightLeft className="h-4 w-4 shrink-0" />
            Chuyển lô
          </button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                aria-label="Thêm tùy chọn"
                className="inline-flex items-center justify-center h-8 w-8 bg-background text-foreground hover:bg-muted transition-colors"
              >
                <MoreVertical className="h-4 w-4" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-52">
              {/* Group 1: Nhập / Input */}
              <DropdownMenuItem onClick={handleAddTimesheet}>
                <Plus className="w-4 h-4 mr-2" />
                Nhập công
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleBulkTransferResultUpload}>
                <FileUp className="w-4 h-4 mr-2" />
                Nhập KQ chuyển lô
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {/* Group 2: Import / Upload */}
              <DropdownMenuItem onClick={() => setBccUploadOpen(true)}>
                <FileUp className="w-4 h-4 mr-2" />
                Tải lên BCC
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              {/* Group 3: Báo cáo / Reports */}
              <DropdownMenuItem onClick={handleChuyenOnePay} disabled={exportOnePayMutation.isPending}>
                <Banknote className="w-4 h-4 mr-2" />
                {exportOnePayMutation.isPending ? 'Đang xuất...' : 'Chuyển OnePay'}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleApprovedTimesheetsExport} disabled={exportApprovedTimesheetsMutation.isPending}>
                <FileText className="w-4 h-4 mr-2" />
                {exportApprovedTimesheetsMutation.isPending ? 'Đang xuất...' : 'Xuất bảng công'}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleBulkTransferHistory}>
                <History className="w-4 h-4 mr-2" />
                Lịch sử chuyển lô
              </DropdownMenuItem>
              <DropdownMenuItem onClick={handleBccHistory}>
                <FileSpreadsheet className="w-4 h-4 mr-2" />
                Lịch sử BCC
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => setRejectUnpaidDialogOpen(true)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="w-4 h-4 mr-2" />
                Loại công
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </PageHeader>

      {/* ── Payroll Control Center: cash readiness + operational KPIs ── */}
      <PayrollControlCenter
        cashReadiness={{
          data: cashReadiness.data,
          isLoading: cashReadiness.isLoading,
          isError: cashReadiness.isError,
        }}
        stats={{
          summary: timesheetSummary,
          isLoading: isTimesheetSummaryLoading,
        }}
        activeFilter={timesheetManagement.statusFilter}
        onFilterChange={timesheetManagement.setStatusFilter}
      />

      <MissingBankDetailsSection />

      <TimesheetDisplaySection
        timesheetManagement={timesheetManagement}
        onEdit={handleEditTimesheet}
        onApprove={handleApprove}
        onDelete={handleDelete}
        onAddTimesheet={handleAddTimesheet}
        onBulkApprove={handleBulkApprove}
        bulkTransferPercentage={bulkTransferPercentage}
        showEditRequestTable={true}
        userRole="admin"
        onEditRequestRowClick={handleEditRequestRowClick}
      />

      <ChuyenLoDialog
        open={chuyenLoDialogOpen}
        onOpenChange={setChuyenLoDialogOpen}
      />

      <BulkTransferExportDialog
        open={onePayDialogOpen}
        onOpenChange={setOnePayDialogOpen}
        onExport={handleChuyenOnePaySubmit}
        isLoading={exportOnePayMutation.isPending}
        mode="onepay"
        preselectedProjectIds={onePayProjectIds}
      />

      <TimesheetsExportDialog
        open={approvedTimesheetsDialogOpen}
        onOpenChange={setApprovedTimesheetsDialogOpen}
        onExport={handleApprovedTimesheetsExportSubmit}
        isLoading={exportApprovedTimesheetsMutation.isPending}
      />

      <ConfirmDialog
        open={bulkApproveDialogOpen}
        onOpenChange={setBulkApproveDialogOpen}
        title="Duyệt hết bảng công"
        description={
          <div className="space-y-0 divide-y divide-border/60">
            <div className="flex items-center justify-between py-2 text-sm">
              <span className="text-muted-foreground">Bảng công chờ duyệt</span>
              <span className="font-semibold tabular-nums">{globalSummary?.pendingApproval ?? 0}</span>
            </div>
            <div className="flex items-center justify-between py-2 text-sm">
              <span className="text-muted-foreground">Số NV liên quan</span>
              <span className="font-semibold tabular-nums">{globalSummary?.pendingEmployees ?? '—'}</span>
            </div>
          </div>
        }
        confirmText="Duyệt"
        onConfirm={handleConfirmBulkApprove}
        loading={approveAllMutation.isPending}
        disabled={(globalSummary?.pendingApproval ?? 0) === 0}
      />

      {/* Bulk Transfer Result Upload Dialog */}
      <BulkTransferResultUploadDialog
        open={bulkTransferResultDialogOpen}
        onOpenChange={setBulkTransferResultDialogOpen}
      />

      {/* Bulk Transfer History List Dialog */}
      <BulkTransferHistoryDialog
        open={bulkTransferHistoryDialogOpen && selectedHistoryId === null}
        onOpenChange={handleCloseHistory}
        onSelectHistory={handleSelectHistory}
        shouldFetchHistories={shouldLoadBulkTransferHistory}
      />

      {/* Bulk Transfer History Detail Dialog */}
      <BulkTransferHistoryDetailDialog
        open={bulkTransferHistoryDialogOpen && selectedHistoryId !== null}
        historyId={selectedHistoryId}
        filename={selectedHistoryFilename}
        uploadedAt={selectedHistoryUploadedAt || undefined}
        onOpenChange={handleCloseHistory}
        onBack={handleBackToHistoryList}
      />

      {/* BCC Upload */}
      <BCCUploadModal
        open={bccUploadOpen}
        onClose={() => setBccUploadOpen(false)}
        projectId={
          timesheetManagement.selectedProject !== 'all'
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

      <RejectUnpaidTimesheetsDialog
        open={rejectUnpaidDialogOpen}
        onOpenChange={setRejectUnpaidDialogOpen}
        initialProjectId={
          timesheetManagement.selectedProject !== 'all'
            ? Number(timesheetManagement.selectedProject)
            : undefined
        }
      />
    </div>
  );
};

export default TimesheetPage;
