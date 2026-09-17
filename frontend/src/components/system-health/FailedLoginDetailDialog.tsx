import React, { useMemo } from "react";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetClose,
} from "@/components/ui/sheet";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ShieldAlert, Globe, Monitor, Clock, MapPin, X } from "lucide-react";
import { useFailedLoginsByIdentifier } from "@/hooks/api/useSystemHealth";
import { cn } from "@/lib/utils";
import type { FailedLoginAttempt } from "@/types/api/system-health.types";

function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleString("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function ReasonBadge({ reason }: { reason: string }) {
  if (reason.includes("sai mật khẩu")) {
    return <Badge variant="warning" className="text-[11px] shrink-0">Sai mật khẩu</Badge>;
  }
  if (reason.includes("không tìm thấy")) {
    return <Badge variant="destructive" className="text-[11px] shrink-0">Không tìm thấy</Badge>;
  }
  return <Badge variant="secondary" className="text-[11px] shrink-0">{reason}</Badge>;
}

function AttemptCard({ attempt, index }: { attempt: FailedLoginAttempt; index: number }) {
  const locationStr = attempt.location
    ? [attempt.location.city, attempt.location.region, attempt.location.country]
        .filter(Boolean)
        .join(", ")
    : null;

  return (
    <div className="rounded-lg border bg-card p-3 space-y-2.5">
      {/* Header row: index + reason + name / time */}
      <div className="flex flex-wrap items-start justify-between gap-x-2 gap-y-1">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-[11px] font-mono text-muted-foreground shrink-0">
            #{index + 1}
          </span>
          <ReasonBadge reason={attempt.reason} />
          {attempt.user_name && (
            <span className="text-xs text-muted-foreground">
              {attempt.user_name}
            </span>
          )}
        </div>
        <div className="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
          <Clock className="h-3 w-3" />
          {formatTime(attempt.created_at)}
        </div>
      </div>

      {/* Detail rows */}
      <div className="space-y-1.5 text-sm">
        {/* IP Address */}
        <div className="flex items-center gap-2">
          <Globe className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          <span
            className={cn(
              "font-mono font-semibold",
              attempt.ip_address ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {attempt.ip_address || "—"}
          </span>
        </div>

        {/* Location (if resolved) */}
        {locationStr && (
          <div className="flex items-center gap-2">
            <MapPin className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
            <span className="text-muted-foreground">{locationStr}</span>
          </div>
        )}

        {/* Browser + Platform */}
        {(attempt.browser || attempt.platform) && (
          <div className="flex items-center gap-2">
            <Monitor className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
            <span className="text-muted-foreground">
              {[attempt.browser, attempt.platform].filter(Boolean).join(" · ")}
            </span>
          </div>
        )}
      </div>
    </div>
  );
}

interface FailedLoginDetailDialogProps {
  identifier: string | null;
  days: number;
  onClose: () => void;
}

export function FailedLoginDetailDialog({ identifier, days, onClose }: FailedLoginDetailDialogProps) {
  const { data: attempts, isLoading } = useFailedLoginsByIdentifier(
    identifier ?? "",
    days,
    !!identifier,
  );

  const sortedAttempts = useMemo(() => {
    if (!attempts) return [];
    return [...attempts].sort(
      (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime(),
    );
  }, [attempts]);

  return (
    <Sheet open={!!identifier} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full sm:w-[480px] sm:max-w-[480px] flex flex-col p-0">
        <SheetHeader
          className="flex-shrink-0 px-4 py-3 border-b"
          style={{ paddingTop: "max(12px, calc(12px + env(safe-area-inset-top)))" }}
        >
          <div className="flex items-start justify-between gap-2">
            <div className="min-w-0">
              <SheetTitle className="flex items-center gap-2">
                <ShieldAlert className="h-4 w-4 text-destructive shrink-0" />
                Chi tiết đăng nhập thất bại
              </SheetTitle>
              {identifier && (
                <p className="text-sm text-muted-foreground font-mono mt-1 break-all">
                  {identifier}
                </p>
              )}
            </div>
            <SheetClose asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 rounded-full shrink-0 text-muted-foreground hover:text-foreground"
                aria-label="Đóng"
              >
                <X className="h-4 w-4" />
              </Button>
            </SheetClose>
          </div>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-4 py-4" style={{ paddingBottom: "max(16px, calc(16px + env(safe-area-inset-bottom)))" }}>
          {isLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map((i) => (
                <div key={i} className="h-24 rounded-lg bg-muted animate-pulse" />
              ))}
            </div>
          ) : sortedAttempts.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4">
              Không có dữ liệu
            </p>
          ) : (
            <div className="space-y-2">
              <p className="text-xs text-muted-foreground mb-3">
                {sortedAttempts.length} lần thử
              </p>
              {sortedAttempts.map((attempt, i) => (
                <AttemptCard key={attempt.id} attempt={attempt} index={i} />
              ))}
            </div>
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
