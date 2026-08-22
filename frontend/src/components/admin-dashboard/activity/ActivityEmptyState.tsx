import React from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { EmptyState } from '@/components/shared/EmptyState';

interface ActivityEmptyStateProps {
  type: 'loading' | 'error' | 'empty';
  error?: string;
  onRefresh?: () => void;
}

export const ActivityEmptyState: React.FC<ActivityEmptyStateProps> = ({ 
  type, 
  error, 
  onRefresh 
}) => {
  if (type === 'loading') {
    return (
      <div className="space-y-4">
        {Array.from({ length: 5 }, (_, index) => (
          <div key={index} className="flex gap-3">
            <Skeleton className="h-10 w-10 rounded-full flex-shrink-0" />
            <div className="flex-1 space-y-2 pt-1">
              <div className="flex items-center justify-between">
                <Skeleton className="h-4 w-2/3" />
                <Skeleton className="h-3 w-16" />
              </div>
              <Skeleton className="h-3 w-full" />
              <Skeleton className="h-3 w-1/2" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (type === 'error') {
    return (
      <div className="flex flex-col items-center justify-center py-12 space-y-4">
        <div className="p-3 rounded-full bg-red-50">
          <AlertCircle className="h-6 w-6 text-red-500" />
        </div>
        <div className="text-center space-y-2">
          <h4 className="text-sm font-medium text-foreground">
            Không thể tải dữ liệu
          </h4>
          <p className="text-xs text-muted-foreground max-w-xs">
            {error || 'Đã xảy ra lỗi khi tải danh sách hoạt động'}
          </p>
          {onRefresh && (
            <Button
              variant="outline"
              size="sm"
              onClick={onRefresh}
              className="mt-3"
            >
              <RefreshCw className="h-3 w-3 mr-1" />
              Thử lại
            </Button>
          )}
        </div>
      </div>
    );
  }

  // Empty state
  return (
    <EmptyState
      title="Chưa có hoạt động nào"
      description="Chưa có hoạt động nào được ghi nhận gần đây"
      action={onRefresh ? { label: 'Làm mới', onClick: onRefresh } : undefined}
      size="sm"
    />
  );
};
