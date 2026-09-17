import { ErrorState } from "@/components/ui/error-state";
import { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { MobileStatStrip } from "@/components/shared/MobileStatStrip";
import { MobilePagination } from '@/components/shared/MobilePagination';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { useLoans, useLenders } from '@/hooks/api/useLoans';
import { formatVND, daysUntil, getPaymentUrgencyColor } from '@/utils/loanHelpers';
import { Plus, Landmark, ArrowUp, ArrowDown } from 'lucide-react';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import { SearchableSelect } from '@/components/ui/searchable-select';
import type { Lender, LoanStatus, Loan, LoanFilters } from '@/types/api/loan.types';
import { AddLoanSheet } from '@/components/sheets/AddLoanSheet';
import { LoanDetailsSheet } from '@/components/sheets/LoanDetailsSheet';
import { MobileSearchInput } from '@/components/shared/MobileSearchInput';
import { EmptyState } from '@/components/shared/EmptyState';
import { cn } from '@/lib/utils';
import { vietnameseIncludes } from '@/utils/vietnameseNormalization';

const LOAN_STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  active: { label: "Đang vay", className: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  paid: { label: "Đã trả", className: "bg-muted/50 text-muted-foreground border-border" },
  overdue: { label: "Quá hạn", className: "bg-red-50 text-red-700 border-red-200" },
};

const LoanMobileCard = ({ loan, onClick }: { loan: Loan; onClick: (loan: Loan) => void }) => {
  const urgencyColor = loan.next_payment_date
    ? getPaymentUrgencyColor(daysUntil(loan.next_payment_date))
    : 'text-muted-foreground';
  const statusConfig = LOAN_STATUS_CONFIG[loan.status ?? ''] ?? { label: loan.status ?? '', className: 'bg-muted/50 text-muted-foreground border-border' };

  return (
    <div
      onClick={() => onClick(loan)}
      className="bg-card border border-border rounded-xl px-3.5 py-3 shadow-sm card-lift active:bg-muted/50 transition-all cursor-pointer touch-manipulation"
      role="button"
      tabIndex={0}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick(loan); } }}
      aria-label={`Khoản vay ${loan.loan_code}`}
    >
      {/* Line 1: code + status badge */}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <span className="min-w-0 break-all font-mono text-sm font-semibold text-primary">{loan.loan_code}</span>
        {loan.status && (
          <Badge variant="outline" className={cn("min-h-6 shrink-0 border px-1.5 text-xs", statusConfig.className)}>
            {statusConfig.label}
          </Badge>
        )}
      </div>
      {/* Line 2: lender · principal · next payment */}
      <div className="mt-0.5 flex min-w-0 flex-wrap items-center gap-1.5">
        <span className="min-w-0 break-words text-xs text-muted-foreground">{loan.lender.name}</span>
        <span className="text-gray-300 shrink-0">·</span>
        <span className="break-words text-xs font-medium text-foreground">{formatVND(loan.principal_amount)}</span>
        {loan.next_payment_date && (
          <>
            <span className="text-gray-300 shrink-0">·</span>
            <span className={cn("break-words text-xs font-medium", urgencyColor)}>{format(new Date(loan.next_payment_date), 'dd/MM/yyyy')}</span>
          </>
        )}
      </div>
    </div>
  );
};

