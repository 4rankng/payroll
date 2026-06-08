import { useState, useMemo, useCallback } from "react";
import { TransactionPageHeaderMobile } from "@/components/transaction/mobile/TransactionPageHeaderMobile";
import { TransactionSummaryCardMobile } from "@/components/transaction/mobile/TransactionSummaryCardMobile";
import { CapitalContributionsCardMobile } from "@/components/transaction/mobile/CapitalContributionsCardMobile";
import { TransactionMobileList } from "@/components/transaction/mobile/TransactionMobileList";
import { TransactionFiltersMobile } from "@/components/transaction/mobile/TransactionFiltersMobile";
import { BulkTransferResultUploadDialog } from "@/components/timesheet/BulkTransferResultUploadDialog";
import { BulkTransferHistoryDialog } from "@/components/transaction/BulkTransferHistoryDialog";
import { BulkTransferHistoryDetailDialog } from "@/components/transaction/BulkTransferHistoryDetailDialog";
import { SaoKeHistoryDialog } from "@/components/transaction/SaoKeHistoryDialog";
import { PayrollReportEmailDialog } from "@/components/timesheet/PayrollReportEmailDialog";
import { useTransactions } from "@/hooks/transactions/useTransactions";
import { useLedgerSummary } from "@/hooks/ledger/useLedgerBalance";
import { useGeneralModals } from "@/hooks/useModalNavigation";
import { useSendPayrollReportEmail } from "@/hooks/transactions/useSendPayrollReportEmail";
import { dateToString } from "@/utils/dateHelpers";
import type { TransactionFilters as FiltersType, Transaction } from "@/services/api/transaction.service";
import type { PayrollReportEmailParams } from "@/components/timesheet/PayrollReportEmailDialog";

const TransactionsPageMobile = () => {
  const getInitialDateRange = useMemo(() => {
    const now = new Date();
    const oneYearAgo = new Date(now.getFullYear() - 1, now.getMonth(), 1);
    const endOfCurrentMonth = new Date(
      now.getFullYear(),
      now.getMonth() + 1,
      0,
    );
    return {
      fromDate: dateToString(oneYearAgo),
      toDate: dateToString(endOfCurrentMonth),
    };
  }, []);

  const [filters, setFilters] = useState<FiltersType>(() => ({
    page: 1,
    pageSize: 50,
    sortBy: "created_at",
    sortOrder: "desc",
    fromDate: getInitialDateRange.fromDate,
    toDate: getInitialDateRange.toDate,
  }));

  const [showImportDialog, setShowImportDialog] = useState(false);
  const [showSaoKeHistoryDialog, setShowSaoKeHistoryDialog] = useState(false);
  const [showHistoryDialog, setShowHistoryDialog] = useState(false);
  const [selectedHistoryId, setSelectedHistoryId] = useState<number | null>(null);
  const [selectedHistoryFilename, setSelectedHistoryFilename] = useState("");
  const [shouldLoadBulkTransferHistory, setShouldLoadBulkTransferHistory] = useState(false);
  const [payrollEmailDialogOpen, setPayrollEmailDialogOpen] = useState(false);

  const { data: transactionsData, isLoading: isLoadingTransactions } =
    useTransactions(filters);
  const transactions = transactionsData?.data || [];
  const pagination = transactionsData?.pagination || null;
  const totalRecords = pagination?.totalRecords || 0;

  const { data: ledgerSummary, isLoading: isLoadingSummary } = useLedgerSummary(
    filters.fromDate || getInitialDateRange.fromDate,
    filters.toDate || getInitialDateRange.toDate,
  );

  const { openTransactionDetails, openAddLedgerEntry } = useGeneralModals();
  const sendEmailMutation = useSendPayrollReportEmail();

  const handleViewHistory = useCallback(() => {
    setShowHistoryDialog(true);
    setSelectedHistoryId(null);
    setShouldLoadBulkTransferHistory(true);
  }, []);

  const handleCloseHistory = useCallback(() => {
    setShowHistoryDialog(false);
    setSelectedHistoryId(null);
    setSelectedHistoryFilename("");
    setShouldLoadBulkTransferHistory(false);
  }, []);

  const handlePayrollReportEmailSend = useCallback(
    async (params: PayrollReportEmailParams) => {
      try {
        await sendEmailMutation.mutateAsync(params);
        setPayrollEmailDialogOpen(false);
      } catch { /* handled by mutation */ }
    },
    [sendEmailMutation],
  );

  const hasFilters = !!(
    filters.search ||
    filters.transaction_type ||
    filters.status ||
    filters.party ||
    filters.created_by
  );

  return (
    <div className="p-4 pb-20 space-y-4 max-w-full overflow-hidden">
      <TransactionPageHeaderMobile
        onAddTransaction={() => openAddLedgerEntry()}
        onImportTransactions={() => setShowImportDialog(true)}
        onViewHistory={handleViewHistory}
        onSendStatementEmail={() => setPayrollEmailDialogOpen(true)}
        onViewSaoKeHistory={() => setShowSaoKeHistoryDialog(true)}
        isSendingStatementEmail={sendEmailMutation.isPending}
      />

      <TransactionSummaryCardMobile
        ledgerSummary={ledgerSummary}
        isLoading={isLoadingSummary}
      />
      <CapitalContributionsCardMobile
        contributions={ledgerSummary?.by_owners}
        isLoading={isLoadingSummary}
      />

      <TransactionFiltersMobile
        filters={filters}
        onFiltersChange={setFilters}
        onClearFilters={() =>
          setFilters({
            page: 1,
            pageSize: 50,
            sortBy: "created_at",
            sortOrder: "desc",
            fromDate: getInitialDateRange.fromDate,
            toDate: getInitialDateRange.toDate,
          })
        }
        hasFilters={hasFilters}
        totalResults={totalRecords}
      />

      <TransactionMobileList
        transactions={transactions}
        isLoading={isLoadingTransactions}
        onRowClick={(tx: Transaction) =>
          openTransactionDetails(tx.id.toString())
        }
        pagination={pagination}
      />

      <BulkTransferResultUploadDialog
        open={showImportDialog}
        onOpenChange={setShowImportDialog}
      />
      <SaoKeHistoryDialog
        open={showSaoKeHistoryDialog}
        onOpenChange={setShowSaoKeHistoryDialog}
      />

      <BulkTransferHistoryDialog
        open={showHistoryDialog && selectedHistoryId === null}
        onOpenChange={handleCloseHistory}
        onSelectHistory={(id, filename) => {
          setSelectedHistoryId(id);
          setSelectedHistoryFilename(filename);
        }}
        shouldFetchHistories={shouldLoadBulkTransferHistory}
      />
      <BulkTransferHistoryDetailDialog
        open={showHistoryDialog && selectedHistoryId !== null}
        historyId={selectedHistoryId}
        filename={selectedHistoryFilename}
        onOpenChange={handleCloseHistory}
        onBack={() => {
          setSelectedHistoryId(null);
          setSelectedHistoryFilename("");
        }}
      />

      <PayrollReportEmailDialog
        open={payrollEmailDialogOpen}
        onOpenChange={setPayrollEmailDialogOpen}
        onSendEmail={handlePayrollReportEmailSend}
        isLoading={sendEmailMutation.isPending}
      />
    </div>
  );
};

export default TransactionsPageMobile;
