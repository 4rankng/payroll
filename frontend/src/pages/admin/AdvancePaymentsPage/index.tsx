import { useState, useMemo, useCallback, memo } from "react";
import { useQuery } from "@tanstack/react-query";
import { ResponsiveTable } from "@/components/ui/responsive-table";
import { PageHeader } from "@/components/shared/PageHeader";
import { WalletBalanceCard } from "@/components/disbursement/WalletBalanceCard";
import { WalletDemandChart } from "@/components/wallet/WalletDemandChart";
import { WalletDemandCard } from "@/components/wallet/WalletDemandCard";
import { QueryKeys } from "@/lib/queryKeys";
import { walletService } from "@/services/api/wallet.service";
import { TimesheetMonthSelector } from "@/components/timesheet/TimesheetMonthSelector";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Wallet, Calendar, Users, Banknote } from "lucide-react";
import { StatusFilterBar } from "@/components/advance-payment/StatusFilterBar";
import { SearchBar } from "@/components/shared/SearchBar";
import { formatMonthDisplay } from "@/utils/advancePaymentHelpers";
import { useExportAdvancePayments } from "@/hooks/api/useAdvancePayments";
import { useAdvancePaymentsPage } from "@/hooks/advance-payment/useAdvancePaymentsPage";
import { useAdminAttendancePage } from "@/hooks/advance-payment/useAdminAttendancePage";
import { ActionBar, ButtonGroup, ImportAction, ExportListAction, ExportBatchAction, UploadResultAction, CheckInAction, HistoryAction } from "@/components/advance-payment/actions";
import { ImportPayrollDialog } from "@/components/advance-payment/ImportPayrollDialog";
import { AdvancePaymentResultUploadDialog } from "@/components/advance-payment/AdvancePaymentResultUploadDialog";
import { FlexibleEmployeeListUploadDialog } from "@/components/advance-payment/FlexibleEmployeeListUploadDialog";
import { CheckInBulkDialog } from "@/components/advance-payment/CheckInBulkDialog";
import { FileHistorySheet } from "@/components/advance-payment/FileHistorySheet";
import { EmployeeAdvancePaymentDetailSheet } from "@/components/advance-payment/EmployeeAdvancePaymentDetailSheet";
import {
  getAdvancePaymentColumns,
  getFlexPayColumns,
  requestMobileFields,
  flexPayMobileFields,
  requestEmptyState,
  flexPayEmptyState,
} from "@/components/advance-payment/table-config";
import {
  getAdminAttendanceColumns,
  attendanceMobileFields,
  attendanceEmptyState,
} from "@/components/attendance/AdminAttendanceTableConfig";
import { AttendanceMapDialog } from "@/components/attendance/AttendanceMapDialog";
import {
  AttendanceReviewDialogs,
  type ReviewMode,
} from "@/components/attendance/AttendanceReviewDialogs";
import {
  useApproveAttendance,
  useRejectAttendance,
} from "@/hooks/api/useAdminAttendance";
import type { AdminAttendanceResponse } from "@/types/api/attendance.types";
import { AdvPartnerHeroStrip } from "@/components/advance-payment/AdvPartnerHeroStrip";
import { AdvPartnerStatusOverview } from "@/components/advance-payment/AdvPartnerStatusOverview";
import { AdvPartnerMetricsStrip } from "@/components/advance-payment/AdvPartnerMetricsStrip";
import { TreasuryFeePanel } from "@/components/advance-payment/TreasuryFeePanel";
import { TabBarWithBadges } from "@/components/shared/TabBarWithBadges";
import { useAuth } from "@/contexts";
import { useIsMobile } from "@/hooks/useBreakpoint";
import { useMobilePageAnimations } from "@/hooks/useMobilePageAnimations";
import { cn } from "@/lib/utils";
import type {
  AdvancePaymentListItem,
  FlexPayEmployeeListItem,
  AdvancePaymentRequestStatus,
} from "@/types/api/advance-payment.types";
import type { ActiveTab } from "./types";

