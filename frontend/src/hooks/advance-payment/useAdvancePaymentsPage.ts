import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { format } from "date-fns";
import {
  useAdvancePayments,
  useAdvancePaymentSummary,
  useAdminCancelAdvancePayment,
  useFlexPayEmployees,
  useExportFlexPayEmployees,
  useAvailableMonths,
} from "@/hooks/api/useAdvancePayments";
import { GroupedStatCard, StatItem } from "@/components/shared/GroupedStatCard";
import { formatCurrency } from "@/utils/formatters";
import { Wallet, TrendingUp } from "lucide-react";
import type {
  AdvancePaymentFilters,
  AdvancePaymentRequestStatus,
  FlexPayEmployeeFilters,
} from "@/types/api/advance-payment.types";

export function useAdvancePaymentsPage(options?: { employeesTabActive?: boolean }) {
  const employeesTabActive = options?.employeesTabActive ?? false;
  const [filters, setFilters] = useState<AdvancePaymentFilters>({
    page: 1,
    pageSize: 20,
  });
  const [selectedMonth, setSelectedMonth] = useState<string>(
    format(new Date(), "yyyy-MM"),
  );
  const [flexPayFilters, setFlexPayFilters] = useState<FlexPayEmployeeFilters>({
    page: 1,
    pageSize: 20,
    sortBy: "max_advance_amount",
    sortOrder: "DESC",
    forMonth: selectedMonth,
  });
  const [searchInput, setSearchInput] = useState("");
  const [flexPaySearchInput, setFlexPaySearchInput] = useState("");
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const flexPaySearchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);


  const [cancelConfirmId, setCancelConfirmId] = useState<number | null>(null);

  const { data: requestsResponse } = useAdvancePayments(filters);
  const { data: summary, isLoading: summaryLoading } =
    useAdvancePaymentSummary({ forMonth: selectedMonth === "all" ? undefined : selectedMonth });
  const { data: flexPayResponse } = useFlexPayEmployees(flexPayFilters);
  const { data: availableMonthsData } = useAvailableMonths();

  const requests = useMemo(
    () =>
      Array.isArray(requestsResponse?.data)
        ? (requestsResponse!
            .data as import("@/types/api/advance-payment.types").AdvancePaymentListItem[])
        : [],
    [requestsResponse],
  );
  const pagination = requestsResponse?.pagination;

  const flexPayEmployees = useMemo(
    () =>
      Array.isArray(flexPayResponse?.data)
        ? (flexPayResponse!
            .data as import("@/types/api/advance-payment.types").FlexPayEmployeeListItem[])
        : [],
    [flexPayResponse],
  );
  const flexPayPagination = flexPayResponse?.pagination;
  const flexPayMonth = selectedMonth;
  const availableMonths = useMemo(
    () => Array.isArray(availableMonthsData?.data) ? availableMonthsData.data : [],
    [availableMonthsData],
  );

  useEffect(() => {
    setFlexPayFilters((prev) => ({
      ...prev,
      forMonth: selectedMonth,
      page: 1,
    }));
  }, [selectedMonth]);

  useEffect(() => {
    setFilters((prev) => {
      const newFilters = { ...prev, page: 1 };
      delete newFilters.fromDate;
      delete newFilters.toDate;
      if (selectedMonth !== "all") {
        newFilters.forMonth = selectedMonth;
      } else {
        delete newFilters.forMonth;
      }
      return newFilters;
    });
  }, [selectedMonth]);

  const cancelMutation = useAdminCancelAdvancePayment();
  const exportFlexPayMutation = useExportFlexPayEmployees();

  const statusCounts = useMemo(() => {
    if (!summary?.data) return undefined;
    const d = summary.data;
    return {
      all: d.totalRequests,
      pending: d.totalPending,
      completed: d.totalPaid,
      failed: d.totalFailed,
      cancelled: d.totalCancelled,
    };
  }, [summary]);

  const overviewStats = useMemo<StatItem[]>(() => {
    if (!summary?.data) return [];
    const d = summary.data;
    return [
      { label: "Tổng yêu cầu", value: d.totalRequests },
      { label: "Chờ xử lý", value: d.totalPending, variant: "accent" as const },
      { label: "Đã thanh toán", value: d.totalPaid },
      { label: "Đã hủy", value: d.totalCancelled },
    ];
  }, [summary]);

  const financeStats = useMemo<StatItem[]>(() => {
    if (!summary?.data) return [];
    const d = summary.data;
    return [
      { label: "Tổng yêu cầu", value: formatCurrency(d.totalAmount) },
      { label: "Đã trả", value: formatCurrency(d.totalPaidAmount) },
      { label: "Phí thu được", value: formatCurrency(d.totalFeeEarned) },
      { label: "Tổng phí thu", value: formatCurrency(d.totalFeeEarnedAllTime) },
    ];
  }, [summary]);

  const providerFees = useMemo(() => {
    if (!summary?.data) return { monthlyProviderFee: undefined, totalProviderFee: undefined };
    return {
      monthlyProviderFee: summary.data.totalProviderFee,
      totalProviderFee: summary.data.totalProviderFeeAllTime,
    };
  }, [summary]);

  const handleCancelRequest = useCallback(
    (id: number) => {
      setCancelConfirmId(id);
    },
    [],
  );

  const confirmCancelRequest = useCallback(() => {
    if (cancelConfirmId !== null) {
      cancelMutation.mutate(cancelConfirmId);
      setCancelConfirmId(null);
    }
  }, [cancelConfirmId, cancelMutation]);

  const dismissCancelConfirm = useCallback(() => {
    setCancelConfirmId(null);
  }, []);

  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setSearchInput(value);
      if (searchTimerRef.current) clearTimeout(searchTimerRef.current);
      searchTimerRef.current = setTimeout(
        () => setFilters((prev) => ({ ...prev, search: value, page: 1 })),
        300,
      );
    },
    [],
  );

  // String-based version for use with SearchBar / MobileSearchInput
  const handleSearch = useCallback((value: string) => {
    setSearchInput(value);
    setFilters((prev) => ({ ...prev, search: value || undefined, page: 1 }));
  }, []);

  const handleFlexPaySearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setFlexPaySearchInput(value);
      if (flexPaySearchTimerRef.current) clearTimeout(flexPaySearchTimerRef.current);
      flexPaySearchTimerRef.current = setTimeout(
        () => setFlexPayFilters((prev) => ({ ...prev, search: value, page: 1 })),
        300,
      );
    },
    [],
  );

  // String-based version for use with SearchBar / MobileSearchInput
  const handleFlexPaySearch = useCallback((value: string) => {
    setFlexPaySearchInput(value);
    setFlexPayFilters((prev) => ({ ...prev, search: value || undefined, page: 1 }));
  }, []);

  const handleStatusChange = useCallback((value: string) => {
    setFilters((prev) => ({
      ...prev,
      status:
        value === "all" ? undefined : (value as AdvancePaymentRequestStatus),
      page: 1,
    }));
  }, []);

  const handlePageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, page }));
  }, []);

  const handlePageSizeChange = useCallback((pageSize: number) => {
    setFilters((prev) => ({ ...prev, pageSize, page: 1 }));
  }, []);

  const handleFlexPaySort = useCallback((sortBy: string) => {
    setFlexPayFilters((prev) => ({
      ...prev,
      sortBy,
      sortOrder: prev.sortBy === sortBy && prev.sortOrder === "DESC" ? "ASC" : "DESC",
      page: 1,
    }));
  }, []);

  const handleFlexPayPageChange = useCallback((page: number) => {
    setFlexPayFilters((prev) => ({ ...prev, page }));
  }, []);

  const handleFlexPayPageSizeChange = useCallback((pageSize: number) => {
    setFlexPayFilters((prev) => ({ ...prev, pageSize, page: 1 }));
  }, []);

  const handleFlexPayRowClick = useCallback(
    (
      row: import("@/types/api/advance-payment.types").FlexPayEmployeeListItem,
    ) => {
      setSearchInput(row.cccd);
      setFilters((prev) => ({ ...prev, search: row.cccd, page: 1 }));
    },
    [],
  );

  const handleExportFlexPayEmployees = useCallback(() => {
    exportFlexPayMutation.mutate({ forMonth: flexPayMonth });
  }, [exportFlexPayMutation, flexPayMonth]);

  return {
    filters,
    setFilters,
    flexPayFilters,
    searchInput,
    flexPaySearchInput,
    selectedMonth,
    setSelectedMonth,
    requests,
    pagination,
    flexPayEmployees,
    flexPayPagination,
    flexPayMonth,
    availableMonths,
    selectedViewMonth: selectedMonth,
    setSelectedViewMonth: setSelectedMonth,
    summary,
    summaryLoading,
    statusCounts,
    overviewStats,
    financeStats,
    providerFees,
    cancelMutation,
    cancelConfirmId,
    exportFlexPayMutation,
    handleCancelRequest,
    confirmCancelRequest,
    dismissCancelConfirm,
    handleSearchChange,
    handleSearch,
    handleFlexPaySearchChange,
    handleFlexPaySearch,
    handleStatusChange,
    handlePageChange,
    handlePageSizeChange,
    handleFlexPayPageChange,
    handleFlexPayPageSizeChange,
    handleFlexPayRowClick,
    handleFlexPaySort,
    handleExportFlexPayEmployees,
  };
}
