import { ResponsiveTable } from "@/components/ui/responsive-table";
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminSectionCard,
  AdminFilterRow,
} from "@/components/shared/AdminPageFrame";
import { UserPageHeader } from "@/components/users/UserPageHeader";
import { UserStatsCards } from "@/components/users/UserStatsCards";
import { UserFiltersBar } from "@/components/users/UserFiltersBar";
import { useUserDataInfinite } from "@/hooks/users/useUserDataInfinite";
import { useUserModals } from "@/hooks/useModalNavigation";
import { useUserFiltersWithBackend } from "@/hooks/users/useUserFiltersWithBackend";
import { useUsersSummary, useResetPassword } from "@/hooks/api/useUsers";
import { createUserColumns } from "@/config/user-table-columns";
import { createUserMobileConfig } from "@/config/user-table-mobile";
import { Skeleton } from "@/components/ui/skeleton";
import { Loader2 } from "lucide-react";
import { EmptyState } from "@/components/shared/EmptyState";
import { useCallback } from "react";
import { useTableSorting } from "@/utils/sorting";
import { useInfiniteScroll } from "@/hooks/use-infinite-scroll";
import { useAuth } from "@/contexts";

const UsersPage = () => {
  const { user } = useAuth();
  const isAdvPartner = user?.role === "adv_partner";

  const filterState = useUserFiltersWithBackend();
  const userData = useUserDataInfinite({
    pageSize: filterState.filters.pageSize,
    sortBy: filterState.filters.sortBy,
    sortOrder: filterState.filters.sortOrder,
    search: filterState.filters.search,
    role: filterState.filters.role,
    last_login_today: filterState.filters.last_login_today,
  });
  const {
    data: summaryData,
    isLoading: summaryLoading,
    error: summaryError,
  } = useUsersSummary();
  const resetPasswordMutation = useResetPassword();
  const { openUserDetails, openAddUser } = useUserModals();

  const handleResetPassword = (userId: number, password: string) => {
    resetPasswordMutation.mutate({ id: userId, password });
  };

  const handleRoleSelect = useCallback(
    (role?: "admin" | "partner" | "employee" | "accountant") => {
      filterState.setRole(role);
    },
    [filterState],
  );

  const handleDropdownRoleChange = useCallback(
    (role: "admin" | "partner" | "employee" | "accountant" | undefined) => {
      filterState.setRole(role);
    },
    [filterState],
  );

  const columns = createUserColumns();
  const mobileConfig = createUserMobileConfig();

  const { sorting, onSortingChange } = useTableSorting(
    filterState.sortBy,
    filterState.sortOrder,
    filterState.onSortChange,
  );

  const { observerRef } = useInfiniteScroll({
    hasMore: userData.hasMore,
    isLoading: userData.isFetchingNextPage,
    onLoadMore: () => {
      userData.fetchNextPage();
    },
    threshold: 0.1,
    rootMargin: "200px",
  });

  if (userData.isLoading && userData.users.length === 0) {
    return (
      <AdminPageCanvas>
        <AdminPageHeaderCard>
          <Skeleton className="h-8 w-64" />
          <Skeleton className="mt-2 h-4 w-96" />
        </AdminPageHeaderCard>
        <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-2xl" />
          ))}
        </div>
        <AdminSectionCard>
          <Skeleton className="m-3 h-11 rounded-xl sm:m-4" />
          <Skeleton className="m-3 h-96 rounded-xl sm:m-4" />
        </AdminSectionCard>
      </AdminPageCanvas>
    );
  }

  return (
    <AdminPageCanvas>
      {!isAdvPartner && (
        <AdminPageHeaderCard>
          <UserPageHeader onAddUser={() => openAddUser()} />
        </AdminPageHeaderCard>
      )}

      {!isAdvPartner && (
        <UserStatsCards
          stats={summaryData}
          isLoading={summaryLoading}
          error={!!summaryError}
          onRoleSelect={handleRoleSelect}
          selectedRole={filterState.role}
          lastLoginToday={filterState.lastLoginToday}
        />
      )}

      <AdminSectionCard aria-label="Danh sách người dùng">
        <AdminFilterRow>
          <p className="text-sm font-semibold text-slate-800">Danh sách</p>
          <div className="flex flex-1 items-center justify-end">
            <UserFiltersBar
              search={filterState.search}
              onSearchChange={filterState.setSearch}
              role={isAdvPartner ? undefined : filterState.role}
              onRoleChange={handleDropdownRoleChange}
              onClearFilters={filterState.clearFilters}
              hasActiveFilters={filterState.hasActiveFilters}
              hideRoleFilter={isAdvPartner}
            />
          </div>
        </AdminFilterRow>

        <ResponsiveTable
          data={userData.users.filter((u) => u.role !== 'adv_partner')}
          columns={columns}
          mobileFields={mobileConfig.mobileFields}
          rowTitle={mobileConfig.rowTitle}
          rowSubtitle={mobileConfig.rowSubtitle}
          getRowId={(row) => String(row.id)}
          onRowClick={(user) => openUserDetails(user.id.toString())}
          sorting={sorting}
          onSortingChange={onSortingChange}
          emptyState={
            <EmptyState
              title="Không tìm thấy người dùng nào"
              description={filterState.hasActiveFilters
                ? "Không có kết quả phù hợp với bộ lọc của bạn."
                : "Hãy tạo người dùng đầu tiên để bắt đầu quản lý."}
              size="sm"
            />
          }
          accordionType="single"
          embedded
        />

        {userData.hasMore && !userData.isFetchingNextPage && (
          <div ref={observerRef} className="h-px w-full" aria-hidden="true" />
        )}

        {userData.isFetchingNextPage && (
          <div className="flex items-center justify-center border-t border-slate-200/70 py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            <span className="ml-2 text-sm text-muted-foreground">
              Đang tải thêm...
            </span>
          </div>
        )}

        {!userData.hasMore &&
          !userData.isFetchingNextPage &&
          userData.users.length > 0 && (
            <p className="border-t border-slate-200/70 py-3 text-center text-xs text-muted-foreground">
              {userData.users.length} người dùng
            </p>
          )}
      </AdminSectionCard>
    </AdminPageCanvas>
  );
};

export default UsersPage;
