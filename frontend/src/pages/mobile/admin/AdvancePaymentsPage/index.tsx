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
import { WalletBalanceCard } from "@/components/disbursement/WalletBalanceCard";
import { TimesheetMonthSelector } from "@/components/timesheet/TimesheetMonthSelector";
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
      successRate: summaryData?.successRate ?? 0,
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
      <div className="p-4 space-y-4">
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
    <div className="p-4 pb-20 space-y-4 max-w-full">

      {/* Header + actions */}
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

      {/* Month selector */}
      <div className="overflow-x-auto w-full">
        <TimesheetMonthSelector
          value={page.selectedMonth}
          onChange={page.setSelectedMonth}
        />
      </div>

      {/* Hero metrics — same components as desktop, already grid-cols-2 on mobile */}
      <div className="space-y-3">
        <AdvPartnerHeroStrip {...heroProps} isLoading={page.summaryLoading} />
        <AdvPartnerStatusOverview {...statusProps} isLoading={page.summaryLoading} />
        {!isAdvPartner && (
          <WalletBalanceCard
            monthlyProviderFee={page.providerFees.monthlyProviderFee}
            totalProviderFee={page.providerFees.totalProviderFee}
          />
        )}
      </div>

      {/* Requests list */}
      <div className="space-y-3">
        <div className="space-y-2">
          <MobileSearchInput
            value={page.searchInput}
            onSearch={page.handleSearch}
            placeholder="Tìm tên hoặc CCCD..."
            className="h-10"
          />
          {/* Horizontally scrollable filter bar with right-edge fade hint */}
          <div className="relative w-full">
            <div className="overflow-x-auto pb-0.5 scrollbar-none">
              <StatusFilterBar
                value={((page.filters.status as string) || "all") as import("@/types/api/advance-payment.types").AdvancePaymentRequestStatus | "all"}
                onChange={page.handleStatusChange}
                counts={page.statusCounts}
              />
            </div>
            <div className="pointer-events-none absolute inset-y-0 right-0 w-8 bg-gradient-to-l from-background to-transparent" />
          </div>
        </div>

        <AdvancePaymentMobileList
          requests={page.requests}
          isCancelling={page.cancelMutation.isPending}
          cancellingId={page.cancelMutation.variables ?? undefined}
          onCancel={(id) => page.cancelMutation.mutate(id)}
          pagination={page.pagination}
          onPageChange={page.handlePageChange}
        />
      </div>

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
    </div>
  );
};

export default AdvancePaymentsPageMobile;
