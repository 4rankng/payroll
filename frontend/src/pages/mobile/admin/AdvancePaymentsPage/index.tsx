import { useState, useCallback, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { StatusFilterBar } from "@/components/advance-payment/StatusFilterBar";
import { useExportAdvancePayments } from "@/hooks/api/useAdvancePayments";
import { useAdvancePaymentsPage } from "@/hooks/advance-payment/useAdvancePaymentsPage";
import { useSendPayrollReportEmail } from "@/hooks/transactions/useSendPayrollReportEmail";
import { FileDown, ArrowRightLeft, History, Mail, FileText } from "lucide-react";
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
import type { PayrollReportEmailParams } from "@/components/timesheet/PayrollReportEmailDialog";
import { useAuth } from "@/contexts";

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

  const page = useAdvancePaymentsPage({ employeesTabActive: false });
  const exportBatchMutation = useExportAdvancePayments();
  const sendEmailMutation = useSendPayrollReportEmail();

  const summaryData = page.summary?.data;

  const heroProps = useMemo(
    () => ({
      totalPaidAmount: summaryData?.totalPaidAmount ?? 0,
      totalAmount: summaryData?.totalAmount ?? 0,
      totalFeeEarned: summaryData?.totalFeeEarned ?? 0,
      totalPaid: summaryData?.totalPaid ?? 0,
      totalRequests: summaryData?.totalRequests ?? 0,
      totalCancelled: summaryData?.totalCancelled ?? 0,
      avgProcessingTimeSecs: summaryData?.avgProcessingTimeSecs ?? 0,
      completedUnder30s: summaryData?.completedUnder30s ?? 0,
      disbursementPercentage: summaryData?.disbursementPercentage ?? 0,
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
      totalPaid: summaryData?.totalPaid ?? 0,
      totalRequests: summaryData?.totalRequests ?? 0,
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

  if (page.summaryLoading && page.requests.length === 0) {
    return (
      <div className="p-4 pb-20 space-y-4 max-w-full overflow-hidden">
        <Skeleton className="h-10 w-full" />
        <div className="grid grid-cols-2 gap-px">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-none first:rounded-tl-xl [&:nth-child(2)]:rounded-tr-xl [&:nth-child(3)]:rounded-bl-xl last:rounded-br-xl" />
          ))}
        </div>
        <div className="grid grid-cols-2 gap-px">
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
                icon={FileDown}
                label="Chuyển lô"
                onClick={() => { exportBatchMutation.mutate(undefined); close(); }}
                disabled={exportBatchMutation.isPending}
                isLoading={exportBatchMutation.isPending}
              />
              {!isAdvPartner && (
                <MobileOverflowAction
                  icon={ArrowRightLeft}
                  label="Nhập KQ"
                  onClick={() => { setIsResultUploadOpen(true); close(); }}
                />
              )}
              <MobileOverflowAction
                icon={FileText}
                label="Sao kê"
                onClick={() => { setIsStatementSheetOpen(true); close(); }}
              />
              <MobileOverflowAction
                icon={Mail}
                label="Email sao kê"
                onClick={() => { setIsEmailDialogOpen(true); close(); }}
                disabled={sendEmailMutation.isPending}
                isLoading={sendEmailMutation.isPending}
              />
              <MobileOverflowDivider />
              {!isAdvPartner && (
                <MobileOverflowAction
                  icon={History}
                  label="Lịch sử file"
                  onClick={() => { setIsHistorySheetOpen(true); close(); }}
                />
              )}
            </>
          )}
        />

        <div className="mt-3 overflow-x-auto rounded-xl border border-[#D8E2EE] bg-[#F3F7FB] p-1">
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
            className="rounded-2xl border border-slate-900/10 shadow-[0_16px_42px_-30px_rgba(15,23,42,0.75)]"
          />
        )}

        <div className="grid grid-cols-1 gap-3">
          <AdvPartnerHeroStrip
            {...heroProps}
            compact
            isLoading={page.summaryLoading}
            className="rounded-2xl border-[#D8E2EE] bg-white"
          />
          <TreasuryFeePanel
            {...feePanelProps}
            compact
            isLoading={page.summaryLoading}
            className="rounded-2xl border border-[#D8E2EE] bg-white shadow-[0_1px_2px_rgba(15,23,42,0.05),0_14px_32px_-28px_rgba(15,49,103,0.55)]"
          />
        </div>
      </div>

      {/* Requests list */}
      <MobileSurface className="space-y-3 p-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-[10px] font-semibold uppercase tracking-[0.14em] text-slate-400">
              Yêu cầu
            </p>
            <h2 className="text-[15px] font-bold leading-tight text-slate-900">
              Danh sách ứng lương
            </h2>
          </div>
          <div className="grid h-8 min-w-8 place-items-center rounded-full bg-slate-100 px-2 text-xs font-semibold tabular-nums text-slate-600">
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
                triggerClassName="h-11 min-w-[108px] rounded-lg border-slate-200 bg-slate-50 px-3 text-[13px]"
              />
            </div>
          </div>
        </div>

        <AdvancePaymentMobileList
          requests={page.requests}
          isCancelling={page.cancelMutation.isPending}
          cancellingId={page.cancelMutation.variables ?? undefined}
          onCancel={(id) => page.cancelMutation.mutate(id)}
          onRetry={(id) => page.retryMutation.mutate(id)}
          isRetrying={page.retryMutation.isPending}
          retryingId={page.retryMutation.variables ?? undefined}
          pollingIds={page.pollingIds}
          pagination={page.pagination}
          onPageChange={page.handlePageChange}
        />
      </MobileSurface>

      <AdvPartnerStatusOverview
        {...statusProps}
        isLoading={page.summaryLoading}
        className="rounded-2xl border-[#D8E2EE] bg-white"
      />

      <ImportPayrollDialog open={isImportSheetOpen} onOpenChange={setIsImportSheetOpen} />
      <FlexibleEmployeeListUploadDialog open={isEmployeeListUploadOpen} onOpenChange={setIsEmployeeListUploadOpen} />
      <AdvancePaymentResultUploadDialog open={isResultUploadOpen} onOpenChange={setIsResultUploadOpen} />
      <StatementDialog open={isStatementSheetOpen} onOpenChange={setIsStatementSheetOpen} forMonth={page.flexPayMonth} />
      <FileHistorySheet open={isHistorySheetOpen} onOpenChange={setIsHistorySheetOpen} />

      <PayrollReportEmailDialog
        open={isEmailDialogOpen}
        onOpenChange={setIsEmailDialogOpen}
        onSendEmail={handleSendEmail}
        isLoading={sendEmailMutation.isPending}
      />
    </MobilePageShell>
  );
};

export default AdvancePaymentsPageMobile;
