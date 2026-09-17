import React from "react";
import { UserAvatar } from "@/components/ui/user-avatar";
import { EmptyState } from "@/components/shared/EmptyState";
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
    sub: "text-red-700",
  },
  partner: {
    label: "Quản lý",
    dot: "bg-blue-500",
    card: "border-blue-200 bg-blue-50/60",
    name: "text-blue-900",
    sub: "text-blue-700",
  },
  adv_partner: {
    label: "Quản lý ứng lương",
    dot: "bg-violet-500",
    card: "border-violet-200 bg-violet-50/60",
    name: "text-violet-900",
    sub: "text-violet-700",
  },
  accountant: {
    label: "Kế toán",
    dot: "bg-amber-500",
    card: "border-amber-200 bg-amber-50/60",
    name: "text-amber-900",
    sub: "text-amber-800",
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
        <p className={`min-h-8 break-words text-xs font-semibold leading-4 line-clamp-2 ${cfg.name}`}>{user.fullname}</p>
        <p className={`text-[11px] break-all line-clamp-2 mt-0.5 ${cfg.sub}`}>@{user.username}</p>
        <p className={`mt-1 text-[10px] font-medium leading-4 ${cfg.sub}`}>{cfg.label}</p>
      </div>
    </button>
  );
});

export function UserMobileList({ users, onRowClick, emptyState }: UserMobileListProps) {
  if (users.length === 0) {
    return (
      emptyState || <EmptyState title="Không tìm thấy người dùng nào" description="Không có dữ liệu để hiển thị." size="sm" />
    );
  }

  return (
    <div className="w-full space-y-3">
      {/* Grid */}
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
        {users.map((user) => (
          <UserGridCard key={user.id} user={user} onRowClick={onRowClick} />
        ))}
      </div>
    </div>
  );
}
