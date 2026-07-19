import { useState, useMemo, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { SortingState } from "@tanstack/react-table";
import { ArrowLeft, FileSpreadsheet, History, Receipt } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { MobilePageShell, MobileSurface } from "@/components/shared/MobilePageShell";
import { PaymentHistoryFilters } from "@/components/payroll/PaymentHistoryFilters";
import { PaymentHistoryMobileList } from "@/components/payroll/mobile/PaymentHistoryMobileList";
import { InfiniteScrollContainer } from "@/components/ui/infinite-scroll-container";
import {
  useInfinitePaymentHistories,
  useExportPaymentHistories,
} from "@/hooks/api/usePayrolls";
import { dateToString } from "@/utils/dateHelpers";
import type { PaymentHistoryFilters as PaymentHistoryFiltersType } from "@/types/api/payroll.types";

const PaymentHistoryPage = () => {
  const navigate = useNavigate();

  const [sorting] = useState<SortingState>(() => [
    { id: "paid_date", desc: true },
  ]);

  const exportPaymentHistories = useExportPaymentHistories();

  const [filters, setFilters] = useState<
    Omit<PaymentHistoryFiltersType, "page" | "pageSize">
  >(() => {
    const now = new Date();
    const thirtyDaysAgo = new Date(now.getTime() - 30 * 24 * 60 * 60 * 1000);
    return {
      sortBy: "paid_date",
      sortOrder: "desc",
      fromDate: dateToString(thirtyDaysAgo),
      toDate: dateToString(now),
    };
  });

  // Keep sort in sync with sorting state
  useEffect(() => {
    if (sorting.length === 0) return;
    const col = sorting[0];
    const sortByMap: Record<string, PaymentHistoryFiltersType["sortBy"]> = {
      total_paid_amount: "amount",
      employee_name: "employee_name",
      project_name: "project_name",
      paid_date: "paid_date",
    };
    const apiSortBy =
      sortByMap[col.id] ?? (col.id as PaymentHistoryFiltersType["sortBy"]);
    setFilters((prev) => ({
      ...prev,
      sortBy: apiSortBy,
      sortOrder: col.desc ? ("desc" as const) : ("asc" as const),
    }));
  }, [sorting]);

  const handleFiltersChange = useCallback(
    (newFilters: PaymentHistoryFiltersType) => {
      setFilters((prev) => {
        const { page: _page, pageSize: _pageSize, ...rest } = newFilters;
        return { ...prev, ...rest };
      });
    },
    [],
  );

  const handleClearFilters = useCallback(() => {
    setFilters({ sortBy: "paid_date", sortOrder: "desc" });
  }, []);

  const hasFilters = useMemo(
    () =>
      Boolean(
        filters.search ||
          filters.fromDate ||
          filters.toDate ||
          filters.projectId?.length ||
          filters.employeeId?.length ||
          filters.position,
      ),
    [filters],
  );

  const {
    data: mobileData,
    isLoading,
    fetchNextPage,
    hasNextPage,
    isFetchingNextPage,
  } = useInfinitePaymentHistories(filters);

  const payments = useMemo(
    () => mobileData?.pages.flatMap((p) => p?.data || []) ?? [],
    [mobileData],
  );

  const handleExport = useCallback(() => {
    if (!filters.fromDate || !filters.toDate) return;
    exportPaymentHistories.mutate({
      fromDate: filters.fromDate,
      toDate: filters.toDate,
    });
  }, [filters.fromDate, filters.toDate, exportPaymentHistories]);

  return (
    <MobilePageShell className="space-y-3">
      <MobilePageHeader
        title="Bút toán ngân hàng"
        subtitle="Danh sách các lần thanh toán lương"
        icon={Receipt}
        sticky={false}
        bordered={false}
        actions={
          <>
            <Button
              size="sm"
              onClick={handleExport}
              disabled={exportPaymentHistories.isPending}
              className="h-11 px-3 text-xs shrink-0 rounded-xl bg-primary text-primary-foreground hover:bg-primary/90"
            >
              <FileSpreadsheet className="h-3.5 w-3.5 mr-1.5" />
              Xuất
            </Button>
            <Button variant="outline" size="icon" className="h-11 w-11 rounded-xl border-slate-300 bg-white" onClick={() => navigate(-1)}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </>
        }
      />

      {/* Filters */}
      <MobileSurface className="p-3">
        <PaymentHistoryFilters
          filters={{ ...filters, page: 1, pageSize: 20 }}
          onFiltersChange={handleFiltersChange}
          onClearFilters={handleClearFilters}
          hasFilters={hasFilters}
        />
      </MobileSurface>

      {/* Infinite scroll list */}
      <MobileSurface className="flex-1 p-3">
        <InfiniteScrollContainer
          onLoadMore={() => { fetchNextPage(); }}
          hasMore={!!hasNextPage}
          isLoading={isFetchingNextPage}
          className="space-y-3"
        >
          <PaymentHistoryMobileList
            data={payments}
            isLoading={isLoading && payments.length === 0}
          />
        </InfiniteScrollContainer>
      </MobileSurface>
    </MobilePageShell>
  );
};

export default PaymentHistoryPage;
