import { ErrorState } from "@/components/ui/error-state";
import { useState, useMemo, useCallback } from 'react';
import { ResponsiveTable } from '@/components/ui/responsive-table';
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
  AdminFilterRow,
} from '@/components/shared/AdminPageFrame';
import { PageHeader } from '@/components/shared/PageHeader';
import { KpiHeroCard } from '@/components/admin-dashboard/KpiHeroCard';
import { SearchBar } from '@/components/shared/SearchBar';
import { FilterPill } from '@/components/shared/FilterPill';
import { useLoans, useLenders } from '@/hooks/api/useLoans';
import { useTableSorting } from '@/utils/sorting';
import { Plus, Users2, Landmark, TrendingUp, Coins, Wallet } from 'lucide-react';
import type { Lender, LoanStatus, Loan, LoanFilters } from '@/types/api/loan.types';
import { LenderManagementSheet } from '@/components/sheets/LenderManagementSheet';
import { AddLoanSheet } from '@/components/sheets/AddLoanSheet';
import { LoanDetailsSheet } from '@/components/sheets/LoanDetailsSheet';
import {
  loanColumns,
  loanMobileFields,
  loansEmptyState,
  loanSortFieldMap,
} from '@/components/loans/loan-table-config';
import { computeLoansSummary, computeLoanStatItems, filterLoansBySearch } from './utils';

