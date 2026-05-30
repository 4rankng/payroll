import { ResponsiveTable } from "@/components/ui/responsive-table";
import { UserPageHeader } from "@/components/users/UserPageHeader";
import { UserStatsCards } from "@/components/users/UserStatsCards";
import { UserFiltersBar } from "@/components/users/UserFiltersBar";
import { AddUserSheet } from "@/components/sheets/AddUserSheet";
import { useUserDataInfinite } from "@/hooks/users/useUserDataInfinite";
import { useUserModals } from "@/hooks/useModalNavigation";
import { useUserFiltersWithBackend } from "@/hooks/users/useUserFiltersWithBackend";
import { useUsersSummary, useResetPassword } from "@/hooks/api/useUsers";
import { createUserColumns } from "@/config/user-table-columns";
import { createUserMobileConfig } from "@/config/user-table-mobile";
import { Skeleton } from "@/components/ui/skeleton";
import { Users, Loader2 } from "lucide-react";
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
    resetPasswordMutation.mutate({ id: userId, data: { password } });
  };

  const handleRoleSelect = useCallback(
    (role?: "admin" | "partner" | "employee") => {
      filterState.setRole(role);
    },
    [filterState],
  );

  const handleDropdownRoleChange = useCallback(
    (role: "admin" | "partner" | "employee" | undefined) => {
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
      <div className="p-4 space-y-6">
        <div className="space-y-2">
          <Skeleton className="h-8 w-64" />
          <Skeleton className="h-4 w-96" />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-96" />
      </div>
    );
  }

  return (
    <div className="min-h-full">
      <div className="p-4 lg:p-6 max-w-[1280px] mx-auto space-y-5">
        {!isAdvPartner && <UserPageHeader onAddUser={() => openAddUser()} />}

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

        <UserFiltersBar
          search={filterState.search}
          onSearchChange={filterState.setSearch}
          role={isAdvPartner ? undefined : filterState.role}
          onRoleChange={handleDropdownRoleChange}
          onClearFilters={filterState.clearFilters}
          hasActiveFilters={filterState.hasActiveFilters}
          hideRoleFilter={isAdvPartner}
        />

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
            <div className="text-center py-12">
              <Users className="mx-auto h-12 w-12 text-muted-foreground/50" />
              <h3 className="mt-4 typography-title-large">
                Không tìm thấy người dùng nào
              </h3>
              <p className="mt-2 typography-body-medium text-muted-foreground">
                {filterState.hasActiveFilters
                  ? "Không có kết quả phù hợp với bộ lọc của bạn."
                  : "Hãy tạo người dùng đầu tiên để bắt đầu quản lý."}
              </p>
            </div>
          }
          accordionType="single"
        />

        {userData.hasMore && !userData.isFetchingNextPage && (
          <div ref={observerRef} className="h-px w-full" aria-hidden="true" />
        )}

        {userData.isFetchingNextPage && (
          <div className="flex items-center justify-center py-6">
            <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            <span className="ml-2 typography-body-medium text-muted-foreground">
              Đang tải thêm...
            </span>
          </div>
        )}

        {!userData.hasMore &&
          !userData.isFetchingNextPage &&
          userData.users.length > 0 && (
            <p className="text-center py-3 text-xs text-muted-foreground">
              {userData.users.length} người dùng
            </p>
          )}
      </div>
    </div>
  );
};

export default UsersPage;