const AdvancePaymentsPage = () => {
  const [activeTab, setActiveTab] = useState<ActiveTab>("requests");
  const [isImportSheetOpen, setIsImportSheetOpen] = useState(false);
  const [isHistorySheetOpen, setIsHistorySheetOpen] = useState(false);
  const [isUploadResultDialogOpen, setIsUploadResultDialogOpen] = useState(false);
  const [isEmployeeListUploadDialogOpen, setIsEmployeeListUploadDialogOpen] =
    useState(false);
  const [isCheckInDialogOpen, setIsCheckInDialogOpen] = useState(false);
  const [selectedEmployee, setSelectedEmployee] =
    useState<FlexPayEmployeeListItem | null>(null);
  const [selectedAttendance, setSelectedAttendance] =
    useState<AdminAttendanceResponse | null>(null);
  const [reviewMode, setReviewMode] = useState<ReviewMode>(null);
  const [reviewRow, setReviewRow] = useState<AdminAttendanceResponse | null>(null);

  const approveAttendanceMutation = useApproveAttendance();
  const rejectAttendanceMutation = useRejectAttendance();

  const { user } = useAuth();
  const isAdvPartner = user?.role === "adv_partner";
  const isMobile = useIsMobile();
  const animRoot = useMobilePageAnimations();

  const page = useAdvancePaymentsPage({ employeesTabActive: activeTab === "employees" });
  const attendancePage = useAdminAttendancePage({ active: activeTab === "attendances" });
  const exportBatchMutation = useExportAdvancePayments();

  // Demand forecast (advisory) — admin only. adv_partner has no wallet context;
  // gating the query prevents a 403 storm on the shared mobile component.
  const { data: demandForecast, isLoading: demandForecastLoading } = useQuery({
    queryKey: QueryKeys.wallet.demandForecast(),
    queryFn: () => walletService.getDemandForecast(),
    enabled: !isAdvPartner,
    staleTime: 5 * 60_000,
    refetchInterval: 5 * 60_000,
  });

  const summaryData = page.summary?.data;

  const heroProps = useMemo(
    () => ({
      totalAmount: summaryData?.totalAmount ?? 0,
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

  const metricsProps = useMemo(
    () => ({
      totalPaid: summaryData?.totalPaid ?? 0,
      totalRequests: summaryData?.totalRequests ?? 0,
      totalCancelled: summaryData?.totalCancelled ?? 0,
      avgProcessingTimeSecs: summaryData?.avgProcessingTimeSecs ?? 0,
      completedUnder30s: summaryData?.completedUnder30s ?? 0,
      feePercentage: summaryData?.feePercentage ?? 0,
      avgFeePerRequest: summaryData?.avgFeePerRequest ?? 0,
      successRate: summaryData?.successRate ?? 0,
    }),
    [summaryData],
  );

  const feePanelProps = useMemo(
    () => ({
      totalFeeEarned: summaryData?.totalFeeEarned ?? 0,
      totalPaid: summaryData?.totalPaid ?? 0,
      totalRequests: summaryData?.totalRequests ?? 0,
      feePercentage: summaryData?.feePercentage ?? 0,
      avgFeePerRequest: summaryData?.avgFeePerRequest ?? 0,
      avgFeePerEmployee: summaryData?.avgFeePerEmployee ?? 0,
    }),
    [summaryData],
  );

  const columns = useMemo(
    () => getAdvancePaymentColumns({
      onCancel: page.handleCancelRequest,
      onRetry: (id: number) => page.retryMutation.mutate(id),
      isRetrying: page.retryMutation.isPending,
      retryingId: page.retryMutation.variables ?? undefined,
      pollingIds: page.pollingIds,
    }),
    [page.handleCancelRequest, page.retryMutation, page.pollingIds],
  );

  const flexPayColumns = useMemo(
    () =>
      getFlexPayColumns({
        sortBy: page.flexPayFilters.sortBy,
        sortOrder: page.flexPayFilters.sortOrder,
        onSort: page.handleFlexPaySort,
      }),
    [page.flexPayFilters.sortBy, page.flexPayFilters.sortOrder, page.handleFlexPaySort],
  );

  const attendanceActions = useMemo(
    () => ({
      onViewMap: (row: AdminAttendanceResponse) => setSelectedAttendance(row),
      onApprove: (row: AdminAttendanceResponse) => {
        setReviewRow(row);
        setReviewMode("approve");
      },
      onReject: (row: AdminAttendanceResponse) => {
        setReviewRow(row);
        setReviewMode("reject");
      },
    }),
    [],
  );

  const attendanceColumns = useMemo(
    () => getAdminAttendanceColumns(attendanceActions),
    [attendanceActions],
  );

  const handleReviewClose = useCallback(() => {
    setReviewMode(null);
    setReviewRow(null);
  }, []);

  const handleApproveAttendance = useCallback(
    (row: AdminAttendanceResponse, note: string) => {
      approveAttendanceMutation.mutate(
        { id: row.id, note },
        {
          onSuccess: () => handleReviewClose(),
        },
      );
    },
    [approveAttendanceMutation, handleReviewClose],
  );

  const handleRejectAttendance = useCallback(
    (row: AdminAttendanceResponse, note: string) => {
      rejectAttendanceMutation.mutate(
        { id: row.id, note },
        {
          onSuccess: () => handleReviewClose(),
        },
      );
    },
    [rejectAttendanceMutation, handleReviewClose],
  );

  const handleEmployeeClose = useCallback(() => setSelectedEmployee(null), []);

  return (
    <div ref={animRoot} className="min-h-full bg-[radial-gradient(circle_at_top_left,rgba(8,120,62,0.07),transparent_32rem),linear-gradient(180deg,rgba(248,250,249,0.96),#f5f7f9_34rem)]">
      <div className="mx-auto max-w-[1480px] space-y-4 p-4 lg:space-y-5 lg:p-6">

        {/* ─── MOBILE: Wallet hero — full-bleed, above everything ─── */}
        {isMobile && !isAdvPartner && (
          <div
            data-mobile-wallet
            className="mobile-wallet-hero relative -mx-4 -mt-4 overflow-hidden"
          >
            <WalletBalanceCard
              monthlyProviderFee={page.providerFees.monthlyProviderFee}
              totalProviderFee={page.providerFees.totalProviderFee}
              className="rounded-none border-0 shadow-none"
            />
            {/* Shimmer overlay */}
            <div className="wallet-shimmer-bg pointer-events-none absolute inset-0" />
          </div>
        )}

        {/* ─── 1. Page header ─── */}
        <div
          data-mobile-header
          className="rounded-2xl border border-white/80 bg-white/82 px-4 py-3 shadow-[0_1px_2px_rgba(16,24,40,0.05),0_18px_48px_-32px_rgba(8,120,62,0.22)] backdrop-blur sm:px-5"
        >
          <PageHeader
            title="Quản lý ứng lương"
            description="Xem và quản lý các yêu cầu ứng lương của nhân viên"
            icon={Banknote}
          >
            <TimesheetMonthSelector
              value={page.selectedMonth}
              onChange={page.setSelectedMonth}
            />
          </PageHeader>
        </div>

        {/* ─── 2. Treasury Hero — Wallet | Flow | Fee ─── */}
        <section
          aria-label="Tổng quan kỳ ứng lương"
          className={cn(
            "overflow-hidden rounded-2xl border border-slate-200/80 bg-white",
            "shadow-[0_1px_2px_rgba(16,24,40,0.05),0_22px_60px_-42px_rgba(8,120,62,0.30)]",
            isMobile && "mobile-section-enter",
          )}
        >
          <div className={cn(
            "grid divide-border/50",
            isAdvPartner
              ? "grid-cols-1 divide-y lg:grid-cols-[1fr_300px] lg:divide-y-0 lg:divide-x"
              : "grid-cols-1 divide-y lg:grid-cols-[280px_1fr_300px] xl:grid-cols-[320px_1fr_340px] lg:divide-y-0 lg:divide-x",
          )}>

            {/* Panel A: Wallet — admins only (desktop only; mobile shows it above) */}
            {!isAdvPartner && (
              <div className={cn(isMobile && "hidden")}>
                <WalletBalanceCard
                  monthlyProviderFee={page.providerFees.monthlyProviderFee}
                  totalProviderFee={page.providerFees.totalProviderFee}
                  className="rounded-none border-0 shadow-none"
                />
              </div>
            )}

            {/* Panel B: Disbursement flow */}
            <AdvPartnerHeroStrip
              {...heroProps}
              isLoading={page.summaryLoading}
              className="border-0 shadow-none"
            />

            {/* Panel C: Fee earned */}
            <TreasuryFeePanel
              {...feePanelProps}
              isLoading={page.summaryLoading}
            />
          </div>
        </section>

        {/* ─── Demand forecast: cohort chart + recommendation (admin only) ─── */}
        {!isAdvPartner && (
          <section
            aria-label="Nhu cầu ứng lương"
            className="grid grid-cols-1 gap-4 items-stretch md:grid-cols-3"
          >
            <div className="md:col-span-2">
              <WalletDemandChart data={demandForecast} isLoading={demandForecastLoading} />
            </div>
            <WalletDemandCard data={demandForecast} />
          </section>
        )}

        {/* ─── 3. Pipeline + operating health ─── */}
        <section
          data-mobile-stats
          aria-label="Trạng thái xử lý ứng lương"
          className="grid gap-4 xl:grid-cols-[minmax(0,1.45fr)_minmax(360px,0.75fr)]"
        >
          <AdvPartnerStatusOverview
            {...statusProps}
            isLoading={page.summaryLoading}
          />
          <AdvPartnerMetricsStrip
            {...metricsProps}
            isLoading={page.summaryLoading}
            className="sm:grid-cols-3 xl:grid-cols-1"
          />
        </section>

        {/* ─── 5. Operations — tabs + filters + table ─── */}
        <section
          data-mobile-content
          className={cn(
            "overflow-hidden rounded-2xl border border-slate-200/80 bg-white",
            "shadow-[0_1px_2px_rgba(16,24,40,0.04),0_20px_56px_-42px_rgba(8,120,62,0.26)]",
            isMobile && "mobile-section-enter",
          )}
        >

          {/* Header row: tabs + actions */}
          <div
            data-mobile-tabs
            className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200/80 bg-slate-50/80 px-3 py-3 sm:px-5"
          >
            <TabBarWithBadges
              tabs={[
                {
                  id: "requests",
                  label: "Yêu cầu",
                  icon: Wallet,
                  count: page.statusCounts?.all ?? 0,
                },
                {
                  id: "employees",
                  label: "Nhân viên",
                  icon: Users,
                  count: page.flexPayPagination?.totalRecords ?? 0,
                },
                {
                  id: "attendances",
                  label: "Chấm công",
                  icon: Calendar,
                  count: attendancePage.totalRecords ?? 0,
                },
              ]}
              activeTab={activeTab}
              onTabChange={(id) => setActiveTab(id as ActiveTab)}
            />

            <ActionBar>
              <ButtonGroup>
                <ImportAction onClick={() => setIsImportSheetOpen(true)} />
                {page.handleExportFlexPayEmployees && (
                  <ExportListAction
                    onClick={page.handleExportFlexPayEmployees}
                    isLoading={page.exportFlexPayMutation.isPending}
                  />
                )}
              </ButtonGroup>
              {!isAdvPartner && (
                <ButtonGroup>
                  <ExportBatchAction
                    onClick={() => exportBatchMutation.mutate(undefined)}
                    isLoading={exportBatchMutation.isPending}
                  />
                  <UploadResultAction onClick={() => setIsUploadResultDialogOpen(true)} />
                </ButtonGroup>
              )}
              <CheckInAction onClick={() => setIsCheckInDialogOpen(true)} />
              {!isAdvPartner && (
                <HistoryAction onClick={() => setIsHistorySheetOpen(true)} />
              )}
            </ActionBar>
          </div>

          {/* Filter row */}
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-slate-200/70 bg-white px-3 py-3 sm:px-5">
            <p className="text-sm font-semibold text-slate-800">
              {activeTab === "requests" && "Yêu cầu UL"}
              {activeTab === "employees" && "DS nhân viên"}
              {activeTab === "attendances" && "Điểm danh"}
            </p>
            <div className="flex flex-1 flex-wrap items-center justify-end gap-2">
            {activeTab === "requests" && (
              <>
                <SearchBar
                  searchTerm={page.searchInput}
                  onSearchChange={page.handleSearch}
                  placeholder="Tên hoặc CCCD..."
                  className="h-9 w-full min-w-[220px] sm:w-64"
                />
                <StatusFilterBar
                  value={(page.filters.status as AdvancePaymentRequestStatus) || "all"}
                  onChange={page.handleStatusChange}
                  counts={page.statusCounts}
                />
              </>
            )}

            {activeTab === "employees" && (
              <>
                <SearchBar
                  searchTerm={page.flexPaySearchInput}
                  onSearchChange={page.handleFlexPaySearch}
                  placeholder="Tên hoặc CCCD..."
                  className="h-9 w-full min-w-[220px] sm:w-64"
                />
                <Select
                  value={page.selectedViewMonth ?? ""}
                  onValueChange={(v) => page.setSelectedViewMonth(v || undefined)}
                >
                  <SelectTrigger className="h-9 w-auto min-w-36 shrink-0 text-sm">
                    <Calendar className="mr-1 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    <SelectValue placeholder="Tháng" />
                  </SelectTrigger>
                  <SelectContent>
                    {page.availableMonths.length === 0 ? (
                      <SelectItem value="none" disabled>
                        Chưa có dữ liệu
                      </SelectItem>
                    ) : (
                      page.availableMonths.map((m) => (
                        <SelectItem key={m.forMonth} value={m.forMonth} className="text-sm">
                          {formatMonthDisplay(m.forMonth)}{!isAdvPartner && ` (${m.employeeCount})`}
                        </SelectItem>
                      ))
                    )}
                  </SelectContent>
                </Select>
              </>
            )}

            {activeTab === "attendances" && (
              <>
                <Select
                  value={attendancePage.filters.status || "all"}
                  onValueChange={attendancePage.handleStatusChange}
                >
                  <SelectTrigger className="h-9 w-auto min-w-36 shrink-0 text-sm bg-white">
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
              </>
            )}
            </div>
          </div>

          {/* Table */}
          {activeTab === "requests" && (
            <ResponsiveTable
              data={page.requests}
              columns={columns}
              mobileFields={requestMobileFields}
              embedded
              rowTitle={(row: AdvancePaymentListItem) => (
                <div className="text-[15px] font-semibold text-slate-800">{row.employeeName}</div>
              )}
              rowSubtitle={(row: AdvancePaymentListItem) => (
                <div>
                  <div className="text-[13px] font-medium text-slate-500">{row.employeeCCCD}</div>
                  <div className="mt-0.5 max-w-[160px] truncate text-[13px] text-slate-400">{row.projectName || "-"}</div>
                </div>
              )}
              getRowId={(row: AdvancePaymentListItem) => row.id.toString()}
              pagination={page.pagination}
              onPageChange={page.handlePageChange}
              onPageSizeChange={page.handlePageSizeChange}
              emptyState={requestEmptyState}
            />
          )}

          {activeTab === "employees" && (
            <ResponsiveTable
              data={page.flexPayEmployees}
              columns={flexPayColumns}
              mobileFields={flexPayMobileFields}
              embedded
              rowTitle={(row: FlexPayEmployeeListItem) => (
                <div className="text-[15px] font-semibold text-slate-800">{row.fullname}</div>
              )}
              rowSubtitle={(row: FlexPayEmployeeListItem) => (
                <div>
                  <div className="text-[13px] font-medium text-slate-500">{row.cccd}</div>
                  <div className="mt-0.5 max-w-[160px] truncate text-[13px] text-slate-400">{row.project?.name || "-"}</div>
                </div>
              )}
              getRowId={(row: FlexPayEmployeeListItem) =>
                `${row.employeeId}-${row.project.id}`
              }
              onRowClick={(row: FlexPayEmployeeListItem) => setSelectedEmployee(row)}
              pagination={page.flexPayPagination}
              onPageChange={page.handleFlexPayPageChange}
              onPageSizeChange={page.handleFlexPayPageSizeChange}
              emptyState={flexPayEmptyState}
            />
          )}

          {activeTab === "attendances" && (
            <ResponsiveTable
              data={attendancePage.attendances}
              columns={attendanceColumns}
              mobileFields={attendanceMobileFields}
              embedded
              rowTitle={(row: AdminAttendanceResponse) => (
                <div className="text-[15px] font-semibold text-slate-800">{row.employee_name}</div>
              )}
              rowSubtitle={(row: AdminAttendanceResponse) => (
                <div>
                  <div className="mt-0.5 max-w-[160px] truncate text-[13px] text-slate-400">{row.project_name || "-"}</div>
                </div>
              )}
              getRowId={(row: AdminAttendanceResponse) => row.id.toString()}
              pagination={attendancePage.pagination ? {
                page: attendancePage.pagination.page,
                pageSize: attendancePage.pagination.limit,
                totalPages: attendancePage.pagination.totalPages,
                totalRecords: attendancePage.pagination.totalRecords,
              } : undefined}
              onPageChange={attendancePage.handlePageChange}
              onPageSizeChange={attendancePage.handlePageSizeChange}
              emptyState={
                <div className="text-center py-12">
                  <h3 className="mt-4 text-lg font-semibold">{attendanceEmptyState.title}</h3>
                  <p className="mt-2 text-sm text-muted-foreground">{attendanceEmptyState.description}</p>
                </div>
              }
            />
          )}
        </section>

        {/* ─── Dialogs & Sheets ─── */}
        <ImportPayrollDialog
          open={isImportSheetOpen}
          onOpenChange={setIsImportSheetOpen}
        />
        <AdvancePaymentResultUploadDialog
          open={isUploadResultDialogOpen}
          onOpenChange={setIsUploadResultDialogOpen}
        />
        <FileHistorySheet
          open={isHistorySheetOpen}
          onOpenChange={setIsHistorySheetOpen}
        />

        <EmployeeAdvancePaymentDetailSheet
          employee={selectedEmployee}
          onClose={handleEmployeeClose}
        />

        <CheckInBulkDialog
          open={isCheckInDialogOpen}
          onOpenChange={setIsCheckInDialogOpen}
        />

        <AlertDialog
          open={page.cancelConfirmId !== null}
          onOpenChange={(open) => {
            if (!open) page.dismissCancelConfirm();
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Hủy yêu cầu ứng lương?</AlertDialogTitle>
              <AlertDialogDescription>
                Yêu cầu ứng lương sẽ bị hủy vĩnh viễn.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Giữ lại</AlertDialogCancel>
              <AlertDialogAction
                variant="destructive"
                onClick={page.confirmCancelRequest}
              >
                {page.cancelMutation.isPending ? "Đang hủy..." : "Hủy yêu cầu"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {/* Attendance review (approve/reject) + map dialogs */}
        <AttendanceReviewDialogs
          mode={reviewMode}
          row={reviewRow}
          onClose={handleReviewClose}
          onApprove={handleApproveAttendance}
          onReject={handleRejectAttendance}
          approveLoading={approveAttendanceMutation.isPending}
          rejectLoading={rejectAttendanceMutation.isPending}
        />
        <AttendanceMapDialog
          row={selectedAttendance}
          onClose={() => setSelectedAttendance(null)}
        />
      </div>
    </div>
  );
};

export default AdvancePaymentsPage;
