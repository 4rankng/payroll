import { useState, useMemo, useCallback } from "react";
import { TransactionPageHeaderMobile } from "@/components/transaction/mobile/TransactionPageHeaderMobile";
import { TransactionSummaryCardMobile } from "@/components/transaction/mobile/TransactionSummaryCardMobile";
import { CapitalContributionsCardMobile } from "@/components/transaction/mobile/CapitalContributionsCardMobile";
import { TransactionMobileList } from "@/components/transaction/mobile/TransactionMobileList";
import { TransactionFiltersMobile } from "@/components/transaction/mobile/TransactionFiltersMobile";
import { BulkTransferResultUploadDialog } from "@/components/timesheet/BulkTransferResultUploadDialog";
import { SettlementResultUploadDialog } from "@/components/transaction/SettlementResultUploadDialog";
import { OnePayFeeReportDialog } from "@/components/ledger/OnePayFeeReportDialog";
import { SettlementSimulationDialog } from "@/components/ledger/SettlementSimulationDialog";
import { ExportSaoKeDialog } from "@/components/transaction/ExportSaoKeDialog";
import { AdvancePaymentExportDialog } from "@/components/advance-payment/AdvancePaymentExportDialog";
import { BulkTransferHistoryDialog } from "@/components/transaction/BulkTransferHistoryDialog";
import { BulkTransferHistoryDetailDialog } from "@/components/transaction/BulkTransferHistoryDetailDialog";
import { SaoKeHistoryDialog } from "@/components/transaction/SaoKeHistoryDialog";
import {
  PayrollReportEmailDialog,
  type PayrollReportEmailParams,
} from "@/components/timesheet/PayrollReportEmailDialog";
import {
  AdvancePaymentEmailDialog,
  type AdvancePaymentEmailParams,
} from "@/components/advance-payment/AdvancePaymentEmailDialog";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { ArrowDownUp } from "lucide-react";
import { useTransactions } from "@/hooks/transactions/useTransactions";
import { useLedgerSummary } from "@/hooks/ledger/useLedgerBalance";
import { useLedgerManagement } from "@/hooks/ledger/useLedgerManagement";
import { useGeneralModals } from "@/hooks/useModalNavigation";
import { useSendPayrollReportEmail } from "@/hooks/api/usePayrolls";
import { useSendReconciliationEmail } from "@/hooks/api/useAdvancePaymentReconciliation";
import { dateToString } from "@/utils/dateHelpers";
import { extractOnePayFeeIssues } from "@/utils/onepayFeeReport";
import type { TransactionFilters as FiltersType, Transaction } from "@/services/api/transaction.service";
import type { OnePayFeeImportResponse, OnePayFeeReportIssue } from "@/types/api/financial.types";

