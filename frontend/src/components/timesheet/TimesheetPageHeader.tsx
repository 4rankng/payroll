import { PageHeader } from '@/components/shared/PageHeader';
import { EditRequestBadge } from './EditRequestBadge';
import { Plus } from 'lucide-react';

interface TimesheetPageHeaderProps {
  onEditRequestClick?: () => void;
  onAddTimesheet?: () => void;
  userRole?: 'admin' | 'partner';
}

export const TimesheetPageHeader = ({ onEditRequestClick, onAddTimesheet, userRole = 'admin' }: TimesheetPageHeaderProps) => {
  const description = userRole === 'partner'
    ? 'Theo dõi và quản lý bảng công'
    : 'Theo dõi và duyệt bảng công';

  return (
    <PageHeader
      title="Bảng công"
      description={description}
      actions={onAddTimesheet ? [{ label: 'Nhập công', onClick: onAddTimesheet, icon: Plus }] : []}
    >
      {onEditRequestClick && (
        <div className="flex items-center">
          <EditRequestBadge onClick={onEditRequestClick} />
        </div>
      )}
    </PageHeader>
  );
};