const LoansPage = () => {
  const [loanFilters, setLoanFilters] = useState<LoanFilters>({
    page: 1,
    pageSize: 20,
    sortBy: 'created_at',
    sortOrder: 'desc',
  });

  const [loanSearch, setLoanSearch] = useState('');
  const [isLenderSheetOpen, setIsLenderSheetOpen] = useState(false);
  const [isAddLoanOpen, setIsAddLoanOpen] = useState(false);
  const [isLoanDetailsOpen, setIsLoanDetailsOpen] = useState(false);
  const [selectedLoanId, setSelectedLoanId] = useState<number | null>(null);

  const { data: loansResponse, isLoading: isLoadingLoans, isError, refetch } = useLoans(loanFilters);
  const { data: lendersResponse } = useLenders({
    page: 1,
    pageSize: 100,
    sortBy: 'name',
    sortOrder: 'asc',
  });

  const lenders: Lender[] = useMemo(
    () => (Array.isArray(lendersResponse?.data) ? (lendersResponse!.data as Lender[]) : []),
    [lendersResponse],
  );

  const loans: Loan[] = useMemo(
    () => (Array.isArray(loansResponse?.data) ? (loansResponse!.data as Loan[]) : []),
    [loansResponse],
  );
  const loansPagination = loansResponse?.pagination;

  const loansSummary = useMemo(() => computeLoansSummary(loans), [loans]);
  const loanStatItems = useMemo(() => computeLoanStatItems(loansSummary), [loansSummary]);
  const filteredLoans = useMemo(() => filterLoansBySearch(loans, loanSearch), [loans, loanSearch]);

  const handleOpenLenderSheet = useCallback(() => setIsLenderSheetOpen(true), []);
  const handleCloseLenderSheet = useCallback(() => setIsLenderSheetOpen(false), []);
  const handleOpenAddLoan = useCallback(() => setIsAddLoanOpen(true), []);
  const handleCloseAddLoan = useCallback(() => setIsAddLoanOpen(false), []);

  const handleLoanPageChange = useCallback((page: number) => {
    setLoanFilters((prev) => ({ ...prev, page }));
  }, []);

  const handleLoanPageSizeChange = useCallback((pageSize: number) => {
    setLoanFilters((prev) => ({ ...prev, pageSize, page: 1 }));
  }, []);

  const handleLoanRowClick = useCallback((loan: Loan) => {
    setSelectedLoanId(loan.id);
    setIsLoanDetailsOpen(true);
  }, []);

  const handleCloseLoanDetails = useCallback(() => {
    setIsLoanDetailsOpen(false);
  }, []);

  const handleStatusChange = useCallback((value: string) => {
    setLoanFilters((prev) => ({
      ...prev,
      status: value === 'all' ? undefined : (value as LoanStatus),
      page: 1,
    }));
  }, []);

  const handleLenderChange = useCallback((value: string) => {
    setLoanFilters((prev) => ({
      ...prev,
      lender_id: value === 'all' ? undefined : Number(value),
      page: 1,
    }));
  }, []);

  const getLoanRowId = useCallback((row: Loan) => row.id.toString(), []);

  const { sorting, onSortingChange } = useTableSorting(
    loanFilters.sortBy,
    loanFilters.sortOrder,
    useCallback((sortBy, sortOrder) => {
      setLoanFilters((prev) => ({ ...prev, sortBy, sortOrder }));
    }, []),
    loanSortFieldMap,
  );

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader
          icon={Landmark}
          title="Quản lý khoản vay"
          description="Quản lý chủ nợ và các khoản vay của công ty"
          actions={[
            {
              label: 'QL Chủ nợ',
              onClick: handleOpenLenderSheet,
              icon: Users2,
              variant: 'outline' as const,
            },
            {
              label: 'Tạo khoản vay',
              onClick: handleOpenAddLoan,
              icon: Plus,
              variant: 'default' as const,
            },
          ]}
        />
      </AdminPageHeaderCard>

      {!isError && <div className="grid grid-cols-1 min-[380px]:grid-cols-2 md:grid-cols-4 gap-3">
        <KpiHeroCard
          label="Tổng vay"
          value={loansSummary.total_borrowed}
          unit="₫"
          icon={Landmark}
          color="blue"
          variant="stack"
        />
        <KpiHeroCard
          label="Dư nợ hiện tại"
          value={loansSummary.total_outstanding}
          unit="₫"
          icon={TrendingUp}
          color="amber"
          variant="stack"
        />
        <KpiHeroCard
          label="Lãi đã trả"
          value={loansSummary.total_interest_paid}
          unit="₫"
          icon={Coins}
          color="emerald"
          variant="stack"
        />
        <KpiHeroCard
          label="Khoản vay"
          value={loansSummary.active_loans_count}
          icon={Wallet}
          color="teal"
          variant="stack"
        />
      </div>}

      <AdminSectionCard aria-label="Danh sách khoản vay">
        <AdminFilterRow>
          <p className="text-sm font-semibold text-slate-800">Danh sách</p>
          <div className="flex flex-1 flex-wrap items-center justify-end gap-2">
            <SearchBar
              searchTerm={loanSearch}
              onSearchChange={setLoanSearch}
              placeholder="Tìm mã vay, chủ nợ..."
              className="w-52"
            />
            <FilterPill
              value={loanFilters.status ?? 'all'}
              onChange={handleStatusChange}
              placeholder="Trạng thái"
              options={[
                { value: 'active', label: 'Đang vay' },
                { value: 'paid', label: 'Đã trả' },
                { value: 'overdue', label: 'Quá hạn' },
              ]}
            />
            <FilterPill
              value={loanFilters.lender_id?.toString() ?? 'all'}
              onChange={handleLenderChange}
              placeholder="Chủ nợ"
              options={lenders.map((l) => ({ value: l.id.toString(), label: l.name }))}
            />
          </div>
        </AdminFilterRow>

        {isError ? (
          <div role="alert">
            <ErrorState message="Không thể tải danh sách khoản vay. Vui lòng thử lại." onRetry={() => void refetch()} className="px-4" />
          </div>
        ) : (
        <ResponsiveTable
          data={filteredLoans}
          columns={loanColumns}
          mobileFields={loanMobileFields}
          getRowId={getLoanRowId}
          onRowClick={handleLoanRowClick}
          pagination={loansPagination}
          onPageChange={handleLoanPageChange}
          onPageSizeChange={handleLoanPageSizeChange}
          emptyState={loansEmptyState}
          accordionType="single"
          sorting={sorting}
          onSortingChange={onSortingChange}
          embedded
        />
        )}
      </AdminSectionCard>

      <LenderManagementSheet isOpen={isLenderSheetOpen} onClose={handleCloseLenderSheet} />
      <AddLoanSheet isOpen={isAddLoanOpen} onClose={handleCloseAddLoan} />
      <LoanDetailsSheet
        isOpen={isLoanDetailsOpen}
        onClose={handleCloseLoanDetails}
        loanId={selectedLoanId}
      />
    </AdminPageCanvas>
  );
};

export default LoansPage;
