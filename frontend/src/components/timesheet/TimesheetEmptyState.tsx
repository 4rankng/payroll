import { memo } from 'react';
import { Plus, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { EmptyState } from '@/components/shared/EmptyState';

export type TimesheetEmptyVariant = 'no-data' | 'no-filter-results';

interface TimesheetEmptyStateProps {
  variant: TimesheetEmptyVariant;
  onAddTimesheet?: () => void;
  onClearFilters?: () => void;
}

export const TimesheetEmptyState = memo(function TimesheetEmptyState({
  variant,
  onAddTimesheet,
  onClearFilters,
}: TimesheetEmptyStateProps) {
  if (variant === 'no-data') {
    return (
      <EmptyState
        title="Chưa có dữ liệu chấm công cho kỳ này"
        description="Bắt đầu bằng cách nhập công cho nhân viên trong kỳ."
        className="rounded-xl border border-dashed border-border bg-muted/20 px-4"
      >
        {onAddTimesheet && (
          <Button size="sm" onClick={onAddTimesheet}>
            <Plus className="h-4 w-4 mr-1.5" />
            Nhập công
          </Button>
        )}
      </EmptyState>
    );
  }

  if (variant === 'no-filter-results') {
    return (
      <EmptyState
        title="Không tìm thấy kết quả"
        description="Thử điều chỉnh hoặc xoá bộ lọc để xem thêm kết quả."
        className="rounded-xl border border-dashed border-border bg-muted/20 px-4"
      >
        {onClearFilters && (
          <Button size="sm" variant="outline" onClick={onClearFilters}>
            <RotateCcw className="h-4 w-4 mr-1.5" />
            Xoá bộ lọc
          </Button>
        )}
      </EmptyState>
    );
  }

  return null;
});
