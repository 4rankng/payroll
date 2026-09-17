import { useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Plus, Users2, ChevronLeft, ChevronRight, HandCoins } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
import { MobilePageHeader } from "@/components/shared/MobilePageHeader";
import { EmptyState } from "@/components/shared/EmptyState";
import { LenderCard } from "@/components/lenders/LenderCard";
import { AddLenderSheet } from "@/components/sheets/AddLenderSheet";
import { EditLenderSheet } from "@/components/sheets/EditLenderSheet";
import { useLenders } from "@/hooks/api/useLoans";
import type { LenderFilters } from "@/types/api/loan.types";

const LendersPage = () => {
  const navigate = useNavigate();

  const [filters, setFilters] = useState<LenderFilters>({
    page: 1,
    pageSize: 20,
    sortBy: "created_at",
    sortOrder: "desc",
  });
  const [showAddSheet, setShowAddSheet] = useState(false);
  const [editingLenderId, setEditingLenderId] = useState<number | null>(null);

  const { data: lendersResponse, isLoading } = useLenders(filters);

  const lenders = lendersResponse?.data || [];
  const pagination = lendersResponse?.pagination;

  const handleSearch = useCallback((value: string) => {
    setFilters((prev) => ({
      ...prev,
      search: value.trim() || undefined,
      page: 1,
    }));
  }, []);

  const handlePreviousPage = useCallback(() => {
    setFilters((prev) => ({ ...prev, page: Math.max(1, prev.page! - 1) }));
  }, []);

  const handleNextPage = useCallback(() => {
    if (pagination && pagination.page < pagination.totalPages) {
      setFilters((prev) => ({ ...prev, page: (prev.page || 1) + 1 }));
    }
  }, [pagination]);

  return (
    <div className="flex min-h-full flex-col pb-[calc(5rem+env(safe-area-inset-bottom))]">
      <MobilePageHeader
        icon={HandCoins}
        title="Chủ nợ"
        subtitle="Danh sách chủ nợ và thông tin liên quan"
        sticky={false}
        bordered={false}
        actions={
          <>
            <Button
              size="sm"
              className="btn-admin-primary h-11 w-11 shrink-0 p-0 min-[380px]:w-auto min-[380px]:px-3"
              aria-label="Thêm chủ nợ"
              onClick={() => setShowAddSheet(true)}
            >
              <Plus className="h-4 w-4" />
              <span className="hidden min-[380px]:inline">Thêm</span>
            </Button>
            <Button variant="ghost" size="icon" className="h-11 w-11" onClick={() => navigate(-1)} aria-label="Quay lại khoản vay">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </>
        }
      />

      {/* Search */}
      <div className="px-4 pt-3 pb-2">
        <MobileSearchInput
          value={filters.search ?? ""}
          onSearch={handleSearch}
          placeholder="Tìm kiếm chủ nợ..."
          className="h-11"
        />
      </div>

      {/* Content */}
      <div className="flex-1 px-4 py-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-sm text-muted-foreground">Đang tải...</div>
          </div>
        ) : lenders.length === 0 ? (
          <EmptyState
            icon={Users2}
            title="Chưa có chủ nợ nào"
            description="Thêm chủ nợ đầu tiên để bắt đầu quản lý khoản vay."
            action={{ label: "Thêm chủ nợ", onClick: () => setShowAddSheet(true) }}
          />
        ) : (
          <>
            <div className="grid grid-cols-1 gap-3">
              {lenders.map((lender) => (
                <LenderCard
                  key={lender.id}
                  lender={lender}
                  onClick={() => setEditingLenderId(lender.id)}
                />
              ))}
            </div>

            {/* Pagination */}
            {pagination && pagination.totalPages > 1 && (
              <div className="mt-2 grid grid-cols-1 gap-3 border-t pt-4 min-[380px]:grid-cols-[minmax(0,1fr)_auto] min-[380px]:items-center">
                <p className="break-words text-xs text-muted-foreground">
                  Trang {pagination.page} / {pagination.totalPages} (
                  {pagination.totalRecords} kết quả)
                </p>
                <div className="grid grid-cols-2 gap-2 min-[380px]:flex min-[380px]:items-center">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handlePreviousPage}
                    disabled={pagination.page === 1}
                    className="h-11"
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Trước
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleNextPage}
                    disabled={pagination.page === pagination.totalPages}
                    className="h-11"
                  >
                    Sau
                    <ChevronRight className="h-4 w-4 ml-1" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </div>

      <AddLenderSheet
        isOpen={showAddSheet}
        onClose={() => setShowAddSheet(false)}
      />
      {editingLenderId && (
        <EditLenderSheet
          isOpen={!!editingLenderId}
          onClose={() => setEditingLenderId(null)}
          lenderId={editingLenderId}
        />
      )}
    </div>
  );
};

export default LendersPage;
