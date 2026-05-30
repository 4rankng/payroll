import React from "react";
import { Users } from "lucide-react";
import { UserAvatar } from "@/components/ui/user-avatar";
import type { User } from "@/types/user";

interface UserMobileListProps {
  users: User[];
  onRowClick?: (user: User) => void;
  emptyState?: React.ReactNode;
}

const ROLE_CONFIG = {
  admin: {
    label: "Quản trị viên",
    dot: "bg-red-500",
    card: "border-red-200 bg-red-50/60",
    name: "text-red-900",
    sub: "text-red-700/70",
  },
  partner: {
    label: "Quản lý",
    dot: "bg-blue-500",
    card: "border-blue-200 bg-blue-50/60",
    name: "text-blue-900",
    sub: "text-blue-700/70",
  },
  employee: {
    label: "Nhân viên",
    dot: "bg-gray-400",
    card: "border-border/60 bg-card",
    name: "text-foreground",
    sub: "text-muted-foreground",
  },
} as const;

type Role = keyof typeof ROLE_CONFIG;

const UserGridCard = React.memo(function UserGridCard({
  user,
  onRowClick,
}: {
  user: User;
  onRowClick?: (user: User) => void;
}) {
  const cfg = ROLE_CONFIG[(user.role as Role)] ?? ROLE_CONFIG.employee;

  return (
    <button
      type="button"
      onClick={() => onRowClick?.(user)}
      className={`flex flex-col items-center gap-2 rounded-xl border p-3 text-center transition-all active:scale-95 active:brightness-95 touch-manipulation w-full ${cfg.card}`}
      aria-label={`Xem chi tiết ${user.fullname}`}
    >
      <div className="relative">
        <UserAvatar
          name={user.fullname}
          username={user.username}
          size="lg"
          className="ring-2 ring-background shadow-sm"
        />
        {/* Role dot */}
        <span className={`absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full ring-2 ring-background ${cfg.dot}`} />
      </div>
      <div className="w-full min-w-0">
        <p className={`text-xs font-semibold leading-tight truncate ${cfg.name}`}>{user.fullname}</p>
        <p className={`text-[10px] truncate mt-0.5 ${cfg.sub}`}>@{user.username}</p>
      </div>
    </button>
  );
});

export function UserMobileList({ users, onRowClick, emptyState }: UserMobileListProps) {
  if (users.length === 0) {
    return (
      <div className="text-center py-12">
        {emptyState || (
          <>
            <Users className="mx-auto h-12 w-12 text-muted-foreground/50" />
            <h3 className="mt-4 text-base font-semibold">Không tìm thấy người dùng nào</h3>
            <p className="mt-1 text-sm text-muted-foreground">Không có dữ liệu để hiển thị.</p>
          </>
        )}
      </div>
    );
  }

  return (
    <div className="w-full space-y-3">
      {/* Grid */}
      <div className="grid grid-cols-2 gap-2">
        {users.map((user) => (
          <UserGridCard key={user.id} user={user} onRowClick={onRowClick} />
        ))}
      </div>
    </div>
  );
}
