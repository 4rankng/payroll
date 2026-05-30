import { PageHeader } from '@/components/shared/PageHeader';
import { Plus, Download, Users } from 'lucide-react';
import { useAuth } from '@/contexts';

interface EmployeePageHeaderProps {
  totalEmployees: number;
  onAddEmployeeClick: () => void;
  onExportClick?: () => void;
  onSearchFocus?: () => void;
  isExporting?: boolean;
}

export const EmployeePageHeader = ({
  totalEmployees,
  onAddEmployeeClick,
  onExportClick,
  isExporting = false,
}: EmployeePageHeaderProps) => {
  const { user } = useAuth();

  return (
    <PageHeader
      title="Nhân viên"
      description={`Quản lý thông tin ${totalEmployees} nhân viên trong hệ thống`}
      icon={Users}
      actions={[
        ...(onExportClick && (user?.role === 'admin' || user?.role === 'partner') ? [{
          label: isExporting ? 'Đang xuất...' : 'Xuất Excel',
          onClick: onExportClick,
          icon: Download,
          variant: 'outline' as const,
          className: isExporting ? 'opacity-50 pointer-events-none' : '',
        }] : []),
        {
          label: 'Thêm nhân viên',
          onClick: onAddEmployeeClick,
          icon: Plus,
          variant: 'default' as const,
        },
      ]}
    />
  );
};
