import { PageHeader } from "@/components/shared/PageHeader";
import { getConsoleKicker } from "@/lib/console";
import { useLocation } from "react-router-dom";
import { Download, Plus } from "lucide-react";

interface UserPageHeaderProps {
  onAddUser: () => void;
  onExportUsers: () => void;
  isExporting?: boolean;
  /** Drives the console meta line (exact user count). */
  totalUsers?: number;
}

export const UserPageHeader = ({ onAddUser, onExportUsers, isExporting = false, totalUsers }: UserPageHeaderProps) => {
  const { pathname } = useLocation();

  return (
    <PageHeader
      kicker={getConsoleKicker(pathname)}
      title="Người dùng"
      meta={totalUsers !== undefined ? `${totalUsers.toLocaleString("vi-VN")} người dùng` : undefined}
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
