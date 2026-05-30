import { FileEdit } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useEditRequests } from '@/hooks/api/useTimesheetEditRequests';

interface EditRequestBadgeProps {
  onClick: () => void;
}

export const EditRequestBadge = ({ onClick }: EditRequestBadgeProps) => {
  const { data: editRequests, isLoading } = useEditRequests({
    status: 'pending',
    page: 1,
    pageSize: 100
  });

  const pendingCount = editRequests?.pagination?.totalRecords ?? 0;

  if (isLoading) {
    return null;
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={onClick}
      className="relative"
    >
      <FileEdit className="h-4 w-4 mr-2" />
      <span className="hidden sm:inline">Yêu cầu sửa</span>
      {pendingCount > 0 && (
        <>
          <Badge
            variant="destructive"
            className="ml-2 h-5 min-w-5 px-1.5"
          >
            {pendingCount}
          </Badge>
          {/* Notification dot */}
          <span className="absolute -top-1 -right-1 flex h-3 w-3">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-destructive opacity-75"></span>
            <span className="relative inline-flex rounded-full h-3 w-3 bg-destructive"></span>
          </span>
        </>
      )}
    </Button>
  );
};
