import { useState, useMemo, useEffect, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { LedgerPageHeader } from '@/components/ledger/LedgerPageHeader';
import { LedgerSummaryCard } from '@/components/ledger/LedgerSummaryCard';
import { LedgerMobileList } from '@/components/ledger/LedgerMobileList';
import { TransactionFilters } from '@/components/transaction/TransactionFilters';
import { MobilePageShell } from '@/components/shared/MobilePageShell';
import { DoubleEntryModal } from '@/components/ledger/DoubleEntryModal';
import { PayrollReportEmailDialog, type PayrollReportEmailParams } from '@/components/timesheet/PayrollReportEmailDialog';
import { AdvancePaymentEmailDialog, type AdvancePaymentEmailParams } from '@/components/advance-payment/AdvancePaymentEmailDialog';
import { SaoKeHistoryDialog } from '@/components/transaction/SaoKeHistoryDialog';
import { ExportSaoKeDialog } from '@/components/transaction/ExportSaoKeDialog';
import { AdvancePaymentExportDialog } from '@/components/advance-payment/AdvancePaymentExportDialog';
import { OnePayFeeReportDialog } from '@/components/ledger/OnePayFeeReportDialog';
import { SettlementResultUploadDialog } from '@/components/transaction/SettlementResultUploadDialog';
import { SettlementSimulationDialog } from '@/components/ledger/SettlementSimulationDialog';
import { useTransactions } from '@/hooks/transactions/useTransactions';
import { useOverallBalance, useLedgerSummary } from '@/hooks/ledger/useLedgerBalance';
import { useLedgerManagement } from '@/hooks/ledger/useLedgerManagement';
import { useLedgerMetadata } from '@/hooks/ledger/useLedgerMetadata';
import { useSendPayrollReportEmail } from '@/hooks/api/usePayrolls';
import { useSendReconciliationEmail } from '@/hooks/api/useAdvancePaymentReconciliation';
import { useProjects } from '@/hooks/api/useProjects';
import { useGeneralModals } from '@/hooks/useModalNavigation';
import { dateToString } from '@/utils/dateHelpers';
import { transactionToLedgerEntryView } from '@/utils/transactionLedgerAdapter';
import { extractOnePayFeeIssues } from '@/utils/onepayFeeReport';
import type { TransactionFilters as FiltersType } from '@/services/api/transaction.service';
import type { LedgerEntry, AccountMetadata, OnePayFeeImportResponse, OnePayFeeReportIssue } from '@/types/api/financial.types';

// PAYR-20: this list is served by the Transaction API — the same model/API as
// the desktop TransactionsPage at the same URL. Cards render through the
// transactionToLedgerEntryView adapter (transaction ids preserved).
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
    sortBy: 'created_at',
    sortOrder: 'desc',
    fromDate: getInitialDateRange.fromDate,
    toDate: getInitialDateRange.toDate,
  }));
  const [showDoubleEntryModal, setShowDoubleEntryModal] = useState(false);

  // Dialog state
  const [payrollEmailDialogOpen, setPayrollEmailDialogOpen] = useState(false);
  const [advanceEmailDialogOpen, setAdvanceEmailDialogOpen] = useState(false);
  const [saoKeHistoryDialogOpen, setSaoKeHistoryDialogOpen] = useState(false);
  const [exportSaoKeDialogOpen, setExportSaoKeDialogOpen] = useState(false);
  const [exportAdvanceDialogOpen, setExportAdvanceDialogOpen] = useState(false);
  const [showSettlementDialog, setShowSettlementDialog] = useState(false);
  const [simulationDialogOpen, setSimulationDialogOpen] = useState(false);
  const [onePayFeeDialogOpen, setOnePayFeeDialogOpen] = useState(false);
  const [onePayFeeResult, setOnePayFeeResult] = useState<OnePayFeeImportResponse | null>(null);
  const [onePayFeeIssues, setOnePayFeeIssues] = useState<OnePayFeeReportIssue[]>([]);

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

  // Data fetching — Transaction model, shared with desktop
  const { data: transactionsData, isLoading: isLoadingTransactions } = useTransactions(filters);
  const { data: overallBalance, isLoading: isLoadingBalance } = useOverallBalance();
  const { data: ledgerSummary, isLoading: isLoadingSummary } = useLedgerSummary(
    filters.fromDate || getInitialDateRange.fromDate,
    filters.toDate || getInitialDateRange.toDate,
    undefined,
    !!(filters.fromDate && filters.toDate)
  );

  const { data: projectsData } = useProjects();
  const { data: accountMetadata, isLoading: isLoadingAccountMetadata } = useLedgerMetadata();

  // Ledger tools (double-entry, recalculator, settlement run) + OnePay fee import
  const {
    importOnePayFeeReport,
    isImportingOnePayFeeReport,
    recalculateBalances: performRecalculate,
    runWalletSettlement: performRunWalletSettlement,
    isRecalculating,
    isRunningWalletSettlement,
  } = useLedgerManagement();

  // Modal navigation
  const { openTransactionDetails, openAddLedgerEntry } = useGeneralModals();

  // Derived data
  const transactions = transactionsData?.data || [];
  const entries = useMemo(
    () => transactions.map(transactionToLedgerEntryView),
    [transactions],
  );
  const pagination = transactionsData?.pagination ?? null;
  const netCashFlow = ledgerSummary?.totals.net_cashflow;
  const projects = projectsData?.data || [];

  // Handlers
  const handleAddEntry = () => {
    openAddLedgerEntry();
  };

  const handleRecalculateBalance = async () => {
    try {
      await performRecalculate();
    } catch (error) {
      // Error is handled by the hook
    }
  };

  const handleRunWalletSettlement = async () => {
    try {
      await performRunWalletSettlement();
    } catch {
      // Error is handled by the hook
    }
  };

  const handleRowClick = (entry: LedgerEntry) => {
    // Adapter preserves the transaction id — open the shared transaction details.
    openTransactionDetails(entry.id.toString());
  };

  // Filter handlers
  const handleFiltersChange = (newFilters: FiltersType) => {
    setFilters(newFilters);
  };

  const handleClearFilters = () => {
    setFilters({
      page: 1,
      pageSize: 50,
      sortBy: 'created_at',
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

  // OnePay fee import handlers (parity with desktop TransactionsPage)
  const handleOnePayFeeDialogOpenChange = useCallback((open: boolean) => {
    setOnePayFeeDialogOpen(open);
    if (open) {
      setOnePayFeeResult(null);
      setOnePayFeeIssues([]);
    }
  }, []);

  const handleImportOnePayFeeReport = useCallback(async (file: File) => {
    setOnePayFeeResult(null);
    setOnePayFeeIssues([]);
    try {
      const result = await importOnePayFeeReport(file);
      setOnePayFeeResult(result);
    } catch (error) {
      setOnePayFeeIssues(extractOnePayFeeIssues(error));
    }
  }, [importOnePayFeeReport]);

  // Check if any filters are active (excluding default values)
  const hasFilters = !!(filters.search || filters.transaction_type || filters.status || filters.party || filters.created_by);

  return (
    <MobilePageShell className="mx-auto w-full max-w-[1280px] space-y-4 lg:px-6">
      <LedgerPageHeader
        overallBalance={overallBalance}
        netCashFlow={netCashFlow}
        isLoadingBalance={isLoadingBalance}
        isLoadingCashFlow={isLoadingSummary}
        onAddEntry={handleAddEntry}
        onAddDoubleEntry={() => setShowDoubleEntryModal(true)}
        onRecalculateBalance={handleRecalculateBalance}
        isRecalculating={isRecalculating}
        onRunWalletSettlement={handleRunWalletSettlement}
        isRunningWalletSettlement={isRunningWalletSettlement}
        onSendSaoKePayroll={() => setPayrollEmailDialogOpen(true)}
        onSendSaoKeAdvance={() => setAdvanceEmailDialogOpen(true)}
        onViewSaoKeHistory={() => setSaoKeHistoryDialogOpen(true)}
        onImportOnePayFeeReport={() => handleOnePayFeeDialogOpenChange(true)}
        onExportSaoKePayroll={() => setExportSaoKeDialogOpen(true)}
        onExportSaoKeAdvance={() => setExportAdvanceDialogOpen(true)}
        onUploadSettlementResult={() => setShowSettlementDialog(true)}
        onSimulateSettlement={() => setSimulationDialogOpen(true)}
        isSendingSaoKe={sendPayrollEmailMutation.isPending || sendAdvanceEmailMutation.isPending}
      />

      <LedgerSummaryCard
        summary={ledgerSummary}
        isLoading={isLoadingSummary}
      />

      <TransactionFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
        totalResults={pagination?.totalRecords}
      />

      <LedgerMobileList
        entries={entries}
        isLoading={isLoadingTransactions}
        onRowClick={handleRowClick}
        projects={projects}
        accountMetadata={(accountMetadata ?? []) as AccountMetadata[]}
        pagination={pagination}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
      />

      <DoubleEntryModal
        isOpen={showDoubleEntryModal}
        onClose={() => setShowDoubleEntryModal(false)}
        projects={projects}
      />

      {/* Sao kê / export / fee / settlement dialogs — shared with desktop */}
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
      <ExportSaoKeDialog
        open={exportSaoKeDialogOpen}
        onOpenChange={setExportSaoKeDialogOpen}
      />
      <AdvancePaymentExportDialog
        open={exportAdvanceDialogOpen}
        onOpenChange={setExportAdvanceDialogOpen}
      />
      <OnePayFeeReportDialog
        open={onePayFeeDialogOpen}
        onOpenChange={handleOnePayFeeDialogOpenChange}
        onUpload={handleImportOnePayFeeReport}
        onFileChange={() => {
          setOnePayFeeResult(null);
          setOnePayFeeIssues([]);
        }}
        isUploading={isImportingOnePayFeeReport}
        result={onePayFeeResult}
        issues={onePayFeeIssues}
      />
      <SettlementResultUploadDialog
        open={showSettlementDialog}
        onOpenChange={setShowSettlementDialog}
      />
      <SettlementSimulationDialog
        open={simulationDialogOpen}
        onOpenChange={setSimulationDialogOpen}
      />
    </MobilePageShell>
  );
};

export default LedgerEntriesPageMobile;
