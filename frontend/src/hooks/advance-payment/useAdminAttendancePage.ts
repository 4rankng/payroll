import { useState, useCallback, useMemo } from "react";
import { useAdminAttendances } from "@/hooks/api/useAdminAttendance";
import type { AttendanceFilters } from "@/types/api/attendance.types";

export function useAdminAttendancePage(options?: { active?: boolean }) {
  const [filters, setFilters] = useState<AttendanceFilters>({
    page: 1,
    limit: 50,
  });

  const { data, isLoading } = useAdminAttendances(
    filters,
    { enabled: options?.active }
  );

  const attendances = data?.data ?? [];
  const totalRecords = data?.total ?? 0;

  const handlePageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, page }));
  }, []);

  const handlePageSizeChange = useCallback((limit: number) => {
    setFilters((prev) => ({ ...prev, limit, page: 1 }));
  }, []);

  const handleStatusChange = useCallback((status: string) => {
    setFilters((prev) => ({
      ...prev,
      status: status === "all" ? undefined : status,
      page: 1,
    }));
  }, []);

  const pagination = useMemo(() => {
    return {
      page: filters.page ?? 1,
      limit: filters.limit ?? 50,
      totalRecords,
      totalPages: Math.ceil(totalRecords / (filters.limit ?? 50)),
    };
  }, [filters.page, filters.limit, totalRecords]);

  return {
    filters,
    attendances,
    totalRecords,
    isLoading,
    pagination,
    handlePageChange,
    handlePageSizeChange,
    handleStatusChange,
  };
}
