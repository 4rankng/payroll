import { TableLoadingSkeleton } from '@/components/ui/loading-states';

interface InfiniteScrollInfoProps {
  total: number;
  visible: number;
  loading: boolean;
}

export const InfiniteScrollInfo = ({ total, visible, loading }: InfiniteScrollInfoProps) => {
  if (total === 0) return null;
  return (
    <div className="flex flex-col items-center gap-2 p-4 sm:p-6 border-t">
      <div className="typography-body-medium text-muted-foreground">
        Hiển thị 1 đến {Math.min(visible, total)} trong tổng số {total} nhân viên
      </div>
      {loading && (
        <div className="py-2">
          <TableLoadingSkeleton rows={3} columns={6} />
        </div>
      )}
    </div>
  );
};
