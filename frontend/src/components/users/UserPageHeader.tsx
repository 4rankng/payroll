import { PageHeader } from "@/components/shared/PageHeader";
import { Plus, UserCog } from "lucide-react";

interface UserPageHeaderProps {
  onAddUser: () => void;
}

export const UserPageHeader = ({ onAddUser }: UserPageHeaderProps) => {
  return (
    <PageHeader
      title="Người dùng"
      description="Quản lý tài khoản người dùng và phân quyền truy cập hệ thống"
      icon={UserCog}
      actions={[
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
