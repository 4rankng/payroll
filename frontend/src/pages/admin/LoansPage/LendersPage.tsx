import { useCallback, useState } from "react";
import {
  AdminPageCanvas,
  AdminPageHeaderCard,
  AdminFilterRow,
  AdminSectionCard,
} from "@/components/shared/AdminPageFrame";
import { PageHeader } from "@/components/shared/PageHeader";
import { EmptyState } from "@/components/shared/EmptyState";
import { SearchBar } from "@/components/shared/SearchBar";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { AddLenderSheet } from "@/components/sheets/AddLenderSheet";
import { EditLenderSheet } from "@/components/sheets/EditLenderSheet";
import { useLenders } from "@/hooks/api/useLoans";
import type { LenderFilters } from "@/types/api/loan.types";

/** Desktop variant of the lenders workspace (PAYR-23): dense table for
 *  >=768px, paired with the mobile card page via ResponsivePage. */
const LendersPage = () => {
  const [filters, setFilters] = useState<LenderFilters>({ page: 1, pageSize: 20 });
  const [showAddSheet, setShowAddSheet] = useState(false);
  const [editingLenderId, setEditingLenderId] = useState<number | null>(null);

  const { data: lendersResponse, isLoading } = useLenders(filters);
  const lenders = lendersResponse?.data || [];
  const pagination = lendersResponse?.pagination;

  const handleSearch = useCallback((value: string) => {
    setFilters((prev) => ({ ...prev, search: value.trim() || undefined, page: 1 }));
  }, []);

  const handlePageChange = useCallback((page: number) => {
    setFilters((prev) => ({ ...prev, page }));
  }, []);

  const handlePageSizeChange = useCallback((pageSize: number) => {
    setFilters((prev) => ({ ...prev, pageSize, page: 1 }));
  }, []);

  return (
    <AdminPageCanvas>
      <AdminPageHeaderCard>
        <PageHeader
          title="Chủ nợ"
          description="Danh sách chủ nợ và thông tin liên quan"
          actions={[
            {
              label: "Thêm chủ nợ",
              onClick: () => setShowAddSheet(true),
              variant: "default" as const,
            },
          ]}
        />
      </AdminPageHeaderCard>

      <AdminSectionCard aria-label="Danh sách chủ nợ">
        <AdminFilterRow>
          <p className="text-sm font-semibold text-fg-primary">Chủ nợ</p>
          <div className="flex flex-1 items-center justify-end">
            <SearchBar
              searchTerm={filters.search ?? ""}
              onSearchChange={handleSearch}
              placeholder="Tìm kiếm chủ nợ..."
            />
          </div>
        </AdminFilterRow>

        {isLoading ? (
          <p className="py-8 text-center text-sm text-muted-foreground">Đang tải...</p>
        ) : lenders.length === 0 ? (
          <EmptyState
            title="Chưa có chủ nợ nào"
            description="Thêm chủ nợ đầu tiên để bắt đầu quản lý khoản vay."
            size="sm"
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="whitespace-nowrap">Tên</TableHead>
                <TableHead className="whitespace-nowrap">CCCD</TableHead>
                <TableHead className="whitespace-nowrap">Email</TableHead>
                <TableHead className="whitespace-nowrap">SĐT</TableHead>
                <TableHead className="whitespace-nowrap">Số tài khoản</TableHead>
                <TableHead className="whitespace-nowrap">Tên tài khoản</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {lenders.map((lender) => (
                <TableRow
                  key={lender.id}
                  className="cursor-pointer"
                  onClick={() => setEditingLenderId(lender.id)}
                >
                  <TableCell className="align-top text-sm font-medium text-fg-primary">
                    {lender.name}
                  </TableCell>
                  <TableCell className="align-top text-sm tabular-nums text-fg-secondary">
                    {lender.cccd || "—"}
                  </TableCell>
                  <TableCell className="align-top text-sm text-fg-secondary">
                    {lender.email || "—"}
                  </TableCell>
                  <TableCell className="align-top text-sm tabular-nums text-fg-secondary">
                    {lender.mobile || "—"}
                  </TableCell>
                  <TableCell className="align-top text-sm tabular-nums text-fg-secondary">
                    {lender.bank_account_number || "—"}
                  </TableCell>
                  <TableCell className="align-top text-sm text-fg-secondary">
                    {lender.bank_account_name || "—"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}

        {pagination && pagination.totalPages > 1 && (
          <PaginationControls
            pagination={pagination}
            onPageChange={handlePageChange}
            onPageSizeChange={handlePageSizeChange}
            className="border-t pt-3"
          />
        )}
      </AdminSectionCard>

      <AddLenderSheet isOpen={showAddSheet} onClose={() => setShowAddSheet(false)} />
      {editingLenderId && (
        <EditLenderSheet
          isOpen={!!editingLenderId}
          onClose={() => setEditingLenderId(null)}
          lenderId={editingLenderId}
        />
      )}
    </AdminPageCanvas>
  );
};

export default LendersPage;
