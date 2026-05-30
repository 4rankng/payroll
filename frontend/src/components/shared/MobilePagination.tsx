import { memo } from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { cn } from '@/lib/utils';

export interface PaginationInfo {
  page: number;
  pageSize: number;
  totalPages: number;
  totalRecords: number;
}

export interface MobilePaginationProps {
  pagination: PaginationInfo;
  onPageChange: (page: number) => void;
  className?: string;
}

/**
 * Mobile-optimised prev/next pagination with record range label.
 * Implements the duplicated pagination pattern from Requirement 16.
 */
export const MobilePagination = memo(function MobilePagination({
  pagination,
  onPageChange,
  className,
}: MobilePaginationProps) {
  if (pagination.totalPages <= 1) return null;

  const start = (pagination.page - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.page * pagination.pageSize, pagination.totalRecords);

  return (
    <div
      className={cn(
        'flex items-center justify-between pt-4 mt-1 border-t border-border',
        className,
      )}
    >
      <span className="text-xs text-muted-foreground tabular-nums">
        {start}–{end} / {pagination.totalRecords}
      </span>
      <div className="flex items-center gap-1.5">
        <button
          onClick={() => onPageChange(Math.max(1, pagination.page - 1))}
          disabled={pagination.page <= 1}
          className="h-8 w-8 flex items-center justify-center rounded-xl border border-border disabled:opacity-40 touch-manipulation active:bg-muted"
          aria-label="Trang trước"
        >
          <ChevronLeft className="h-4 w-4" />
        </button>
        <span className="text-xs font-medium px-1 tabular-nums text-muted-foreground">
          {pagination.page}/{pagination.totalPages}
        </span>
        <button
          onClick={() => onPageChange(Math.min(pagination.totalPages, pagination.page + 1))}
          disabled={pagination.page >= pagination.totalPages}
          className="h-8 w-8 flex items-center justify-center rounded-xl border border-border disabled:opacity-40 touch-manipulation active:bg-muted"
          aria-label="Trang tiếp"
        >
          <ChevronRight className="h-4 w-4" />
        </button>
      </div>
    </div>
  );
});
