import { memo } from 'react';
import { CalendarPlus, FilterX, Plus, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';

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
      <div className="rounded-2xl border border-dashed border-border bg-muted/20 p-10 flex flex-col items-center text-center gap-3">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
          <CalendarPlus className="h-6 w-6 text-muted-foreground" />
        </div>
        <div className="space-y-1">
          <h3 className="text-base font-semibold text-foreground">Chưa có dữ liệu chấm công cho kỳ này</h3>
          <p className="text-sm text-muted-foreground max-w-md">
            Bắt đầu bằng cách nhập công cho nhân viên trong kỳ.
          </p>
        </div>
        {onAddTimesheet && (
          <Button size="sm" onClick={onAddTimesheet}>
            <Plus className="h-4 w-4 mr-1.5" />
            Nhập công
          </Button>
        )}
      </div>
    );
  }

  if (variant === 'no-filter-results') {
    return (
      <div className="rounded-2xl border border-dashed border-border bg-muted/20 p-10 flex flex-col items-center text-center gap-3">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-muted">
          <FilterX className="h-6 w-6 text-muted-foreground" />
        </div>
        <div className="space-y-1">
          <h3 className="text-base font-semibold text-foreground">Không tìm thấy kết quả</h3>
          <p className="text-sm text-muted-foreground max-w-md">
            Thử điều chỉnh hoặc xoá bộ lọc để xem thêm kết quả.
          </p>
        </div>
        {onClearFilters && (
          <Button size="sm" variant="outline" onClick={onClearFilters}>
            <RotateCcw className="h-4 w-4 mr-1.5" />
            Xoá bộ lọc
          </Button>
        )}
      </div>
    );
  }

  return null;
});
