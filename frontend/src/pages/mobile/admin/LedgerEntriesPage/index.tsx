import { useState, useMemo, useEffect, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { LedgerPageHeaderMobile } from '@/components/ledger/mobile/LedgerPageHeaderMobile';
import { LedgerSummaryCardMobile } from '@/components/ledger/mobile/LedgerSummaryCardMobile';
import { LedgerMobileList } from '@/components/ledger/LedgerMobileList';
import { LedgerFiltersMobile } from '@/components/ledger/mobile/LedgerFiltersMobile';
import { DoubleEntryModal } from '@/components/ledger/DoubleEntryModal';
import { ReversalDialog } from '@/components/ledger/ReversalDialog';
import { PayrollReportEmailDialog, type PayrollReportEmailParams } from '@/components/timesheet/PayrollReportEmailDialog';
import { AdvancePaymentEmailDialog, type AdvancePaymentEmailParams } from '@/components/advance-payment/AdvancePaymentEmailDialog';
import { SaoKeHistoryDialog } from '@/components/transaction/SaoKeHistoryDialog';
import { useLedgerEntries } from '@/hooks/ledger/useLedgerEntries';
import { useOverallBalance, useCashFlowSummary, useLedgerSummary } from '@/hooks/ledger/useLedgerBalance';
import { useLedgerManagement } from '@/hooks/ledger/useLedgerManagement';
import { useLedgerMetadata } from '@/hooks/ledger/useLedgerMetadata';
import { useSendPayrollReportEmail } from '@/hooks/api/usePayrolls';
import { useSendReconciliationEmail } from '@/hooks/api/useAdvancePaymentReconciliation';
import { useProjects } from '@/hooks/api/useProjects';
import { useGeneralModals } from '@/hooks/useModalNavigation';
import { dateToString } from '@/utils/dateHelpers';
import type { LedgerFilters as FiltersType, LedgerEntry } from '@/types/api/financial.types';

const LedgerEntriesPageMobile = () => {
  const [searchParams] = useSearchParams();

  // Get last 1 year range for initial filters
  const getInitialDateRange = useMemo(() => {
    const now = new Date();
    const oneYearAgo = new Date(now.getFullYear() - 1, now.getMonth(), 1);
    const endOfCurrentMonth = new Date(now.getFullYear(), now.getMonth() + 1, 0);
    const dateRange = {
      fromDate: dateToString(oneYearAgo),
      toDate: dateToString(endOfCurrentMonth),
    };
    return dateRange;
  }, []);

  // State management
  const [filters, setFilters] = useState<FiltersType>(() => ({
    page: 1,
    pageSize: 50,
    sortBy: 'date',
    sortOrder: 'desc',
    fromDate: getInitialDateRange.fromDate,
    toDate: getInitialDateRange.toDate,
  }));
  const [showDoubleEntryModal, setShowDoubleEntryModal] = useState(false);
  const [reverseEntry, setReverseEntry] = useState<LedgerEntry | null>(null);

  // Sao kê dialog state
  const [payrollEmailDialogOpen, setPayrollEmailDialogOpen] = useState(false);
  const [advanceEmailDialogOpen, setAdvanceEmailDialogOpen] = useState(false);
  const [saoKeHistoryDialogOpen, setSaoKeHistoryDialogOpen] = useState(false);

  const sendPayrollEmailMutation = useSendPayrollReportEmail();
  const sendAdvanceEmailMutation = useSendReconciliationEmail();

  // Handle period filter parameter from dashboard navigation
  useEffect(() => {
    const period = searchParams.get('period');
    if (period) {
      const now = new Date();
      let newDateRange = { fromDate: '', toDate: '' };

      switch (period) {
        case 'current-month':
          newDateRange = {
            fromDate: dateToString(new Date(now.getFullYear(), now.getMonth(), 1)),
            toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
          };
          break;
        case 'salary':
          // For salary-related entries, show recent entries (last 3 months)
          newDateRange = {
            fromDate: dateToString(new Date(now.getFullYear(), now.getMonth() - 2, 1)),
            toDate: dateToString(new Date(now.getFullYear(), now.getMonth() + 1, 0)),
          };
          break;
        default:
          return;
      }

      setFilters(prev => ({
        ...prev,
        fromDate: newDateRange.fromDate,
        toDate: newDateRange.toDate,
        page: 1, // Reset pagination
      }));
    }
  }, [searchParams]);

  // Get current month date range for cash flow (keep as current month for cash flow summary)
  const currentMonth = useMemo(() => {
    const now = new Date();
    const firstDay = new Date(now.getFullYear(), now.getMonth(), 1);
    const lastDay = new Date(now.getFullYear(), now.getMonth() + 1, 0);
    return {
      fromDate: dateToString(firstDay),
      toDate: dateToString(lastDay),
    };
  }, []);

  // Data fetching
  const { data: entriesData, isLoading: isLoadingEntries } = useLedgerEntries(filters);
  const { data: overallBalance, isLoading: isLoadingBalance } = useOverallBalance();
  const { data: cashFlowData, isLoading: isLoadingCashFlow } = useCashFlowSummary(
    currentMonth.fromDate,
    currentMonth.toDate
  );
  const { data: ledgerSummary, isLoading: isLoadingSummary } = useLedgerSummary(
    filters.fromDate || getInitialDateRange.fromDate,
    filters.toDate || getInitialDateRange.toDate,
    filters.project_id,
    !!(filters.fromDate && filters.toDate)
  );

  const { data: projectsData } = useProjects();
  const { data: accountMetadata, isLoading: isLoadingAccountMetadata } = useLedgerMetadata();

  // CRUD operations
  const { reverseEntry: performReverse, recalculateBalances: performRecalculate, isReversing, isRecalculating } = useLedgerManagement();

  // Modal navigation
  const { openLedgerDetails, openAddLedgerEntry } = useGeneralModals();

  // Derived data - handle case where entriesData might be array directly
  const entries = Array.isArray(entriesData) ? entriesData : (entriesData?.data || []);
  const pagination = Array.isArray(entriesData)
    ? {
        page: filters.page,
        pageSize: filters.pageSize,
        totalPages: Math.ceil(entriesData.length / filters.pageSize) || 1,
        totalRecords: entriesData.length,
      }
    : entriesData?.pagination;
  const netCashFlow = cashFlowData?.net_cash_flow;
  const projects = projectsData?.data || [];

  // Handlers
  const handleAddEntry = () => {
    openAddLedgerEntry();
  };

  const handleReverseEntry = (entry: LedgerEntry) => {
    setReverseEntry(entry);
  };

  const handleConfirmReverse = async (entryId: number, reason: string) => {
    await performReverse({ id: entryId, reason });
    setReverseEntry(null);
  };

  const handleRecalculateBalance = async () => {
    try {
      await performRecalculate();
    } catch (error) {
      // Error is handled by the hook
    }
  };

  const handleRowClick = (entry: LedgerEntry) => {
    openLedgerDetails(entry.id.toString());
  };

  // Filter handlers
  const handleFiltersChange = (newFilters: FiltersType) => {
    setFilters(newFilters);
  };

  const handleClearFilters = () => {
    setFilters({
      page: 1,
      pageSize: 50,
      sortBy: 'date',
      sortOrder: 'desc',
      fromDate: getInitialDateRange.fromDate,
      toDate: getInitialDateRange.toDate,
    });
  };

  // Pagination handlers
  const handlePageChange = (page: number) => {
    setFilters(prev => ({ ...prev, page }));
  };

  const handlePageSizeChange = (pageSize: number) => {
    setFilters(prev => ({ ...prev, pageSize, page: 1 }));
  };

  // Sao kê handlers
  const handleSendPayrollEmail = useCallback(async (params: PayrollReportEmailParams) => {
    try {
      await sendPayrollEmailMutation.mutateAsync(params);
      setPayrollEmailDialogOpen(false);
    } catch { /* handled by mutation */ }
  }, [sendPayrollEmailMutation]);

  const handleSendAdvanceEmail = useCallback(async (params: AdvancePaymentEmailParams) => {
    try {
      await sendAdvanceEmailMutation.mutateAsync(params);
      setAdvanceEmailDialogOpen(false);
    } catch { /* handled by mutation */ }
  }, [sendAdvanceEmailMutation]);

  // Check if any filters are active (excluding default values)
  const hasFilters = !!(filters.project_id || filters.account || filters.party || filters.created_by || filters.has_evidence !== undefined);

  return (
    <div className="p-4 pb-20 space-y-4 max-w-full overflow-hidden">
      <LedgerPageHeaderMobile
        overallBalance={overallBalance}
        netCashFlow={netCashFlow}
        isLoadingBalance={isLoadingBalance}
        isLoadingCashFlow={isLoadingCashFlow}
        onAddEntry={handleAddEntry}
        onAddDoubleEntry={() => setShowDoubleEntryModal(true)}
        onRecalculateBalance={handleRecalculateBalance}
        isRecalculating={isRecalculating}
        onSendSaoKePayroll={() => setPayrollEmailDialogOpen(true)}
        onSendSaoKeAdvance={() => setAdvanceEmailDialogOpen(true)}
        onViewSaoKeHistory={() => setSaoKeHistoryDialogOpen(true)}
        isSendingSaoKe={sendPayrollEmailMutation.isPending || sendAdvanceEmailMutation.isPending}
      />

      <LedgerSummaryCardMobile
        summary={ledgerSummary}
        isLoading={isLoadingSummary}
      />

      <LedgerFiltersMobile
        filters={filters}
        onFiltersChange={handleFiltersChange}
        projects={projects}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
        totalResults={pagination?.totalRecords}
        accountMetadata={accountMetadata ?? []}
        isLoadingAccountMetadata={isLoadingAccountMetadata}
      />

      {!isLoadingAccountMetadata && accountMetadata && accountMetadata.length > 0 ? (
        <LedgerMobileList
          entries={entries}
          isLoading={isLoadingEntries}
          onRowClick={handleRowClick}
          projects={projects}
          accountMetadata={accountMetadata}
          pagination={pagination}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      ) : (
        <div className="flex justify-center items-center py-12 rounded-xl border border-border">          <p className="text-sm text-muted-foreground">Đang tải thông tin tài khoản...</p>
        </div>
      )}

      <DoubleEntryModal
        isOpen={showDoubleEntryModal}
        onClose={() => setShowDoubleEntryModal(false)}
        projects={projects}
      />

      <ReversalDialog
        isOpen={!!reverseEntry}
        onClose={() => setReverseEntry(null)}
        entry={reverseEntry}
        onConfirm={handleConfirmReverse}
        isLoading={isReversing}
      />

      {/* Sao kê dialogs */}
      <PayrollReportEmailDialog
        open={payrollEmailDialogOpen}
        onOpenChange={setPayrollEmailDialogOpen}
        onSendEmail={handleSendPayrollEmail}
        isLoading={sendPayrollEmailMutation.isPending}
      />
      <AdvancePaymentEmailDialog
        open={advanceEmailDialogOpen}
        onOpenChange={setAdvanceEmailDialogOpen}
        onSendEmail={handleSendAdvanceEmail}
        isLoading={sendAdvanceEmailMutation.isPending}
      />
      <SaoKeHistoryDialog
        open={saoKeHistoryDialogOpen}
        onOpenChange={setSaoKeHistoryDialogOpen}
      />
    </div>
  );
};

export default LedgerEntriesPageMobile;
