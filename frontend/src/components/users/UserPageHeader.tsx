import { PageHeader } from "@/components/shared/PageHeader";
import { Download, Plus, UserCog } from "lucide-react";

interface UserPageHeaderProps {
  onAddUser: () => void;
  onExportUsers: () => void;
  isExporting?: boolean;
}

export const UserPageHeader = ({ onAddUser, onExportUsers, isExporting = false }: UserPageHeaderProps) => {
  return (
    <PageHeader
      title="Người dùng"
      description="Quản lý tài khoản người dùng và phân quyền truy cập hệ thống"
      icon={UserCog}
      actions={[
        {
          label: isExporting ? 'Đang xuất...' : 'Xuất Excel',
          onClick: onExportUsers,
          icon: Download,
          variant: 'outline' as const,
          className: isExporting ? 'opacity-50 pointer-events-none' : '',
        },
        {
          label: 'Thêm người dùng',
          onClick: onAddUser,
          icon: Plus,
          variant: 'default' as const,
        },
      ]}
    />
  );
};
