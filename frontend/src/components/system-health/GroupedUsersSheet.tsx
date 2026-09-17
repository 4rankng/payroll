import React, { useMemo } from "react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetClose,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Users, Zap, Clock, X } from "lucide-react";
import { useOSUsers, useBrowserUsers } from "@/hooks/api/useSystemHealth";
import type { BrowserPlatformUser } from "@/types/api/system-health.types";

function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const ROLE_LABELS: Record<string, string> = {
  admin: "Quản trị",
  partner: "Đối tác",
  employee: "Nhân viên",
  adv_partner: "Quản lý ứng lương",
};

const ROLE_VARIANTS: Record<string, "default" | "secondary" | "outline" | "destructive" | "warning"> = {
  admin: "destructive",
  partner: "warning",
  employee: "secondary",
  adv_partner: "outline",
};

function UserCard({ user, index }: { user: BrowserPlatformUser; index: number }) {
  return (
    <div className="rounded-lg border bg-card p-3 space-y-2">
      <div className="flex flex-wrap items-start gap-2">
        <span className="text-[11px] font-mono text-muted-foreground shrink-0">
          #{index + 1}
        </span>
        <span className="min-w-0 flex-1 break-words text-sm font-semibold text-foreground">
          {user.fullname || user.username}
        </span>
        <Badge
          variant={ROLE_VARIANTS[user.role] ?? "secondary"}
          className="text-[11px] shrink-0"
        >
          {ROLE_LABELS[user.role] ?? user.role}
        </Badge>
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span className="min-w-0 break-all font-mono">{user.username}</span>
        <span className="flex shrink-0 items-center gap-1">
          <Zap className="h-3 w-3" />
          {user.actions.toLocaleString()}
        </span>
        <span className="flex min-w-0 items-center gap-1">
          <Clock className="h-3 w-3" />
          <span className="break-words">{formatTime(user.last_seen)}</span>
        </span>
      </div>
    </div>
  );
}

// ─── OS users sheet ───────────────────────────────────────────────────────────

interface OSUsersSheetProps {
  osFamily: string | null;
  days: number;
  onClose: () => void;
}

export function OSUsersSheet({ osFamily, days, onClose }: OSUsersSheetProps) {
  const open = !!osFamily;
  const { data: users, isLoading } = useOSUsers(osFamily ?? "", days, open);
  const sorted = useMemo(() => (users ? [...users] : []), [users]);
  const totalActions = sorted.reduce((sum, u) => sum + u.actions, 0);

  return (
    <Sheet open={open} onOpenChange={(o) => !o && onClose()}>
      <SheetContent className="flex w-full flex-col p-0 sm:w-[480px] sm:max-w-[480px]">
        <SheetHeader
          className="flex-shrink-0 px-4 py-3 border-b"
          style={{ paddingTop: "max(12px, calc(12px + env(safe-area-inset-top)))" }}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <SheetTitle className="flex min-w-0 items-center gap-2">
                <Users className="h-4 w-4 text-primary shrink-0" />
                <span className="min-w-0 break-words">Người dùng · {osFamily}</span>
              </SheetTitle>
              <p className="mt-1 break-words text-sm text-muted-foreground">Tất cả phiên bản {osFamily}</p>
            </div>
            <SheetClose asChild>
              <Button variant="ghost" size="icon" className="h-11 w-11 shrink-0 rounded-full text-muted-foreground hover:text-foreground" aria-label="Đóng">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>
        <div className="flex-1 overflow-y-auto px-4 py-4" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
          {isLoading ? (
            <div className="space-y-3">{[1, 2, 3].map((i) => <div key={i} className="h-16 rounded-lg bg-muted animate-pulse" />)}</div>
          ) : sorted.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4">Không có dữ liệu</p>
          ) : (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground mb-3">
                {sorted.length} người dùng · {totalActions.toLocaleString()} hành động
              </p>
              {sorted.map((user, i) => <UserCard key={user.user_id} user={user} index={i} />)}
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}

// ─── Browser users sheet ──────────────────────────────────────────────────────

interface BrowserUsersSheetProps {
  browserFamily: string | null;
  days: number;
  onClose: () => void;
}

export function BrowserUsersSheet({ browserFamily, days, onClose }: BrowserUsersSheetProps) {
  const open = !!browserFamily;
  const { data: users, isLoading } = useBrowserUsers(browserFamily ?? "", days, open);
  const sorted = useMemo(() => (users ? [...users] : []), [users]);
  const totalActions = sorted.reduce((sum, u) => sum + u.actions, 0);

  return (
    <Sheet open={open} onOpenChange={(o) => !o && onClose()}>
      <SheetContent className="flex w-full flex-col p-0 sm:w-[480px] sm:max-w-[480px]">
        <SheetHeader
          className="flex-shrink-0 px-4 py-3 border-b"
          style={{ paddingTop: "max(12px, calc(12px + env(safe-area-inset-top)))" }}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <SheetTitle className="flex min-w-0 items-center gap-2">
                <Users className="h-4 w-4 text-primary shrink-0" />
                <span className="min-w-0 break-words">Người dùng · {browserFamily}</span>
              </SheetTitle>
              <p className="mt-1 break-words text-sm text-muted-foreground">Tất cả phiên bản {browserFamily}</p>
            </div>
            <SheetClose asChild>
              <Button variant="ghost" size="icon" className="h-11 w-11 shrink-0 rounded-full text-muted-foreground hover:text-foreground" aria-label="Đóng">
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>
        <div className="flex-1 overflow-y-auto px-4 py-4" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
          {isLoading ? (
            <div className="space-y-3">{[1, 2, 3].map((i) => <div key={i} className="h-16 rounded-lg bg-muted animate-pulse" />)}</div>
          ) : sorted.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4">Không có dữ liệu</p>
          ) : (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground mb-3">
                {sorted.length} người dùng · {totalActions.toLocaleString()} hành động
              </p>
              {sorted.map((user, i) => <UserCard key={user.user_id} user={user} index={i} />)}
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
