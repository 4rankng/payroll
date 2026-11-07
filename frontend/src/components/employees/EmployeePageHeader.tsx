import { PageHeader } from '@/components/shared/PageHeader';
import { getConsoleKicker } from '@/lib/console';
import { useLocation } from 'react-router-dom';
import { Plus, Download, PhoneOff, PhoneIncoming } from 'lucide-react';
import { useAuth } from '@/contexts';

interface EmployeePageHeaderProps {
  totalEmployees: number;
  onAddEmployeeClick: () => void;
  onExportClick?: () => void;
  onExportNoMobileClick?: () => void;
  onImportMobilesClick?: () => void;
  onSearchFocus?: () => void;
  isExporting?: boolean;
}

export const EmployeePageHeader = ({
  totalEmployees,
  onAddEmployeeClick,
  onExportClick,
  onExportNoMobileClick,
  onImportMobilesClick,
  isExporting = false,
}: EmployeePageHeaderProps) => {
  const { user } = useAuth();
  const { pathname } = useLocation();

  return (
    <PageHeader
      kicker={getConsoleKicker(pathname)}
      title="Nhân viên"
      meta={`${totalEmployees.toLocaleString('vi-VN')} nhân viên`}
      actions={[
        ...(onExportClick && (user?.role === 'admin' || user?.role === 'partner') ? [{
          label: isExporting ? 'Đang xuất...' : 'Xuất Excel',
          onClick: onExportClick,
          icon: Download,
          variant: 'outline' as const,
          className: isExporting ? 'opacity-50 pointer-events-none' : '',
        }] : []),
        ...(onExportNoMobileClick && user?.role === 'admin' ? [{
          label: isExporting ? 'Đang xuất...' : 'NV chưa có SĐT',
          onClick: onExportNoMobileClick,
          icon: PhoneOff,
          variant: 'outline' as const,
          className: isExporting ? 'opacity-50 pointer-events-none' : '',
        }] : []),
        ...(onImportMobilesClick && user?.role === 'admin' ? [{
          label: 'Nhập SĐT',
          onClick: onImportMobilesClick,
          icon: PhoneIncoming,
          variant: 'outline' as const,
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
