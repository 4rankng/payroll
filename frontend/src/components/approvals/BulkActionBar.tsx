import { Card, CardContent } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { CheckCircle, XCircle } from 'lucide-react';

interface BulkActionBarProps {
  selectedCount: number;
  onBulkApprove: () => void;
  onBulkReject: () => void;
  onClearSelection: () => void;
  isLoading?: boolean;
}

export const BulkActionBar = ({
  selectedCount,
  onBulkApprove,
  onBulkReject,
  onClearSelection,
  isLoading
}: BulkActionBarProps) => {
  return (
    <Card className="border-border bg-muted/50">
      <CardContent className="py-3">
        <div className="flex items-center justify-between">
          <span className="typography-body-medium text-foreground">
            Đã chọn {selectedCount} yêu cầu
          </span>
          <div className="flex gap-2">
            <Button
              size="sm"
              onClick={onBulkApprove}
              disabled={isLoading}
            >
              <CheckCircle className="w-4 h-4 mr-1" />
              Duyệt
            </Button>
            <Button
              size="sm"
              variant="destructive"
              onClick={onBulkReject}
              disabled={isLoading}
            >
              <XCircle className="w-4 h-4 mr-1" />
              Loại
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={onClearSelection}
            >
              Hủy chọn
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
