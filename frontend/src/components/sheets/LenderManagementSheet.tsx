import { useState, useCallback } from 'react';
import { Button } from '@/components/ui/button';
import { SlideSheetTemplate } from './templates/SlideSheetTemplate';
import { AddLenderSheet } from './AddLenderSheet';
import { EditLenderSheet } from './EditLenderSheet';
import { LenderCard } from '@/components/lenders/LenderCard';
import { useLenders } from '@/hooks/api/useLoans';
import { Plus, Users2, ChevronLeft, ChevronRight } from 'lucide-react';
import { SearchBar } from '@/components/shared/SearchBar';
import type { LenderFilters } from '@/types/api/loan.types';
import { cn } from '@/lib/utils';

interface LenderManagementSheetProps {
  isOpen: boolean;
  onClose: () => void;
}

export function LenderManagementSheet({ isOpen, onClose }: LenderManagementSheetProps) {
  const [filters, setFilters] = useState<LenderFilters>({
    page: 1,
    pageSize: 20,
    sortBy: 'created_at',
    sortOrder: 'desc',
  });

  const [showAddSheet, setShowAddSheet] = useState(false);
  const [editingLenderId, setEditingLenderId] = useState<number | null>(null);

  const { data: lendersResponse, isLoading } = useLenders(filters, { enabled: isOpen });

  const lenders = lendersResponse?.data || [];
  const pagination = lendersResponse?.pagination;

  const handleSearch = useCallback((value: string) => {
    setFilters(prev => ({
      ...prev,
      search: value.trim() || undefined,
      page: 1,
    }));
  }, []);

  // Pagination handlers
  const handlePreviousPage = useCallback(() => {
    setFilters(prev => ({
      ...prev,
      page: Math.max(1, prev.page! - 1),
    }));
  }, []);

  const handleNextPage = useCallback(() => {
    if (pagination && pagination.page < pagination.totalPages) {
      setFilters(prev => ({
        ...prev,
        page: (prev.page || 1) + 1,
      }));
    }
  }, [pagination]);

  return (
    <>
      <SlideSheetTemplate
        isOpen={isOpen}
        onClose={onClose}
        title="Quản lý chủ nợ"
        description="Danh sách chủ nợ và thông tin liên quan"
        size="large"
        compact
        className={cn(
          "w-full",
          "sm:w-[600px]",
          "md:w-[700px]",
          "lg:w-[800px]",
          "xl:w-[900px]",
          "2xl:w-[900px]"
        )}
      >
        <div className="space-y-6 mt-2">
          {/* Filters and Actions */}
          <div className="flex flex-col sm:flex-row gap-4 justify-between">
            <div className="flex-1 max-w-sm">
              <SearchBar
                searchTerm={filters.search || ''}
                onSearchChange={handleSearch}
                placeholder="Tìm kiếm chủ nợ..."
                className="w-full h-9"
              />
            </div>
            <Button onClick={() => setShowAddSheet(true)} size="sm">
              <Plus className="h-4 w-4 mr-2" />
              Thêm chủ nợ
            </Button>
          </div>

          {/* Lenders List */}
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <div className="typography-body-medium text-muted-foreground">Đang tải...</div>
            </div>
          ) : lenders.length === 0 ? (
            <div className="text-center py-12">
              <Users2 className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 typography-title-large">Chưa có chủ nợ nào</h3>
              <p className="mt-2 typography-body-medium text-muted-foreground">
                Thêm chủ nợ đầu tiên để bắt đầu quản lý khoản vay.
              </p>
              <Button onClick={() => setShowAddSheet(true)} className="mt-4">
                <Plus className="h-4 w-4 mr-2" />
                Thêm chủ nợ
              </Button>
            </div>
          ) : (
            <>
              {/* Cards Grid */}
              <div className="grid grid-cols-1 gap-4">
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
                <div className="flex items-center justify-between pt-4 border-t">
                  <div className="typography-body-small text-muted-foreground">
                    Trang {pagination.page} / {pagination.totalPages} ({pagination.totalRecords} kết quả)
                  </div>
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
      </SlideSheetTemplate>

      {/* Add Lender Sheet */}
      <AddLenderSheet
        isOpen={showAddSheet}
        onClose={() => setShowAddSheet(false)}
      />

      {/* Edit Lender Sheet */}
      {editingLenderId && (
        <EditLenderSheet
          isOpen={!!editingLenderId}
          onClose={() => setEditingLenderId(null)}
          lenderId={editingLenderId}
        />
      )}
    </>
  );
}

export default LenderManagementSheet;
