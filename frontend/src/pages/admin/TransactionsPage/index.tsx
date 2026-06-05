import { useState, useCallback } from 'react';
import { TransactionPageHeader } from '@/components/transaction/TransactionPageHeader';
import { TransactionSummaryCard } from '@/components/transaction/TransactionSummaryCard';
import { CapitalContributionsCard } from '@/components/transaction/CapitalContributionsCard';
import { TransactionTable } from '@/components/transaction/TransactionTable';
import { TransactionFilters } from '@/components/transaction/TransactionFilters';
import { SettlementResultUploadDialog } from '@/components/transaction/SettlementResultUploadDialog';
import { PayrollReportEmailDialog, type PayrollReportEmailParams } from '@/components/timesheet/PayrollReportEmailDialog';
import { AdvancePaymentEmailDialog, type AdvancePaymentEmailParams } from '@/components/advance-payment/AdvancePaymentEmailDialog';
import { SaoKeHistoryDialog } from '@/components/transaction/SaoKeHistoryDialog';
import { ExportSaoKeDialog } from '@/components/transaction/ExportSaoKeDialog';
import { AdvancePaymentExportDialog } from '@/components/advance-payment/AdvancePaymentExportDialog';
import { useTransactions } from '@/hooks/transactions/useTransactions';
import { useLedgerSummary } from '@/hooks/ledger/useLedgerBalance';
import { useGeneralModals } from '@/hooks/useModalNavigation';
import { useSendPayrollReportEmail } from '@/hooks/api/usePayrolls';
import { useSendReconciliationEmail } from '@/hooks/api/useAdvancePaymentReconciliation';
import { useTableSorting } from '@/utils/sorting';
import type { TransactionFilters as FiltersType, Transaction } from '@/services/api/transaction.service';
import { getInitialDateRange } from './utils';
import { DEFAULT_FILTERS, SORT_FIELD_MAP } from './constants';

const TransactionsPage = () => {
  const initialDateRange = getInitialDateRange();

  const [filters, setFilters] = useState<FiltersType>(() => ({
    page: 1,
    pageSize: 50,
    sortBy: 'created_at',
    sortOrder: 'desc',
    fromDate: initialDateRange.fromDate,
    toDate: initialDateRange.toDate,
  }));

  const [showSettlementDialog, setShowSettlementDialog] = useState(false);
  const [payrollEmailDialogOpen, setPayrollEmailDialogOpen] = useState(false);
  const [advanceEmailDialogOpen, setAdvanceEmailDialogOpen] = useState(false);
  const [saoKeHistoryDialogOpen, setSaoKeHistoryDialogOpen] = useState(false);
  const [exportSaoKeDialogOpen, setExportSaoKeDialogOpen] = useState(false);
  const [exportAdvanceDialogOpen, setExportAdvanceDialogOpen] = useState(false);

  const sendPayrollEmailMutation = useSendPayrollReportEmail();
  const sendAdvanceEmailMutation = useSendReconciliationEmail();

  const { data: transactionsData } = useTransactions(filters);

  const transactions = transactionsData?.data || [];
  const pagination = transactionsData?.pagination || null;
  const totalRecords = pagination?.totalRecords || 0;

  const { data: ledgerSummary, isLoading: isLoadingSummary } = useLedgerSummary(
    filters.fromDate || initialDateRange.fromDate,
    filters.toDate || initialDateRange.toDate
  );

  const { openTransactionDetails, openAddLedgerEntry } = useGeneralModals();

  const handleAddTransaction = () => {
    openAddLedgerEntry();
  };

  const handleUploadSettlementResult = () => {
    setShowSettlementDialog(true);
  };

  const handleRowClick = (transaction: Transaction) => {
    openTransactionDetails(transaction.id.toString());
  };

  const handleFiltersChange = (newFilters: FiltersType) => {
    setFilters(newFilters);
  };

  const handleClearFilters = () => {
    setFilters({
      page: 1,
      pageSize: 50,
      sortBy: 'created_at',
      sortOrder: 'desc',
      fromDate: initialDateRange.fromDate,
      toDate: initialDateRange.toDate,
    });
  };

  const handlePageChange = (page: number) => {
    setFilters(prev => ({ ...prev, page }));
  };

  const handlePageSizeChange = (pageSize: number) => {
    setFilters(prev => ({ ...prev, pageSize, page: 1 }));
  };

  const hasFilters = !!(filters.search || filters.transaction_type || filters.status || filters.party || filters.created_by);

  const { sorting, onSortingChange } = useTableSorting(
    filters.sortBy,
    filters.sortOrder as 'desc' | 'asc',
    useCallback((sortBy, sortOrder) => {
      setFilters(prev => ({ ...prev, sortBy, sortOrder }));
    }, []),
    SORT_FIELD_MAP,
  );

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

  return (
    <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
      <TransactionPageHeader
        onAddTransaction={handleAddTransaction}
        onSendSaoKePayroll={() => setPayrollEmailDialogOpen(true)}
        onSendSaoKeAdvance={() => setAdvanceEmailDialogOpen(true)}
        onViewSaoKeHistory={() => setSaoKeHistoryDialogOpen(true)}
        onExportSaoKePayroll={() => setExportSaoKeDialogOpen(true)}
        onExportSaoKeAdvance={() => setExportAdvanceDialogOpen(true)}
        isSendingSaoKe={sendPayrollEmailMutation.isPending || sendAdvanceEmailMutation.isPending}
      />

      <TransactionSummaryCard
        ledgerSummary={ledgerSummary}
        isLoading={isLoadingSummary}
        renderCapitalCard={() => (
          <CapitalContributionsCard
            contributions={ledgerSummary?.by_owners}
            isLoading={isLoadingSummary}
          />
        )}
      />

      <TransactionFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
        totalResults={totalRecords}
      />

      <TransactionTable
        transactions={transactions}
        onRowClick={handleRowClick}
        pagination={pagination}
        onPageChange={handlePageChange}
        onPageSizeChange={handlePageSizeChange}
        sorting={sorting}
        onSortingChange={onSortingChange}
      />

      <SettlementResultUploadDialog
        open={showSettlementDialog}
        onOpenChange={setShowSettlementDialog}
      />

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
    </div>
  );
};

export default TransactionsPage;
