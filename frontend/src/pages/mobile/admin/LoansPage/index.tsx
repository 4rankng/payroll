import { useState, useMemo, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { MobilePageHeader } from '@/components/shared/MobilePageHeader';
import { MobilePagination } from '@/components/shared/MobilePagination';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { useLoans, useLenders } from '@/hooks/api/useLoans';
import { formatVND, daysUntil, getPaymentUrgencyColor } from '@/utils/loanHelpers';
import { Plus, Landmark, ArrowDownUp } from 'lucide-react';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select';
import type { Lender, LoanStatus, Loan, LoanFilters } from '@/types/api/loan.types';
import { AddLoanSheet } from '@/components/sheets/AddLoanSheet';
import { LoanDetailsSheet } from '@/components/sheets/LoanDetailsSheet';
import { MobileSearchInput } from '@/components/shared/MobileSearchInput';
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
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') onClick(loan); }}
      aria-label={`Khoản vay ${loan.loan_code}`}
    >
      {/* Line 1: code + status badge */}
      <div className="flex flex-wrap items-start justify-between gap-2">
        <span className="min-w-0 break-all font-mono text-sm font-semibold text-primary">{loan.loan_code}</span>
        {loan.status && (
          <Badge variant="outline" className={cn("min-h-6 shrink-0 border px-1.5 text-[11px]", statusConfig.className)}>
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
            <span className={cn("break-words text-xs font-medium", urgencyColor)}>{loan.next_payment_date}</span>
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

  const { data: loansResponse, isLoading } = useLoans(loanFilters);
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
        <div className="flex gap-2">{[...Array(4)].map((_, i) => <Skeleton key={i} className="h-14 w-20 shrink-0 rounded-xl" />)}</div>
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
            <Button className="btn-admin-primary min-h-11" onClick={() => setIsAddLoanOpen(true)}>
              <Plus className="h-4 w-4 mr-1" />
              Tạo vay
            </Button>
          </div>
        }
      />

      {/* Stats strip */}
      <div className="px-4 pb-3">
        <div className="flex gap-2 overflow-x-auto scrollbar-none">
          {[
            { label: "Tổng vay", value: formatVND(loansSummary.total_borrowed) },
            { label: "Dư nợ", value: formatVND(loansSummary.total_outstanding) },
            { label: "Lãi đã trả", value: formatVND(loansSummary.total_interest_paid) },
            { label: "Khoản vay", value: String(loansSummary.active_loans_count) },
          ].map((stat) => (
            <div key={stat.label} className="flex flex-col items-center gap-1 px-3 py-2.5 rounded-xl border border-border bg-card shrink-0 min-w-[80px]">
              <span className="text-sm font-bold tabular-nums leading-none text-foreground">{stat.value}</span>
              <span className="text-[11px] leading-none text-muted-foreground">{stat.label}</span>
            </div>
          ))}
        </div>
      </div>

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
            <SelectTrigger className="min-h-11 flex-1"><SelectValue placeholder="Trạng thái" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tất cả</SelectItem>
              <SelectItem value="active">Đang vay</SelectItem>
              <SelectItem value="paid">Đã trả</SelectItem>
              <SelectItem value="overdue">Quá hạn</SelectItem>
            </SelectContent>
          </Select>
          <Select onValueChange={handleLenderChange} defaultValue="all">
            <SelectTrigger className="min-h-11 flex-1"><SelectValue placeholder="Chủ nợ" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tất cả</SelectItem>
              {lenders.map((lender) => (
                <SelectItem key={lender.id} value={lender.id.toString()}>{lender.name}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {/* Sort selector — restores desktop sorting capability */}
        <div className="flex items-center gap-2">
          <ArrowDownUp className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          <Select
            value={loanFilters.sortBy ?? 'created_at'}
            onValueChange={(v) => setLoanFilters((prev) => ({ ...prev, sortBy: v, page: 1 }))}
          >
            <SelectTrigger className="min-h-11 flex-1 text-xs">
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
            size="sm"
            className="min-h-11 px-3 text-xs"
            onClick={() =>
              setLoanFilters((prev) => ({
                ...prev,
                sortOrder: prev.sortOrder === 'asc' ? 'desc' : 'asc',
              }))
            }
            aria-label="Đảo chiều sắp xếp"
          >
            {loanFilters.sortOrder === 'asc' ? 'Tăng' : 'Giảm'}
          </Button>
        </div>
      </div>

      {/* Loan list */}
      <div className="flex-1 px-4">
        {filteredLoans.length === 0 ? (
          <div className="text-center py-12">
            <Landmark className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 typography-title-large">Chưa có khoản vay nào</h3>
            <p className="mt-2 typography-body-medium text-muted-foreground">Tạo khoản vay đầu tiên để bắt đầu quản lý.</p>
          </div>
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