const LoansPageMobile = () => {
  const navigate = useNavigate();
  const [loanFilters, setLoanFilters] = useState<LoanFilters>({
    page: 1, pageSize: 50, sortBy: 'created_at', sortOrder: 'desc',
  });
  const [loanSearch, setLoanSearch] = useState('');
  const [isAddLoanOpen, setIsAddLoanOpen] = useState(false);
  const [isLoanDetailsOpen, setIsLoanDetailsOpen] = useState(false);
  const [selectedLoanId, setSelectedLoanId] = useState<number | null>(null);

  const { data: loansResponse, isLoading, isError, refetch } = useLoans(loanFilters);
  const { data: lendersResponse } = useLenders({ page: 1, pageSize: 100, sortBy: 'name', sortOrder: 'asc' });

  const lenders: Lender[] = useMemo(() => (
    Array.isArray(lendersResponse?.data) ? (lendersResponse!.data as Lender[]) : []
  ), [lendersResponse]);

  const loans: Loan[] = useMemo(() => (
    Array.isArray(loansResponse?.data) ? (loansResponse!.data as Loan[]) : []
  ), [loansResponse]);

  const loansSummary = useMemo(() => ({
    total_borrowed: loans.reduce((sum, l) => sum + l.principal_amount, 0),
    total_outstanding: loans.reduce((sum, l) => sum + l.outstanding_principal, 0),
    total_interest_paid: loans.reduce((sum, l) => sum + (l.total_interest_paid ?? 0), 0),
    active_loans_count: loans.filter(l => l.status === 'active').length,
  }), [loans]);

  const filteredLoans = useMemo(() => {
    const q = loanSearch.trim();
    if (!q) return loans;
    return loans.filter((l) =>
      vietnameseIncludes(l.loan_code, q) || vietnameseIncludes(l.lender.name, q)
    );
  }, [loans, loanSearch]);

  const handleLoanRowClick = useCallback((loan: Loan) => {
    setSelectedLoanId(loan.id);
    setIsLoanDetailsOpen(true);
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

  if (isLoading && loans.length === 0) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-7 w-28" />
          <div className="flex gap-2"><Skeleton className="h-11 w-20" /><Skeleton className="h-11 w-20" /></div>
        </div>
        <div className="grid grid-cols-2 gap-2">{[...Array(4)].map((_, i) => <Skeleton key={i} className="h-14 rounded-xl" />)}</div>
        <Skeleton className="h-11 w-full rounded-xl" />
        <div className="space-y-2">{[...Array(5)].map((_, i) => <Skeleton key={i} className="h-14 w-full rounded-xl" />)}</div>
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-col pb-[calc(5rem+env(safe-area-inset-bottom))]">
      {/* Header */}
      <MobilePageHeader
        title="Khoản vay"
        icon={Landmark}
        actions={
          <div className="flex items-center gap-2 shrink-0">
            <Button variant="outline" size="sm" className="min-h-11" onClick={() => navigate('lenders')}>
              Chủ nợ
            </Button>
            <Button className="btn-admin-primary h-11 w-11 p-0 min-[380px]:w-auto min-[380px]:px-3" aria-label="Tạo khoản vay" onClick={() => setIsAddLoanOpen(true)}>
              <Plus className="h-4 w-4 min-[380px]:mr-1" />
              <span className="hidden min-[380px]:inline">Tạo vay</span>
            </Button>
          </div>
        }
      />

      {/* Stats strip — 2 columns so full VND totals keep their digits. */}
      {!isError && (
        <div className="px-4 pb-3">
          <MobileStatStrip
            columns={2}
            items={[
              { key: "borrowed", label: "Tổng vay", value: formatVND(loansSummary.total_borrowed) },
              { key: "outstanding", label: "Dư nợ", value: formatVND(loansSummary.total_outstanding) },
              { key: "interest", label: "Lãi đã trả", value: formatVND(loansSummary.total_interest_paid) },
              { key: "count", label: "Khoản vay", value: String(loansSummary.active_loans_count) },
            ]}
          />
        </div>
      )}

      {/* Search + filters */}
      <div className="px-4 pb-3 space-y-2">
        <div className="relative">
          <MobileSearchInput
            value={loanSearch}
            onSearch={setLoanSearch}
            placeholder="Tìm mã vay, chủ nợ..."
          />
        </div>
        <div className="flex gap-2">
          <Select onValueChange={handleStatusChange} defaultValue="all">
            <SelectTrigger aria-label="Lọc trạng thái khoản vay" className="min-h-11 flex-1"><SelectValue placeholder="Trạng thái" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tất cả</SelectItem>
              <SelectItem value="active">Đang vay</SelectItem>
              <SelectItem value="paid">Đã trả</SelectItem>
              <SelectItem value="overdue">Quá hạn</SelectItem>
            </SelectContent>
          </Select>
          <SearchableSelect
            value={loanFilters.lender_id ? String(loanFilters.lender_id) : 'all'}
            onChange={handleLenderChange}
            placeholder="Chủ nợ"
            triggerAriaLabel="Lọc chủ nợ"
            searchPlaceholder="Tìm chủ nợ..."
            triggerClassName="min-h-11 flex-1"
            options={[
              { value: 'all', label: 'Tất cả' },
              ...lenders.map((lender) => ({
                value: lender.id.toString(),
                label: lender.name,
              })),
            ]}
          />
        </div>
        {/* Sort row — same field + square-icon-button rhythm as the other rows. */}
        <div className="flex items-center gap-2">
          <Select
            value={loanFilters.sortBy ?? 'created_at'}
            onValueChange={(v) => setLoanFilters((prev) => ({ ...prev, sortBy: v, page: 1 }))}
          >
            <SelectTrigger aria-label="Sắp xếp khoản vay" className="min-h-11 flex-1 text-sm">
              <SelectValue placeholder="Sắp xếp" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="created_at">Ngày tạo</SelectItem>
              <SelectItem value="loan_code">Mã vay</SelectItem>
              <SelectItem value="lender">Chủ nợ</SelectItem>
              <SelectItem value="principal_amount">Tiền gốc</SelectItem>
              <SelectItem value="interest_rate_bps">Lãi suất</SelectItem>
              <SelectItem value="next_payment_date">Ngày đến hạn</SelectItem>
              <SelectItem value="disbursement_date">Ngày giải ngân</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant="outline"
            size="icon"
            className="h-11 w-11 shrink-0 rounded-xl border-border bg-card"
            onClick={() =>
              setLoanFilters((prev) => ({
                ...prev,
                sortOrder: prev.sortOrder === 'asc' ? 'desc' : 'asc',
              }))
            }
            aria-label={
              loanFilters.sortOrder === 'asc'
                ? 'Sắp xếp tăng dần'
                : 'Sắp xếp giảm dần'
            }
          >
            {loanFilters.sortOrder === 'asc' ? (
              <ArrowUp className="h-4 w-4" />
            ) : (
              <ArrowDown className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>

      {/* Loan list */}
      <div className="flex-1 px-4">
        {isError ? (
          <div role="alert">
            <ErrorState message="Không thể tải danh sách khoản vay. Vui lòng thử lại." onRetry={() => void refetch()} />
          </div>
        ) : filteredLoans.length === 0 ? (
          <EmptyState
            title="Chưa có khoản vay nào"
            description="Tạo khoản vay đầu tiên để bắt đầu quản lý."
            size="sm"
          />
        ) : (
          <>
            <div className="space-y-2">
              {filteredLoans.map((loan) => (
                <LoanMobileCard key={loan.id} loan={loan} onClick={handleLoanRowClick} />
              ))}
            </div>
            {/* Pagination — restores desktop paging capability */}
            {loansResponse?.pagination && (
              <MobilePagination
                pagination={loansResponse.pagination}
                onPageChange={(page) => setLoanFilters((prev) => ({ ...prev, page }))}
                className="pt-4"
              />
            )}
          </>
        )}
      </div>

      <AddLoanSheet isOpen={isAddLoanOpen} onClose={() => setIsAddLoanOpen(false)} />
      <LoanDetailsSheet isOpen={isLoanDetailsOpen} onClose={() => setIsLoanDetailsOpen(false)} loanId={selectedLoanId} />
    </div>
  );
};

export default LoansPageMobile;
