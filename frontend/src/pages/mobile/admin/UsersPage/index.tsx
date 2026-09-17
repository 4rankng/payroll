import { ErrorState } from "@/components/ui/error-state";
import { useState, useCallback, useMemo } from "react";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { UserMobileList } from "@/components/users/UserMobileList";
import { EmptyState } from "@/components/shared/EmptyState";
import { useUserDataInfinite } from "@/hooks/users/useUserDataInfinite";
import { useUserModals } from "@/hooks/useModalNavigation";
import { useUserFiltersWithBackend } from "@/hooks/users/useUserFiltersWithBackend";
import { useUsersSummary } from "@/hooks/api/useUsers";
import { useAuth } from "@/contexts";
import {
  Users,
  UserCog,
  Briefcase,
  User,
  Plus,
  SlidersHorizontal,
  X,
} from "lucide-react";

const UsersPageMobile = () => {
  const [filterSheetOpen, setFilterSheetOpen] = useState(false);
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
  const { data: summaryData, isLoading: summaryLoading } = useUsersSummary();
  const { openUserDetails, openAddUser } = useUserModals();

  const handleRoleSelect = useCallback(
    (role?: "admin" | "partner" | "employee") => {
      filterState.setLastLoginToday(false);
      filterState.setRole(role);
    },
    [filterState],
  );

  const { hasMore, isFetchingNextPage, fetchNextPage } = userData;
  const handleLoadMore = useCallback(() => {
    if (hasMore && !isFetchingNextPage) {
      fetchNextPage();
    }
  }, [hasMore, isFetchingNextPage, fetchNextPage]);

  const stats = useMemo(() => {
    if (!summaryData) return [];
    return [
      {
        label: "Tổng",
        value: summaryData.total_users || 0,
        icon: Users,
        color: "text-blue-600",
        bg: "bg-blue-50",
        role: null as "admin" | "partner" | "employee" | null,
      },
      {
        label: "Quản trị",
        value: summaryData.total_admins || 0,
        icon: UserCog,
        color: "text-red-600",
        bg: "bg-red-50",
        role: "admin" as const,
      },
      {
        label: "Quản lý",
        value: summaryData.total_partners || 0,
        icon: Briefcase,
        color: "text-blue-600",
        bg: "bg-blue-50",
        role: "partner" as const,
      },
      {
        label: "Nhân viên",
        value: summaryData.total_employees || 0,
        icon: User,
        color: "text-muted-foreground",
        bg: "bg-muted/40",
        role: "employee" as const,
      },
    ];
  }, [summaryData]);

  const activeFilterCount = useMemo(() => {
    return filterState.role ? 1 : 0;
  }, [filterState.role]);

  if (userData.isLoading && userData.users.length === 0) {
    return (
      <div className="flex flex-col gap-3 p-4">
        <div className="flex items-center justify-between">
          <Skeleton className="h-7 w-32" />
          <Skeleton className="h-9 w-16" />
        </div>
        <div className="grid grid-cols-4 gap-1.5">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
        <Skeleton className="h-11 w-full rounded-xl" />
        <div className="space-y-2">
          {[...Array(6)].map((_, i) => (
            <Skeleton key={i} className="h-14 w-full rounded-xl" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-full flex-col pb-[calc(6rem+env(safe-area-inset-bottom))]">
      {/* Header */}
      <MobilePageHeader
        title="Người dùng"
        icon={Users}
        actions={
          !isAdvPartner ? (
            <Button
              size="sm"
              className="h-11 px-4 btn-admin-primary shrink-0"
              onClick={() => openAddUser()}
            >
              <Plus className="h-4 w-4 mr-1" />
              Thêm
            </Button>
          ) : undefined
        }
      />

      {/* Stats strip */}
      {!isAdvPartner && !summaryLoading && stats.length > 0 && (
        <div className="px-4 pb-3">
          <div className="grid grid-cols-4 gap-1.5">
            {stats.map((stat) => {
                            const isActive =
                stat.role !== null && filterState.role === stat.role;
              return (
                <button
                  key={stat.label}
                  onClick={() => {
                    if (stat.role === null) {
                      handleRoleSelect(undefined);
                      return;
                    }
                    handleRoleSelect(isActive ? undefined : stat.role);
                  }}
                  className={`flex min-h-14 min-w-0 flex-col items-center justify-center gap-1 rounded-xl border px-1.5 py-2 transition-all active:scale-95 ${isActive ? "border-primary/40 bg-primary/5" : "border-border/60 bg-card"}`}
                >
                  <span
                    className={`text-sm font-bold tabular-nums leading-none ${isActive ? "text-primary" : "text-foreground"}`}
                  >
                    {stat.value.toLocaleString("vi-VN")}
                  </span>
                  <span
                    className={`text-xs leading-tight ${isActive ? "text-primary" : "text-muted-foreground"}`}
                  >
                    {stat.label}
                  </span>
                </button>
              );
            })}
          </div>
        </div>
      )}
      {summaryLoading && !isAdvPartner && (
        <div className="grid grid-cols-4 gap-1.5 px-4 pb-3">
          {[...Array(4)].map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
          ))}
        </div>
      )}

      {/* Search + filter */}
      <div className="px-4 pb-3 flex gap-2">
        <MobileSearchInput
          value={filterState.search}
          onSearch={filterState.setSearch}
          placeholder="Tìm tên, email..."
          className={isAdvPartner ? 'flex-1' : ''}
        />
        {!isAdvPartner && (
          <Button
            variant="outline"
            size="icon"
            className="h-11 w-11 rounded-xl border-border/60 bg-card shrink-0 relative"
            onClick={() => setFilterSheetOpen(true)}
            aria-label="Bộ lọc"
          >
            <SlidersHorizontal className="h-4 w-4" />
            {activeFilterCount > 0 && (
              <span className="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-primary text-xs text-white flex items-center justify-center font-bold">
                {activeFilterCount}
              </span>
            )}
          </Button>
        )}
      </div>

      {/* Active filter chips */}
      {activeFilterCount > 0 && (
        <div className="px-4 pb-3 flex gap-2 flex-wrap">
          {filterState.role && (
            <button
              type="button"
              aria-label="Xóa lọc vai trò"
              className="inline-flex min-h-11 items-center gap-1 rounded-xl bg-secondary px-3 text-xs font-semibold text-secondary-foreground"
              onClick={() => filterState.setRole(undefined)}
            >
              {
                {
                  admin: "Quản trị viên",
                  partner: "Quản lý",
                  employee: "Nhân viên",
                  accountant: "Kế toán",
                }[filterState.role]
              }
              <X className="h-3 w-3" />
            </button>
          )}
          <button
            onClick={filterState.clearFilters}
            className="min-h-11 px-1 text-xs text-muted-foreground underline underline-offset-2"
          >
            Xóa tất cả
          </button>
        </div>
      )}

      {/* List with infinite scroll */}
      <div className="flex-1 px-4">
        {userData.error && (
          <div role="alert">
            <ErrorState
              message={userData.isFetchNextPageError
                ? "Không thể tải thêm người dùng. Danh sách đã tải vẫn được giữ lại."
                : "Không thể tải danh sách người dùng. Vui lòng thử lại."}
              onRetry={() => void (userData.isFetchNextPageError ? userData.fetchNextPage() : userData.refetch())}
              className="px-4"
            />
          </div>
        )}
        {(!userData.error || userData.users.length > 0) && <UserMobileList
          users={userData.users.filter((u) => u.role !== 'adv_partner')}
          onRowClick={(user) => openUserDetails(user.id.toString())}
          emptyState={
            <EmptyState
              title="Không tìm thấy người dùng nào"
              description={filterState.hasActiveFilters
                ? "Không có kết quả phù hợp."
                : "Hãy tạo người dùng đầu tiên."}
              size="sm"
            />
          }
        />}
        {!userData.error && userData.hasMore && (
          <div className="flex justify-center py-4">
            <Button
              variant="outline"
              size="sm"
              className="gap-2"
              onClick={handleLoadMore}
              disabled={userData.isFetchingNextPage}
            >
              {userData.isFetchingNextPage ? (
                <>
                  <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                  Đang tải...
                </>
              ) : (
                "Tải thêm"
              )}
            </Button>
          </div>
        )}
        {userData.isFetchingNextPage && (
          <div className="flex justify-center py-2">
            <div className="w-4 h-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        )}
        {!userData.error && !userData.hasMore &&
          !userData.isFetchingNextPage &&
          userData.users.filter((u) => u.role !== 'adv_partner').length > 0 && (
            <p className="text-center py-3 text-xs text-muted-foreground">
              {userData.users.filter((u) => u.role !== 'adv_partner').length} người dùng
            </p>
          )}
      </div>

      {/* Filter sheet */}
      <Sheet open={filterSheetOpen} onOpenChange={setFilterSheetOpen}>
        <SheetContent
          side="bottom"
          className="max-h-[85dvh] overflow-y-auto rounded-t-2xl px-4 pt-4 pb-[calc(1.25rem+env(safe-area-inset-bottom))]"
        >
          <SheetHeader className="pb-4">
            <SheetTitle>Bộ lọc</SheetTitle>
          </SheetHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Vai trò</label>
              <Select
                value={filterState.role ?? "all"}
                onValueChange={(v) =>
                  filterState.setRole(
                    v === "all"
                      ? undefined
                      : (v as "admin" | "partner" | "employee" | "accountant"),
                  )
                }
              >
                <SelectTrigger aria-label="Lọc vai trò người dùng" className="h-11">
                  <SelectValue placeholder="Tất cả vai trò" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Tất cả vai trò</SelectItem>
                  <SelectItem value="admin">Quản trị viên</SelectItem>
                  <SelectItem value="partner">Quản lý</SelectItem>
                  <SelectItem value="employee">Nhân viên</SelectItem>
                  <SelectItem value="accountant">Kế toán</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Sắp xếp</label>
              <div className="grid grid-cols-[minmax(0,1fr)_auto] gap-2">
                <Select
                  value={filterState.sortBy}
                  onValueChange={(value) =>
                    filterState.onSortChange(value, filterState.sortOrder)
                  }
                >
                  <SelectTrigger aria-label="Sắp xếp người dùng" className="h-11 min-w-0">
                    <SelectValue placeholder="Sắp xếp theo" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="username">Người dùng</SelectItem>
                    <SelectItem value="role">Vai trò</SelectItem>
                    <SelectItem value="last_login">Hoạt động cuối</SelectItem>
                    <SelectItem value="created_at">Ngày tạo</SelectItem>
                  </SelectContent>
                </Select>
                <Button
                  type="button"
                  variant="outline"
                  className="h-11 min-w-20"
                  onClick={() =>
                    filterState.onSortChange(
                      filterState.sortBy,
                      filterState.sortOrder === "asc" ? "desc" : "asc",
                    )
                  }
                >
                  {filterState.sortOrder === "asc" ? "Tăng" : "Giảm"}
                </Button>
              </div>
            </div>
            <div className="grid grid-cols-1 gap-3 pt-2 min-[380px]:grid-cols-2">
              <Button
                variant="outline"
                className="flex-1 h-11"
                onClick={() => {
                  filterState.clearFilters();
                  setFilterSheetOpen(false);
                }}
              >
                Xóa bộ lọc
              </Button>
              <Button
                className="flex-1 h-11 btn-admin-primary"
                onClick={() => setFilterSheetOpen(false)}
              >
                Áp dụng
              </Button>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
};

export default UsersPageMobile;
