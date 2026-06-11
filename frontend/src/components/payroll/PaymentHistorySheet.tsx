import { useState, useMemo, useCallback, useEffect } from 'react';
import { SortingState } from '@tanstack/react-table';
import { FileSpreadsheet } from 'lucide-react';
import { SlideSheetTemplate } from '@/components/sheets/templates/SlideSheetTemplate';
import { PaymentHistoryTable } from './PaymentHistoryTable';
import { PaymentHistoryFilters } from './PaymentHistoryFilters';
import { PaymentHistoryMobileList } from './mobile/PaymentHistoryMobileList';
import { InfiniteScrollContainer } from '@/components/ui/infinite-scroll-container';
import { Button } from '@/components/ui/button';
import { usePaymentHistories, useInfinitePaymentHistories, useExportPaymentHistories } from '@/hooks/api/usePayrolls';
import { useMediaQuery } from '@/hooks/useBreakpoint';
import { dateToString } from '@/utils/dateHelpers';
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination';
import type { PaymentHistoryFilters as PaymentHistoryFiltersType } from '@/types/api/payroll.types';

interface PaymentHistorySheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export function PaymentHistorySheet({ isOpen, onClose }: PaymentHistorySheetProps) {
  const isMobile = useMediaQuery('(max-width: 767px)');
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 20;

  // Sorting state for DataTable
  const [sorting, setSorting] = useState<SortingState>(() => [
    { id: 'paid_date', desc: true },
  ]);

  // Export functionality
  const exportPaymentHistories = useExportPaymentHistories();

  // Filters state - Initialize with last 30 days by default
  const [filters, setFilters] = useState<Omit<PaymentHistoryFiltersType, 'page' | 'pageSize'>>(() => {
    const now = new Date();
    const thirtyDaysAgo = new Date(now.getTime() - (30 * 24 * 60 * 60 * 1000));
    return {
      sortBy: 'paid_date',
      sortOrder: 'desc',
      fromDate: dateToString(thirtyDaysAgo),
      toDate: dateToString(now),
    };
  });

  // Map TanStack column IDs to API sortBy field names
  const sortByMap = useMemo<Record<string, PaymentHistoryFiltersType['sortBy']>>(() => ({
    total_paid_amount: 'amount',
    employee_name: 'employee_name',
    project_name: 'project_name',
    paid_date: 'paid_date',
  }), []);

  // Sync sorting state to filter params
  useEffect(() => {
    if (sorting.length === 0) return;
    const col = sorting[0];
    const apiSortBy = sortByMap[col.id] ?? (col.id as PaymentHistoryFiltersType['sortBy']);
    setFilters(prev => ({
      ...prev,
      sortBy: apiSortBy,
      sortOrder: col.desc ? 'desc' : 'asc',
    }));
    setCurrentPage(1);
  }, [sorting, sortByMap]);

  // Handle filter changes
  const handleFiltersChange = useCallback((newFilters: PaymentHistoryFiltersType) => {
    setFilters((prevFilters) => {
      // Merge with previous filters, excluding page and pageSize
      const { page: _page, pageSize: _pageSize, ...restNewFilters } = newFilters;
      return {
        ...prevFilters,
        ...restNewFilters,
      };
    });
    if (newFilters.page !== currentPage) {
      setCurrentPage(newFilters.page || 1);
    }
  }, [currentPage]);

  // Clear all filters
  const handleClearFilters = useCallback(() => {
    setFilters({
      sortBy: 'paid_date',
      sortOrder: 'desc',
    });
    setCurrentPage(1);
  }, []);

  // Check if any filters are active
  const hasFilters = useMemo(() => {
    return Boolean(
      filters.search ||
      filters.fromDate ||
      filters.toDate ||
      filters.projectId?.length ||
      filters.employeeId?.length ||
      filters.position
    );
  }, [filters]);

  // Desktop: Standard pagination
  const {
    data: desktopData,
    isLoading: isLoadingDesktop,
  } = usePaymentHistories(
    !isMobile && isOpen ? {
      ...filters,
      page: currentPage,
      pageSize,
    } : undefined
  );