const SORT_FIELD_MAP: Record<string, string> = {
  created_at: "created_at",
  description: "description",
  transaction_type: "transaction_type",
  party: "party",
  amount: "amount",
  status: "status",
};

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
  const [showSettlementDialog, setShowSettlementDialog] = useState(false);
  const [showSaoKeHistoryDialog, setShowSaoKeHistoryDialog] = useState(false);
  const [showHistoryDialog, setShowHistoryDialog] = useState(false);
  const [selectedHistoryId, setSelectedHistoryId] = useState<number | null>(null);
  const [selectedHistoryFilename, setSelectedHistoryFilename] = useState("");
  const [selectedHistoryUploadedAt, setSelectedHistoryUploadedAt] = useState<string | null>(null);
  const [shouldLoadBulkTransferHistory, setShouldLoadBulkTransferHistory] = useState(false);
  const [payrollEmailDialogOpen, setPayrollEmailDialogOpen] = useState(false);
  const [advanceEmailDialogOpen, setAdvanceEmailDialogOpen] = useState(false);
  const [exportSaoKeDialogOpen, setExportSaoKeDialogOpen] = useState(false);
  const [exportAdvanceDialogOpen, setExportAdvanceDialogOpen] = useState(false);
  const [onePayFeeDialogOpen, setOnePayFeeDialogOpen] = useState(false);
  const [onePayFeeResult, setOnePayFeeResult] = useState<OnePayFeeImportResponse | null>(null);
  const [onePayFeeIssues, setOnePayFeeIssues] = useState<OnePayFeeReportIssue[]>([]);
  const [simulationDialogOpen, setSimulationDialogOpen] = useState(false);

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
  const sendPayrollEmailMutation = useSendPayrollReportEmail();
  const sendAdvanceEmailMutation = useSendReconciliationEmail();
  const {
    importOnePayFeeReport,
    isImportingOnePayFeeReport,
    runWalletSettlement,
    isRunningWalletSettlement,
  } = useLedgerManagement();

  const handleViewHistory = useCallback(() => {
    setShowHistoryDialog(true);
    setSelectedHistoryId(null);
    setShouldLoadBulkTransferHistory(true);
  }, []);

  const handleCloseHistory = useCallback(() => {
    setShowHistoryDialog(false);
    setSelectedHistoryId(null);
    setSelectedHistoryFilename("");
    setSelectedHistoryUploadedAt(null);
    setShouldLoadBulkTransferHistory(false);
  }, []);

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

  const handleRunWalletSettlement = useCallback(async () => {
    try {
      await runWalletSettlement();
    } catch {
      /* handled by mutation */
    }
  }, [runWalletSettlement]);

  const handlePayrollReportEmailSend = useCallback(
    async (params: PayrollReportEmailParams) => {
      try {
        await sendPayrollEmailMutation.mutateAsync(params);
        setPayrollEmailDialogOpen(false);
      } catch {
        /* handled by mutation */
      }
    },
    [sendPayrollEmailMutation],
  );

  const handleAdvanceEmailSend = useCallback(
    async (params: AdvancePaymentEmailParams) => {
      try {
        await sendAdvanceEmailMutation.mutateAsync(params);
        setAdvanceEmailDialogOpen(false);
      } catch {
        /* handled by mutation */
      }
    },
    [sendAdvanceEmailMutation],
  );

  const hasFilters = !!(
    filters.search ||
    filters.transaction_type ||
    filters.status ||
    filters.party ||
    filters.created_by
  );

  return (
    <div className="ct-card-body space-y-4 overflow-x-hidden bg-base-100 p-4 pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <TransactionPageHeaderMobile
        onAddTransaction={() => openAddLedgerEntry()}
        onImportTransactions={() => setShowImportDialog(true)}
        onViewHistory={handleViewHistory}
        onUploadSettlement={() => setShowSettlementDialog(true)}
        onImportOnePayFee={() => handleOnePayFeeDialogOpenChange(true)}
        onExportSaoKePayroll={() => setExportSaoKeDialogOpen(true)}
        onExportSaoKeAdvance={() => setExportAdvanceDialogOpen(true)}
        onSendStatementEmail={() => setPayrollEmailDialogOpen(true)}
        onSendAdvanceEmail={() => setAdvanceEmailDialogOpen(true)}
        onViewSaoKeHistory={() => setShowSaoKeHistoryDialog(true)}
        isSendingStatementEmail={sendPayrollEmailMutation.isPending}
        isSendingAdvanceEmail={sendAdvanceEmailMutation.isPending}
        isImportingOnePayFee={isImportingOnePayFeeReport}
        onRunWalletSettlement={handleRunWalletSettlement}
        isRunningWalletSettlement={isRunningWalletSettlement}
        onSimulateSettlement={() => setSimulationDialogOpen(true)}
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

      {/* Sort + page-size — restores desktop sorting & paging capability */}
      <div className="ct-card flex items-center gap-2 rounded-2xl border border-base-300/70 bg-base-100/80 p-3 shadow-sm">
        <ArrowDownUp className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <Select
          value={(filters.sortBy as string) ?? "created_at"}
          onValueChange={(v) => setFilters((prev) => ({ ...prev, sortBy: v, page: 1 }))}
        >
          <SelectTrigger className="min-h-11 flex-1 border-base-300 bg-base-100 text-xs">
            <SelectValue placeholder="Sắp xếp" />
          </SelectTrigger>
          <SelectContent>
            {Object.keys(SORT_FIELD_MAP).map((key) => (
              <SelectItem key={key} value={key}>
                {{
                  created_at: "Ngày tạo",
                  description: "Mô tả",
                  transaction_type: "Loại GD",
                  party: "Đối tác",
                  amount: "Số tiền",
                  status: "Trạng thái",
                }[key] ?? key}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <button
          type="button"
          onClick={() =>
            setFilters((prev) => ({
              ...prev,
              sortOrder: prev.sortOrder === "asc" ? "desc" : "asc",
              page: 1,
            }))
          }
          className="ct-btn ct-btn-outline ct-btn-sm min-h-11 h-11 border-base-300 bg-base-100 px-3 text-xs font-semibold normal-case text-base-content shadow-none active:scale-95"
          aria-label="Đảo chiều sắp xếp"
        >
          {filters.sortOrder === "asc" ? "Tăng" : "Giảm"}
        </button>
        <Select
          value={String(filters.pageSize ?? 50)}
          onValueChange={(v) =>
            setFilters((prev) => ({ ...prev, pageSize: Number(v), page: 1 }))
          }
        >
          <SelectTrigger className="min-h-11 w-[4.5rem] border-base-300 bg-base-100 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="20">20</SelectItem>
            <SelectItem value="50">50</SelectItem>
            <SelectItem value="100">100</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <TransactionMobileList
        transactions={transactions}
        isLoading={isLoadingTransactions}
        onRowClick={(tx: Transaction) =>
          openTransactionDetails(tx.id.toString())
        }
        pagination={pagination}
        onPageChange={(page) => setFilters((prev) => ({ ...prev, page }))}
      />

      <BulkTransferResultUploadDialog
        open={showImportDialog}
        onOpenChange={setShowImportDialog}
      />
      <SettlementResultUploadDialog
        open={showSettlementDialog}
        onOpenChange={setShowSettlementDialog}
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
      <ExportSaoKeDialog
        open={exportSaoKeDialogOpen}
        onOpenChange={setExportSaoKeDialogOpen}
      />
      <AdvancePaymentExportDialog
        open={exportAdvanceDialogOpen}
        onOpenChange={setExportAdvanceDialogOpen}
      />
      <SaoKeHistoryDialog
        open={showSaoKeHistoryDialog}
        onOpenChange={setShowSaoKeHistoryDialog}
      />

      <BulkTransferHistoryDialog
        open={showHistoryDialog && selectedHistoryId === null}
        onOpenChange={handleCloseHistory}
        onSelectHistory={(id, filename, uploadedAt) => {
          setSelectedHistoryId(id);
          setSelectedHistoryFilename(filename);
          setSelectedHistoryUploadedAt(uploadedAt);
        }}
        shouldFetchHistories={shouldLoadBulkTransferHistory}
      />
      <BulkTransferHistoryDetailDialog
        open={showHistoryDialog && selectedHistoryId !== null}
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

      <PayrollReportEmailDialog
        open={payrollEmailDialogOpen}
        onOpenChange={setPayrollEmailDialogOpen}
        onSendEmail={handlePayrollReportEmailSend}
        isLoading={sendPayrollEmailMutation.isPending}
      />
      <AdvancePaymentEmailDialog
        open={advanceEmailDialogOpen}
        onOpenChange={setAdvanceEmailDialogOpen}
        onSendEmail={handleAdvanceEmailSend}
        isLoading={sendAdvanceEmailMutation.isPending}
      />
      <SettlementSimulationDialog
        open={simulationDialogOpen}
        onOpenChange={setSimulationDialogOpen}
      />
    </div>
  );
};

export default TransactionsPageMobile;
