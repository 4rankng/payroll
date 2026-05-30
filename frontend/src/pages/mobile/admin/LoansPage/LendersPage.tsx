import { useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Plus, Users2, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MobileSearchInput } from "@/components/shared/MobileSearchInput";
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
    <div className="flex flex-col min-h-full pb-20">
      {/* Sticky header with back button */}
      <div className="sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 border-b border-border/40 shrink-0">
        <div className="flex items-center gap-3 px-4 pt-4 pb-3">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 shrink-0 -ml-1"
            onClick={() => navigate(-1)}
            aria-label="Quay lại"
          >
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-primary/10 shrink-0">
              <Users2 className="h-4 w-4 text-primary" />
            </div>
            <div className="min-w-0">
              <h1 className="text-base font-semibold text-foreground leading-tight">
                Quản lý chủ nợ
              </h1>
              <p className="text-xs text-muted-foreground leading-tight">
                Danh sách chủ nợ và thông tin liên quan
              </p>
            </div>
          </div>
          <Button
            size="sm"
            className="btn-admin-primary shrink-0"
            onClick={() => setShowAddSheet(true)}
          >
            <Plus className="h-4 w-4" />
            Thêm
          </Button>
        </div>
      </div>

      {/* Search */}
      <div className="px-4 pt-3 pb-2">
        <MobileSearchInput
          value={filters.search ?? ""}
          onSearch={handleSearch}
          placeholder="Tìm kiếm chủ nợ..."
          className="h-10"
        />
      </div>

      {/* Content */}
      <div className="flex-1 px-4 py-2">
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="text-sm text-muted-foreground">Đang tải...</div>
          </div>
        ) : lenders.length === 0 ? (
          <div className="text-center py-12">
            <Users2 className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 text-base font-semibold">Chưa có chủ nợ nào</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Thêm chủ nợ đầu tiên để bắt đầu quản lý khoản vay.
            </p>
            <Button onClick={() => setShowAddSheet(true)} className="mt-4">
              <Plus className="h-4 w-4 mr-2" />
              Thêm chủ nợ
            </Button>
          </div>
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
              <div className="flex items-center justify-between pt-4 mt-2 border-t">
                <p className="text-xs text-muted-foreground">
                  Trang {pagination.page} / {pagination.totalPages} (
                  {pagination.totalRecords} kết quả)
                </p>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handlePreviousPage}
                    disabled={pagination.page === 1}
                  >
                    <ChevronLeft className="h-4 w-4 mr-1" />
                    Trước
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleNextPage}
                    disabled={pagination.page === pagination.totalPages}
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
