import { format } from "date-fns";
import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";
import { User } from "@/types/user";
import { MobileField } from "@/components/ui/responsive-table";
interface CreateUserMobileConfigProps {
  // No action handlers needed since we'll use row clicks
}
export const createUserMobileConfig = (
  _props: CreateUserMobileConfigProps = {},
) => {
  const mobileFields: MobileField<User>[] = [
    {
      key: "role",
      label: "Vai trò",
      priority: 1,
      render: (user) => {
        let roleLabel = "Người dùng";
        let variant: "default" | "secondary" | "destructive" | "outline" = "secondary";

        if (user.role === "admin") {
          roleLabel = "Quản trị viên";
          variant = "destructive";
        } else if (user.role === "partner") {
          roleLabel = "Quản lý";
          variant = "default";
        } else if (user.role === "employee") {
          roleLabel = "Nhân viên";
          variant = "outline";
        }

        return <Badge variant={variant}> {roleLabel} </Badge>;
      },
    },
    {
      key: "last_login",
      label: "Hoạt động cuối",
      priority: 2,
      render: (user) => (
        <div className="typography-body-medium text-muted-foreground">
          {" "}
          {user.last_login
            ? format(new Date(user.last_login), 'dd/MM/yyyy')
            : "Chưa đăng nhập"}{" "}
        </div>
      ),
    },
    {
      key: "created_at",
      label: "Ngày tạo",
      priority: 2,
      render: (user) => (
        <div className="typography-body-medium text-muted-foreground">
          {" "}
          {format(new Date(user.created_at), 'dd/MM/yyyy')}{" "}
        </div>
      ),
    },
  ];
  const rowTitle = (user: User) => (
    <div className="flex items-center gap-3">
      <UserAvatar name={user.fullname} username={user.username} size="lg" />
      <div className="min-w-0 flex-1">
        <p className="typography-body-large truncate">{user.fullname}</p>
        <p className="typography-body-medium text-muted-foreground truncate">{user.email}</p>
      </div>
    </div>
  );
  const rowSubtitle = (user: User) => (
    <div className="typography-body-medium text-muted-foreground">
      {" "}
      @{user.username}{" "}
    </div>
  );
  return { mobileFields, rowTitle, rowSubtitle };
};
