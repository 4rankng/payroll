import { ColumnDef } from "@tanstack/react-table";
import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";
import { User } from "@/types/user";
import { formatVietnameseDate } from "@/utils/vietnamese";

interface CreateUserColumnsProps {
  // No action handlers needed since we'll use row clicks
}

export const createUserColumns = (
  _props: CreateUserColumnsProps = {},
): ColumnDef<User>[] => [
  {
    id: "stt",
    header: "STT",
    size: 40,
    cell: ({ row }) => {
      return (
        <div className="text-center typography-label-medium">
          {row.index + 1}
        </div>
      );
    },
  },
  {
    accessorKey: "username",
    header: "Người dùng",
    cell: ({ row }) => {
      const user = row.original;
      return (
        <div className="flex items-center gap-3">
          <UserAvatar size="sm" name={user.fullname} username={user.username} />
          <div>
            <div className="typography-label-medium">{user.fullname}</div>
            <div className="typography-body-medium text-muted-foreground">
              @{user.username}
            </div>
            <div className="typography-body-small text-muted-foreground">
              {user.email}
            </div>
          </div>
        </div>
      );
    },
  },
  {
    accessorKey: "role",
    header: "Vai trò",
    cell: ({ row }) => {
      const role = row.getValue("role") as string;
      let roleLabel = "Người dùng";
      let variant: "default" | "secondary" | "destructive" | "outline" = "secondary";

      if (role === "admin") {
        roleLabel = "Quản trị viên";
        variant = "destructive";
      } else if (role === "partner") {
        roleLabel = "Quản lý";
        variant = "default";
      } else if (role === "employee") {
        roleLabel = "Nhân viên";
        variant = "outline";
      } else if (role === "adv_partner") {
        roleLabel = "Quản lý ứng lương";
        variant = "outline";
      } else if (role === "accountant") {
        roleLabel = "Kế toán";
        variant = "outline";
      }

      return <Badge variant={variant}>{roleLabel}</Badge>;
    },
  },
  {
    accessorKey: "last_login",
    header: "Hoạt động cuối",
    cell: ({ row }) => {
      const lastLogin = row.getValue("last_login") as string | undefined;
      return (
        <div className="typography-body-medium">
          {lastLogin
            ? formatVietnameseDate(lastLogin, "dd/MM/yyyy HH:mm")
            : "Chưa đăng nhập"}
        </div>
      );
    },
  },
  {
    accessorKey: "created_at",
    header: "Ngày tạo",
    cell: ({ row }) => {
      const createdAt = row.getValue("created_at") as string;
      return (
        <div className="typography-body-medium">
          {formatVietnameseDate(createdAt)}
        </div>
      );
    },
  },
];