  // Mobile: Infinite scroll
  const {
    data: mobileData,
    isLoading: isLoadingMobile,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfinitePaymentHistories(
    isMobile && isOpen ? filters : undefined
  );

  // Flatten mobile data for infinite scroll
  const mobilePayments = useMemo(() => {
    if (!mobileData?.pages) return [];
    return mobileData.pages.flatMap(page => page?.data || []);
  }, [mobileData]);

  const handlePageChange = (page: number) => {
    setCurrentPage(page);
  };

  // Handle export
  const handleExport = useCallback(() => {
    if (!filters.fromDate || !filters.toDate) {
      // This shouldn't happen since we always have default dates, but just in case
      return;
    }

    exportPaymentHistories.mutate({
      fromDate: filters.fromDate,
      toDate: filters.toDate,
    });
  }, [filters.fromDate, filters.toDate, exportPaymentHistories]);

  const renderPagination = () => {
    if (!desktopData?.pagination) return null;

    const { page, totalPages } = desktopData.pagination;
    const pages = [];

    // Show max 5 page numbers
    const maxPages = 5;
    let startPage = Math.max(1, page - Math.floor(maxPages / 2));
    const endPage = Math.min(totalPages, startPage + maxPages - 1);

    if (endPage - startPage < maxPages - 1) {
      startPage = Math.max(1, endPage - maxPages + 1);
    }

    for (let i = startPage; i <= endPage; i++) {
      pages.push(i);
    }

    return (
      <Pagination className="mt-4">
        <PaginationContent>
          <PaginationItem>
            <PaginationPrevious
              onClick={() => page > 1 && handlePageChange(page - 1)}
              className={page <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
            />
          </PaginationItem>
          {pages.map((pageNum) => (
            <PaginationItem key={pageNum}>
              <PaginationLink
                onClick={() => handlePageChange(pageNum)}
                isActive={pageNum === page}
                className="cursor-pointer"
              >
                {pageNum}
              </PaginationLink>
            </PaginationItem>
          ))}
          <PaginationItem>
            <PaginationNext
              onClick={() => page < totalPages && handlePageChange(page + 1)}
              className={page >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
            />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    );
  };

  return (
    <SlideSheetTemplate
      isOpen={isOpen}
      onClose={onClose}
      title="Lịch sử trả lương"
      description="Danh sách các lần thanh toán lương cho nhân viên"
      size="full"
      className="w-full max-w-[2000px]"
      headerActions={
        <Button
          size="sm"
          onClick={handleExport}
          disabled={exportPaymentHistories.isPending}
          className="h-8 px-3 bg-green-600 hover:bg-green-700 text-white text-xs"
        >
          <FileSpreadsheet className="h-3.5 w-3.5 mr-1.5" />
          Xuất Excel
        </Button>
      }
    >
      <div className="relative h-full">
        <div className="space-y-4">
          {/* Filters */}
          <PaymentHistoryFilters
            filters={{ ...filters, page: currentPage, pageSize }}
            onFiltersChange={handleFiltersChange}
            onClearFilters={handleClearFilters}
            hasFilters={hasFilters}
            totalResults={desktopData?.pagination?.totalRecords}
          />

          {/* Desktop: Table with pagination */}
          {!isMobile && (
            <>
              <PaymentHistoryTable
                data={desktopData?.data || []}
                isLoading={isLoadingDesktop}
                pagination={desktopData?.pagination}
                sorting={sorting}
                onSortingChange={setSorting}
              />
              {renderPagination()}
            </>
          )}

          {/* Mobile: Cards with infinite scroll */}
          {isMobile && (
            <InfiniteScrollContainer
              onLoadMore={() => { fetchNextPage(); }}
              hasMore={!!hasNextPage}
              isLoading={isFetchingNextPage}
              className="space-y-3"
            >
              <PaymentHistoryMobileList
                data={mobilePayments}
                isLoading={isLoadingMobile && mobilePayments.length === 0}
              />
            </InfiniteScrollContainer>
          )}
        </div>

      </div>
    </SlideSheetTemplate>
  );
}
