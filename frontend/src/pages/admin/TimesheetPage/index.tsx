import { useState, useMemo, useEffect, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useSearchParams } from 'react-router-dom';
import { Plus, FileText, ArrowRightLeft, FileUp, History, MoreVertical, CheckCheck, BarChart3, AlertCircle, CheckCircle2, FileSpreadsheet } from 'lucide-react';
import { Masonry } from 'masonic';
import { PageHeader } from '@/components/shared/PageHeader';
import { MissingBankDetailsSection } from '@/components/employees/MissingBankDetailsSection';
import { TimesheetDisplaySection } from '@/components/timesheet/TimesheetDisplaySection';
import { ChuyenLoDialog } from '@/components/timesheet/ChuyenLoDialog';
import { TimesheetsExportDialog, TimesheetsExportParams } from '@/components/timesheet/TimesheetsExportDialog';
import { BulkTransferResultUploadDialog } from '@/components/timesheet/BulkTransferResultUploadDialog';
import { BulkTransferHistoryDialog } from '@/components/transaction/BulkTransferHistoryDialog';
import { UploadHistorySheet } from '@/components/timesheet/UploadHistorySheet';
import { BCCUploadModal } from '@/components/timesheet/BCCUploadModal';
import { BulkTransferHistoryDetailDialog } from '@/components/transaction/BulkTransferHistoryDetailDialog';
import { ConfirmDialog } from '@/components/ui/confirm-dialog';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { useTimesheetManagement } from '@/hooks/timesheet/useTimesheetManagement';
import { useTimesheetStatsConfig } from '@/hooks/useTimesheetStatsConfig';
import { useExportApprovedTimesheets } from '@/hooks/api/usePayrolls';
import { useApproveAllTimesheets, useTimesheetSummary } from '@/hooks/api/useTimesheets';
import { useModalNavigation } from '@/hooks/useModalNavigation';
import { useSettingByKey } from '@/hooks/api/useSettings';
import { MODAL_IDS } from '@/constants/modalRegistry';
import { InlineStatStrip } from '@/components/shared/InlineStatStrip';
import { TimesheetMonthSelector } from '@/components/timesheet/TimesheetMonthSelector';
import { Skeleton } from '@/components/ui/skeleton';
import { formatCurrency } from '@/utils/formatters';
import type { Timesheet } from '@/types/api/timesheet.types';

// ── masonry ────────────────────────────────────────────────────────────────

interface StatCardData {
  id: string;
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  isLoading: boolean;
  items: Array<{ label: string; value: string | number; highlight: boolean }>;
}

const StatMasonryCard = ({ data }: { index: number; data: StatCardData; width: number }) => (
  <div className="w-full space-y-2">
    <div className="flex items-center gap-2 px-0.5">
      <div className="flex h-7 w-7 items-center justify-center rounded-xl bg-primary/5 border border-primary/10">
        <data.icon className="h-3.5 w-3.5 text-primary/70" />
      </div>
      <span className="text-xs font-bold uppercase tracking-wider text-foreground">{data.label}</span>
    </div>
    <InlineStatStrip isLoading={data.isLoading} items={data.items} wrap={false} direction="vertical" />
  </div>
);


const TimesheetPage = () => {
  const [chuyenLoDialogOpen, setChuyenLoDialogOpen] = useState(false);
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

  const timesheetStats = useTimesheetStatsConfig(statsFilters);

  // Split into three logical rows by category
  const overviewRow = useMemo(() => {
    const s = timesheetStats.summary;
    const paidAmount = s?.paidAmount;
    return [
      { label: 'Đã thanh toán', value: paidAmount ? formatCurrency(paidAmount) : formatCurrency(0), highlight: false },
      { label: 'NV đã thanh toán', value: s?.paidEmployees ?? 0, highlight: false },
    ];
  }, [timesheetStats.summary]);

  const pendingRow = useMemo(() => {
    const s = timesheetStats.summary;
    const editCount = timesheetStats.statsConfig.find(c => c.title === 'Yêu cầu sửa')?.value ?? 0;
    const pendingAmount = s?.pendingPaymentAmount;
    const sf = timesheetManagement.statusFilter;
    return [
      { label: 'Chờ duyệt', value: s?.pendingApproval ?? 0, highlight: true, onClick: () => timesheetManagement.setStatusFilter(sf === 'pending_approval' ? 'all' : 'pending_approval') },
      { label: 'NV chờ thanh toán', value: s?.pendingEmployees ?? 0, highlight: true, onClick: () => timesheetManagement.setStatusFilter(sf === 'pending_payment' ? 'all' : 'pending_payment') },
      { label: 'Chờ thanh toán', value: pendingAmount ? formatCurrency(pendingAmount) : formatCurrency(0), highlight: pendingAmount > 0 },
      { label: 'Yêu cầu sửa', value: editCount as number, highlight: (editCount as number) > 0 },
    ];
  }, [timesheetStats.summary, timesheetStats.statsConfig, timesheetManagement]);

  const completedRow = useMemo(() => {
    const s = timesheetStats.summary;
    const sf = timesheetManagement.statusFilter;
    return [
      { label: 'Đã duyệt', value: s?.approvedEntries ?? 0, highlight: false, onClick: () => timesheetManagement.setStatusFilter(sf === 'approved' ? 'all' : 'approved') },
      { label: 'Đã thanh toán', value: s?.paidEntries ?? 0, highlight: false, onClick: () => timesheetManagement.setStatusFilter(sf === 'paid' ? 'all' : 'paid') },
      { label: 'Bị loại', value: s?.rejectedEntries ?? 0, highlight: false, onClick: () => timesheetManagement.setStatusFilter(sf === 'rejected' ? 'all' : 'rejected') },
    ];
  }, [timesheetStats.summary, timesheetManagement]);

  const statCards = useMemo<StatCardData[]>(() => [
    { id: 'overview', icon: BarChart3, label: 'Tổng quan', isLoading: timesheetStats.isLoading, items: overviewRow },
    { id: 'pending', icon: AlertCircle, label: 'Cần xử lý', isLoading: timesheetStats.isLoading, items: pendingRow },
    { id: 'completed', icon: CheckCircle2, label: 'Hoàn tất', isLoading: timesheetStats.isLoading, items: completedRow },
  ], [timesheetStats.isLoading, overviewRow, pendingRow, completedRow]);

  const exportApprovedTimesheetsMutation = useExportApprovedTimesheets();
  const approveAllMutation = useApproveAllTimesheets();

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
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </PageHeader>

      {/* ── Stat strips ── */}
      <Masonry
        items={statCards}
        render={StatMasonryCard}
        columnWidth={280}
        columnGutter={16}
        rowGutter={16}
        maxColumnCount={3}
        overscanBy={Infinity}
        itemKey={data => data.id}
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
    </div>
  );
};

export default TimesheetPage;
