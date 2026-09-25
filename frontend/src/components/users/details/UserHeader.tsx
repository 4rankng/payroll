import { Badge } from "@/components/ui/badge";
import { UserAvatar } from "@/components/ui/user-avatar";
import { getRoleColor } from "@/utils/userHelpers";
import { cn } from "@/lib/utils";
import type { User } from "@/types/user";

interface UserHeaderProps {
  user: User;
  showName?: boolean;
}

const ROLE_LABELS: Record<string, string> = {
  admin: "Quản trị viên",
  partner: "Quản lý",
  employee: "Nhân viên",
};

export function UserHeader({ user, showName = true }: UserHeaderProps) {
  const roleLabel = ROLE_LABELS[user.role] ?? "Người dùng";
  const roleColor = getRoleColor(user.role);

  return (
    <div className="flex items-center gap-3 w-full min-w-0">
      <UserAvatar
        name={user.fullname}
        username={user.username}
        size="lg"
        className="flex-shrink-0 ring-2 ring-background shadow-sm"
      />
      <div className="flex-1 min-w-0">
        {showName && (
          <div className="flex items-center gap-2 flex-wrap mb-0.5">
            <span className="min-w-0 text-sm font-semibold leading-tight break-words">
              {user.fullname}
            </span>
            <Badge className={cn(roleColor, "text-xs shrink-0")} variant="secondary">
              {roleLabel}
            </Badge>
          </div>
        )}
        <p className="text-xs text-muted-foreground break-all">@{user.username}</p>
        {user.email && (
          <p className="hidden text-xs text-muted-foreground break-all sm:block">{user.email}</p>
        )}
        {(user.role === "admin" || user.role === "partner") && user.mobile && (
          <p className="hidden text-xs text-muted-foreground break-all sm:block">SĐT: {user.mobile}</p>
        )}
      </div>
    </div>
  );
}
