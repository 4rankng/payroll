import { useState, useMemo, useEffect, useCallback } from 'react';
import { useSearchParams } from 'react-router-dom';
import { LedgerPageHeader } from '@/components/ledger/LedgerPageHeader';
import { LedgerSummaryCard } from '@/components/ledger/LedgerSummaryCard';
import { LedgerEntriesTable } from '@/components/ledger/LedgerEntriesTable';
import { LedgerFilters } from '@/components/ledger/LedgerFilters';
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
import { useUsersByIds } from '@/hooks/api/useUsers';
import { useGeneralModals } from '@/hooks/useModalNavigation';
import { dateToString } from '@/utils/dateHelpers';
import type { LedgerFilters as FiltersType, LedgerEntry, AccountMetadata } from '@/types/api/financial.types';
import { getInitialDateRange, getCurrentMonthRange } from './utils';

const LedgerEntriesPage = () => {
  const [searchParams] = useSearchParams();

  const initialDateRange = useMemo(() => getInitialDateRange(), []);
  const currentMonth = useMemo(() => getCurrentMonthRange(), []);

  // Sao kê dialog state
  const [payrollEmailDialogOpen, setPayrollEmailDialogOpen] = useState(false);
  const [advanceEmailDialogOpen, setAdvanceEmailDialogOpen] = useState(false);
  const [saoKeHistoryDialogOpen, setSaoKeHistoryDialogOpen] = useState(false);

  const sendPayrollEmailMutation = useSendPayrollReportEmail();
  const sendAdvanceEmailMutation = useSendReconciliationEmail();

  const [filters, setFilters] = useState<FiltersType>(() => ({
    page: 1,
    pageSize: 50,
    sortBy: 'date',
    sortOrder: 'desc',
    fromDate: initialDateRange.fromDate,
    toDate: initialDateRange.toDate,
  }));
  const [showDoubleEntryModal, setShowDoubleEntryModal] = useState(false);
  const [reverseEntry, setReverseEntry] = useState<LedgerEntry | null>(null);

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
        page: 1,
      }));
    }
  }, [searchParams]);

  const { data: entriesData, isLoading: isLoadingEntries } = useLedgerEntries(filters);
  const { data: overallBalance, isLoading: isLoadingBalance } = useOverallBalance();
  const { data: cashFlowData, isLoading: isLoadingCashFlow } = useCashFlowSummary(
    currentMonth.fromDate,
    currentMonth.toDate
  );
  const { data: ledgerSummary, isLoading: isLoadingSummary } = useLedgerSummary(
    filters.fromDate || initialDateRange.fromDate,
    filters.toDate || initialDateRange.toDate,
    filters.project_id,
    !!(filters.fromDate && filters.toDate)
  );

  const { data: projectsData } = useProjects();
  const { data: accountMetadata, isLoading: isLoadingAccountMetadata } = useLedgerMetadata();

  const { reverseEntry: performReverse, recalculateBalances: performRecalculate, isReversing, isRecalculating } = useLedgerManagement();

  const { openLedgerDetails, openAddLedgerEntry } = useGeneralModals();

  const entries = useMemo(() => (
    Array.isArray(entriesData) ? entriesData : (entriesData?.data || [])
  ), [entriesData]);
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

  const userIds = useMemo(() => {
    const ids = new Set<number>();
    entries.forEach(entry => {
      if (entry.created_by) ids.add(entry.created_by);
    });
    return Array.from(ids);
  }, [entries]);

  const { data: userMap } = useUsersByIds(userIds);

  const users = useMemo(() => {
    if (!userMap) return [];
    return Object.values(userMap).map(user => ({
      id: user.id,
      full_name: user.fullname,
      email: user.email
    }));
  }, [userMap]);

  const handleAddEntry = () => {
    openAddLedgerEntry();
  };

  const handleAddDoubleEntry = () => {
    setShowDoubleEntryModal(true);
  };

  const handleCloseDoubleEntryModal = () => {
    setShowDoubleEntryModal(false);
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

  const handleFiltersChange = (newFilters: FiltersType) => {
    setFilters(newFilters);
  };

  const handleClearFilters = () => {
    setFilters({
      page: 1,
      pageSize: 50,
      sortBy: 'date',
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

  const hasFilters = !!(filters.account || filters.party || filters.created_by || filters.has_evidence !== undefined);

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

  return (
    <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
      <LedgerPageHeader
        overallBalance={overallBalance}
        netCashFlow={netCashFlow}
        isLoadingBalance={isLoadingBalance}
        isLoadingCashFlow={isLoadingCashFlow}
        onAddEntry={handleAddEntry}
        onAddDoubleEntry={handleAddDoubleEntry}
        onRecalculateBalance={handleRecalculateBalance}
        isRecalculating={isRecalculating}
        onSendSaoKePayroll={() => setPayrollEmailDialogOpen(true)}
        onSendSaoKeAdvance={() => setAdvanceEmailDialogOpen(true)}
        onViewSaoKeHistory={() => setSaoKeHistoryDialogOpen(true)}
        isSendingSaoKe={sendPayrollEmailMutation.isPending || sendAdvanceEmailMutation.isPending}
      />
      <LedgerSummaryCard
        summary={ledgerSummary}
        isLoading={isLoadingSummary}
      />
      <LedgerFilters
        filters={filters}
        onFiltersChange={handleFiltersChange}
        projects={projects}
        users={users}
        onClearFilters={handleClearFilters}
        hasFilters={hasFilters}
        totalResults={pagination?.totalRecords}
        accountMetadata={(accountMetadata ?? []) as AccountMetadata[]}
        isLoadingAccountMetadata={isLoadingAccountMetadata}
      />
      {!isLoadingAccountMetadata && accountMetadata && accountMetadata.length > 0 ? (
        <LedgerEntriesTable
          entries={entries}
          isLoading={isLoadingEntries}
          onReverse={handleReverseEntry}
          onRowClick={handleRowClick}
          projects={projects}
          accountMetadata={accountMetadata as AccountMetadata[]}
          pagination={pagination}
          onPageChange={handlePageChange}
          onPageSizeChange={handlePageSizeChange}
        />
      ) : (
        <div className="flex justify-center items-center py-12">
          <p className="text-sm text-muted-foreground">Đang tải thông tin tài khoản...</p>
        </div>
      )}
      <DoubleEntryModal
        isOpen={showDoubleEntryModal}
        onClose={handleCloseDoubleEntryModal}
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

export default LedgerEntriesPage;
